package model

import (
	"errors"
	"math/rand"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"gorm.io/gorm"
)

type Checkin struct {
	Id           int    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId       int    `json:"user_id" gorm:"not null;uniqueIndex:idx_user_checkin_date"`
	CheckinDate  string `json:"checkin_date" gorm:"type:varchar(10);not null;uniqueIndex:idx_user_checkin_date"`
	QuotaAwarded int    `json:"quota_awarded" gorm:"not null"`
	CreatedAt    int64  `json:"created_at" gorm:"bigint"`
}

type CheckinRecord struct {
	CheckinDate  string `json:"checkin_date"`
	QuotaAwarded int    `json:"quota_awarded"`
}

func (Checkin) TableName() string {
	return "checkins"
}

func GetUserCheckinRecords(userId int, startDate, endDate string) ([]Checkin, error) {
	var records []Checkin
	err := DB.Where("user_id = ? AND checkin_date >= ? AND checkin_date <= ?",
		userId, startDate, endDate).
		Order("checkin_date DESC").
		Find(&records).Error
	return records, err
}

func HasCheckedInToday(userId int) (bool, error) {
	today := time.Now().Format("2006-01-02")
	var count int64
	err := DB.Model(&Checkin{}).
		Where("user_id = ? AND checkin_date = ?", userId, today).
		Count(&count).Error
	return count > 0, err
}

func getPreviousDayTimeRange() (int64, int64) {
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterdayStart := todayStart.AddDate(0, 0, -1)
	return yesterdayStart.Unix(), todayStart.Unix()
}

func GetPreviousDayConsumeRequestCount(userId int) (int64, error) {
	yesterdayStart, todayStart := getPreviousDayTimeRange()

	var count int64
	err := LOG_DB.Model(&Log{}).
		Where("user_id = ? AND type = ? AND created_at >= ? AND created_at < ?",
			userId, LogTypeConsume, yesterdayStart, todayStart).
		Count(&count).Error
	return count, err
}

func GetPreviousDayConsumeQuota(userId int) (int64, error) {
	yesterdayStart, todayStart := getPreviousDayTimeRange()

	var quota int64
	err := LOG_DB.Model(&Log{}).
		Where("user_id = ? AND type = ? AND created_at >= ? AND created_at < ?",
			userId, LogTypeConsume, yesterdayStart, todayStart).
		Select("COALESCE(SUM(quota), 0)").
		Scan(&quota).Error
	return quota, err
}

func HasUserRedeemedSingleQuotaAtLeast(userId int, minQuota int) (bool, error) {
	var count int64
	err := DB.Unscoped().Model(&Redemption{}).
		Where("used_user_id = ? AND status = ? AND quota >= ?", userId, common.RedemptionCodeStatusUsed, minQuota).
		Count(&count).Error
	return count > 0, err
}

func UserCheckin(userId int) (*Checkin, error) {
	setting := operation_setting.GetCheckinSetting()
	if !setting.Enabled {
		return nil, errors.New("签到功能未启用")
	}

	hasChecked, err := HasCheckedInToday(userId)
	if err != nil {
		return nil, err
	}
	if hasChecked {
		return nil, errors.New("今日已签到")
	}

	if setting.MinPreviousDayRequests > 0 {
		previousDayRequests, err := GetPreviousDayConsumeRequestCount(userId)
		if err != nil {
			return nil, err
		}
		if previousDayRequests < int64(setting.MinPreviousDayRequests) {
			return nil, errors.New("未达到签到要求")
		}
	}

	if setting.MinSingleRedemptionQuota > 0 {
		qualified, err := HasUserRedeemedSingleQuotaAtLeast(userId, setting.MinSingleRedemptionQuota)
		if err != nil {
			return nil, err
		}
		if !qualified {
			return nil, errors.New("未达到签到要求")
		}
	}

	previousDayQuota, err := GetPreviousDayConsumeQuota(userId)
	if err != nil {
		return nil, err
	}
	if previousDayQuota <= int64(setting.MinQuota) {
		return nil, errors.New("未达到签到要求")
	}

	minQuota := setting.MinQuota
	maxQuota := setting.MaxQuota
	previousDayQuotaLimit := int(previousDayQuota / 2)
	if previousDayQuotaLimit < maxQuota {
		maxQuota = previousDayQuotaLimit
	}
	if maxQuota < minQuota {
		return nil, errors.New("未达到签到要求")
	}
	last10PercentConsumeQuota := operation_setting.GetLast10PercentConsumeQuota()
	if setting.MaxQuota > 0 && last10PercentConsumeQuota > 0 &&
		previousDayQuota >= int64(last10PercentConsumeQuota) {
		highRangeMin := (setting.MaxQuota*9 + 9) / 10
		if highRangeMin > minQuota {
			minQuota = highRangeMin
		}
		maxQuota = setting.MaxQuota
	} else if setting.MaxQuota > 0 &&
		setting.TwentyToTenPercentConsumeQuota > 0 &&
		setting.TwentyToTenPercentConsumeQuota < last10PercentConsumeQuota &&
		previousDayQuota >= int64(setting.TwentyToTenPercentConsumeQuota) {
		midRangeMin := (setting.MaxQuota*8 + 9) / 10
		midRangeMax := (setting.MaxQuota*9+9)/10 - 1
		if midRangeMin < minQuota {
			midRangeMin = minQuota
		}
		if midRangeMax >= midRangeMin {
			minQuota = midRangeMin
			maxQuota = midRangeMax
		}
	}

	quotaAwarded := minQuota
	if maxQuota > minQuota {
		quotaAwarded = minQuota + rand.Intn(maxQuota-minQuota+1)
	}

	today := time.Now().Format("2006-01-02")
	checkin := &Checkin{
		UserId:       userId,
		CheckinDate:  today,
		QuotaAwarded: quotaAwarded,
		CreatedAt:    time.Now().Unix(),
	}

	if common.UsingSQLite {
		return userCheckinWithoutTransaction(checkin, userId, quotaAwarded)
	}

	return userCheckinWithTransaction(checkin, userId, quotaAwarded)
}

func userCheckinWithTransaction(checkin *Checkin, userId int, quotaAwarded int) (*Checkin, error) {
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(checkin).Error; err != nil {
			return errors.New("签到失败，请稍后重试")
		}

		if err := tx.Model(&User{}).Where("id = ?", userId).
			Update("quota", gorm.Expr("quota + ?", quotaAwarded)).Error; err != nil {
			return errors.New("签到失败：更新额度出错")
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	go func() {
		_ = cacheIncrUserQuota(userId, int64(quotaAwarded))
	}()

	return checkin, nil
}

func userCheckinWithoutTransaction(checkin *Checkin, userId int, quotaAwarded int) (*Checkin, error) {
	if err := DB.Create(checkin).Error; err != nil {
		return nil, errors.New("签到失败，请稍后重试")
	}

	if err := IncreaseUserQuota(userId, quotaAwarded, true); err != nil {
		DB.Delete(checkin)
		return nil, errors.New("签到失败：更新额度出错")
	}

	return checkin, nil
}

func GetUserCheckinStats(userId int, month string) (map[string]interface{}, error) {
	startDate := month + "-01"
	endDate := month + "-31"

	records, err := GetUserCheckinRecords(userId, startDate, endDate)
	if err != nil {
		return nil, err
	}

	checkinRecords := make([]CheckinRecord, len(records))
	for i, r := range records {
		checkinRecords[i] = CheckinRecord{
			CheckinDate:  r.CheckinDate,
			QuotaAwarded: r.QuotaAwarded,
		}
	}

	hasCheckedToday, _ := HasCheckedInToday(userId)

	var totalCheckins int64
	var totalQuota int64
	DB.Model(&Checkin{}).Where("user_id = ?", userId).Count(&totalCheckins)
	DB.Model(&Checkin{}).Where("user_id = ?", userId).Select("COALESCE(SUM(quota_awarded), 0)").Scan(&totalQuota)

	return map[string]interface{}{
		"total_quota":      totalQuota,
		"total_checkins":   totalCheckins,
		"checkin_count":    len(records),
		"checked_in_today": hasCheckedToday,
		"records":          checkinRecords,
	}, nil
}
