package model

import (
	"errors"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"gorm.io/gorm"
)

// Checkin 签到记录
type Checkin struct {
	Id           int    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId       int    `json:"user_id" gorm:"not null;uniqueIndex:idx_user_checkin_date"`
	CheckinDate  string `json:"checkin_date" gorm:"type:varchar(10);not null;uniqueIndex:idx_user_checkin_date"` // 格式: YYYY-MM-DD
	QuotaAwarded int    `json:"quota_awarded" gorm:"not null"`
	CreatedAt    int64  `json:"created_at" gorm:"bigint"`
}

// CheckinRecord 用于API返回的签到记录（不包含敏感字段）
type CheckinRecord struct {
	CheckinDate  string `json:"checkin_date"`
	QuotaAwarded int    `json:"quota_awarded"`
}

func (Checkin) TableName() string {
	return "checkins"
}

// GetUserCheckinRecords 获取用户在指定日期范围内的签到记录
func GetUserCheckinRecords(userId int, startDate, endDate string) ([]Checkin, error) {
	var records []Checkin
	err := DB.Where("user_id = ? AND checkin_date >= ? AND checkin_date <= ?",
		userId, startDate, endDate).
		Order("checkin_date DESC").
		Find(&records).Error
	return records, err
}

// HasCheckedInToday 检查用户今天是否已签到
func HasCheckedInToday(userId int) (bool, error) {
	today := time.Now().Format("2006-01-02")
	var count int64
	err := DB.Model(&Checkin{}).
		Where("user_id = ? AND checkin_date = ?", userId, today).
		Count(&count).Error
	return count > 0, err
}

// UserCheckin 执行用户签到
// MySQL 和 PostgreSQL 使用事务保证原子性
// SQLite 不支持嵌套事务，使用顺序操作 + 手动回滚
func UserCheckin(userId int) (*Checkin, error) {
	// A consistent snapshot also prevents a concurrent settings save from
	// mixing old reward bounds with new tier thresholds.
	common.OptionMapRWMutex.RLock()
	setting := *operation_setting.GetCheckinSetting()
	common.OptionMapRWMutex.RUnlock()
	if !setting.Enabled {
		return nil, errors.New("签到功能未启用")
	}

	// 检查今天是否已签到
	hasChecked, err := HasCheckedInToday(userId)
	if err != nil {
		return nil, err
	}
	if hasChecked {
		return nil, errors.New("今日已签到")
	}

	if err := setting.Validate(); err != nil {
		return nil, err
	}
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var consumption struct {
		Requests int64
		Quota    int64
	}
	err = LOG_DB.Model(&Log{}).
		Where("user_id = ? AND type = ? AND created_at >= ? AND created_at < ?",
			userId, LogTypeConsume, todayStart.AddDate(0, 0, -1).Unix(), todayStart.Unix()).
		Select("COUNT(*) AS requests, COALESCE(SUM(quota), 0) AS quota").
		Scan(&consumption).Error
	if err != nil {
		return nil, err
	}
	if consumption.Requests < int64(setting.MinPreviousDayRequests) {
		return nil, errors.New("昨日请求次数不足，未达到签到要求")
	}
	if setting.MinSingleRedemptionQuota > 0 {
		var count int64
		err = DB.Unscoped().Model(&Redemption{}).
			Where("used_user_id = ? AND status = ? AND quota >= ?",
				userId, common.RedemptionCodeStatusUsed, setting.MinSingleRedemptionQuota).
			Count(&count).Error
		if err != nil {
			return nil, err
		}
		if count == 0 {
			return nil, errors.New("未使用过达到最低额度要求的兑换码")
		}
	}
	quotaAwarded, err := setting.Reward(consumption.Quota, rand.Intn)
	if err != nil {
		return nil, err
	}

	today := now.Format("2006-01-02")
	checkin := &Checkin{
		UserId:       userId,
		CheckinDate:  today,
		QuotaAwarded: quotaAwarded,
		CreatedAt:    now.Unix(),
	}

	// 根据数据库类型选择不同的策略
	if common.UsingMainDatabase(common.DatabaseTypeSQLite) {
		// SQLite 不支持嵌套事务，使用顺序操作 + 手动回滚
		return userCheckinWithoutTransaction(checkin, userId, quotaAwarded)
	}

	// MySQL 和 PostgreSQL 支持事务，使用事务保证原子性
	return userCheckinWithTransaction(checkin, userId, quotaAwarded)
}

var checkinSettingsMutex sync.Mutex

// Save the entire validated configuration in one transaction. Keep the
// existing dotted option keys so stored settings survive official updates.
func UpdateCheckinOptions(values map[string]string) error {
	checkinSettingsMutex.Lock()
	defer checkinSettingsMutex.Unlock()
	common.OptionMapRWMutex.RLock()
	next := *operation_setting.GetCheckinSetting()
	common.OptionMapRWMutex.RUnlock()
	for key, value := range values {
		switch key {
		case "checkin_setting":
			if err := common.UnmarshalJsonStr(value, &next); err != nil {
				return errors.New("签到设置格式无效")
			}
		case "checkin_setting.enabled":
			if value != "true" && value != "false" {
				return errors.New("签到开关必须为true或false")
			}
			next.Enabled = value == "true"
		default:
			// Decode each known field strictly; reject fractional numbers and
			// unknown option keys rather than silently truncating/ignoring them.
			field := strings.TrimPrefix(key, "checkin_setting.")
			switch field {
			case "min_quota", "max_quota", "min_previous_day_requests",
				"min_single_redemption_quota", "last_10_percent_consume_quota",
				"twenty_to_ten_percent_consume_quota":
				number, err := strconv.Atoi(value)
				if err != nil {
					return errors.New("签到额度和门槛必须为整数")
				}
				if err := common.UnmarshalJsonStr("{\""+field+"\":"+strconv.Itoa(number)+"}", &next); err != nil {
					return err
				}
			default:
				return errors.New("未知的签到配置项")
			}
		}
	}
	// Disabling remains possible even when legacy saved settings are invalid.
	if next.Enabled {
		if err := next.Validate(); err != nil {
			return err
		}
	}
	fields, err := config.ConfigToMap(next)
	if err != nil {
		return err
	}
	if err := DB.Transaction(func(tx *gorm.DB) error {
		for key, value := range fields {
			option := Option{Key: "checkin_setting." + key}
			if err := tx.FirstOrCreate(&option, Option{Key: option.Key}).Error; err != nil {
				return err
			}
			option.Value = value
			if err := tx.Save(&option).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	common.OptionMapRWMutex.Lock()
	defer common.OptionMapRWMutex.Unlock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	for key, value := range fields {
		common.OptionMap["checkin_setting."+key] = value
	}
	*operation_setting.GetCheckinSetting() = next
	return nil
}

// userCheckinWithTransaction 使用事务执行签到（适用于 MySQL 和 PostgreSQL）
func userCheckinWithTransaction(checkin *Checkin, userId int, quotaAwarded int) (*Checkin, error) {
	err := DB.Transaction(func(tx *gorm.DB) error {
		// 步骤1: 创建签到记录
		// 数据库有唯一约束 (user_id, checkin_date)，可以防止并发重复签到
		if err := tx.Create(checkin).Error; err != nil {
			return errors.New("签到失败，请稍后重试")
		}

		// 步骤2: 在事务中增加用户额度
		if err := tx.Model(&User{}).Where("id = ?", userId).
			Update("quota", gorm.Expr("quota + ?", quotaAwarded)).Error; err != nil {
			return errors.New("签到失败：更新额度出错")
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 事务成功后，异步更新缓存
	go func() {
		_ = cacheIncrUserQuota(userId, int64(quotaAwarded))
	}()

	return checkin, nil
}

// userCheckinWithoutTransaction 不使用事务执行签到（适用于 SQLite）
func userCheckinWithoutTransaction(checkin *Checkin, userId int, quotaAwarded int) (*Checkin, error) {
	// 步骤1: 创建签到记录
	// 数据库有唯一约束 (user_id, checkin_date)，可以防止并发重复签到
	if err := DB.Create(checkin).Error; err != nil {
		return nil, errors.New("签到失败，请稍后重试")
	}

	// 步骤2: 增加用户额度
	// 使用 db=true 强制直接写入数据库，不使用批量更新
	if err := IncreaseUserQuota(userId, quotaAwarded, true); err != nil {
		// 如果增加额度失败，需要回滚签到记录
		DB.Delete(checkin)
		return nil, errors.New("签到失败：更新额度出错")
	}

	return checkin, nil
}

// GetUserCheckinStats 获取用户签到统计信息
func GetUserCheckinStats(userId int, month string) (map[string]any, error) {
	// 获取指定月份的所有签到记录
	startDate := month + "-01"
	endDate := month + "-31"

	records, err := GetUserCheckinRecords(userId, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// 转换为不包含敏感字段的记录
	checkinRecords := make([]CheckinRecord, len(records))
	for i, r := range records {
		checkinRecords[i] = CheckinRecord{
			CheckinDate:  r.CheckinDate,
			QuotaAwarded: r.QuotaAwarded,
		}
	}

	// 检查今天是否已签到
	hasCheckedToday, _ := HasCheckedInToday(userId)

	// 获取用户所有时间的签到统计
	var totalCheckins int64
	var totalQuota int64
	DB.Model(&Checkin{}).Where("user_id = ?", userId).Count(&totalCheckins)
	DB.Model(&Checkin{}).Where("user_id = ?", userId).Select("COALESCE(SUM(quota_awarded), 0)").Scan(&totalQuota)

	return map[string]any{
		"total_quota":      totalQuota,      // 所有时间累计获得的额度
		"total_checkins":   totalCheckins,   // 所有时间累计签到次数
		"checkin_count":    len(records),    // 本月签到次数
		"checked_in_today": hasCheckedToday, // 今天是否已签到
		"records":          checkinRecords,  // 本月签到记录详情（不含id和user_id）
	}, nil
}
