package model

import (
	"errors"
	"maps"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func checkinTestSettings() operation_setting.CheckinSetting {
	return operation_setting.CheckinSetting{
		Enabled: true, MinQuota: 5, MaxQuota: 50,
		Last10PercentConsumeQuota: 700, TwentyToTenPercentConsumeQuota: 550,
	}
}

func TestCheckinRewardBoundaries(t *testing.T) {
	setting := checkinTestSettings()
	cases := []struct {
		name     string
		consumed int64
		rolls    []int
		want     int
	}{
		{"minimum", 10, nil, 5},
		{"below20", 19, nil, 5},
		{"20lower", 20, []int{0}, 5},
		{"20upper", 20, []int{2}, 7},
		{"50inclusive", 50, []int{2}, 7},
		{"51ordinary", 51, []int{2}, 7},
		{"99cap", 99, []int{9}, 14},
		{"100low", 100, []int{84, 3}, 8},
		{"100high", 100, []int{85, 4}, 13},
		{"199noJackpot", 199, []int{99, 9}, 26},
		{"200jackpotLower", 200, []int{0, 0}, 45},
		{"200jackpotUpper", 200, []int{0, 5}, 50},
		{"200missLow", 200, []int{1, 84, 12}, 17},
		{"200missHigh", 200, []int{99, 85, 8}, 26},
		{"549ordinary", 549, []int{1, 85, 14}, 44},
		{"550secondLower", 550, []int{1, 0}, 40},
		{"550secondUpper", 550, []int{99, 4}, 44},
		{"550jackpot", 550, []int{0, 5}, 50},
		{"699second", 699, []int{1, 4}, 44},
		{"700topLower", 700, []int{0}, 45},
		{"700topUpper", 700, []int{5}, 50},
		{"largeConsumption", math.MaxInt64, []int{5}, 50},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			index := 0
			reward, err := setting.Reward(tc.consumed, func(n int) int {
				require.Less(t, index, len(tc.rolls), "unexpected random draw")
				roll := tc.rolls[index]
				index++
				require.Less(t, roll, n)
				return roll
			})
			require.NoError(t, err)
			assert.Equal(t, tc.want, reward)
			assert.Equal(t, len(tc.rolls), index)
		})
	}
	for _, consumed := range []int64{-1, 0, 9} {
		_, err := setting.Reward(consumed, func(int) int { t.Fatal("ineligible user must not draw"); return 0 })
		assert.Error(t, err)
	}
}

func TestCheckinConfigurableTiers(t *testing.T) {
	setting := checkinTestSettings()
	setting.Last10PercentConsumeQuota = 800
	setting.TwentyToTenPercentConsumeQuota = 600
	reward, err := setting.Reward(700, func(n int) int { return n - 1 })
	require.NoError(t, err)
	assert.Equal(t, 44, reward)
	setting.TwentyToTenPercentConsumeQuota = 0
	reward, err = setting.Reward(700, func(int) int { return 1 })
	require.NoError(t, err)
	assert.Equal(t, 6, reward, "disabled second tier uses ordinary rewards")
	setting.Last10PercentConsumeQuota = 300
	reward, err = setting.Reward(300, func(int) int { t.Fatal("fixed capped tier must not draw"); return 0 })
	require.NoError(t, err)
	assert.Equal(t, 45, reward, "top tier still respects the 15 percent cap")
	setting = checkinTestSettings()
	setting.MaxQuota = 100
	reward, err = setting.Reward(700, func(n int) int { return n - 1 })
	require.NoError(t, err)
	assert.Equal(t, 100, reward, "reward bounds remain configurable")
	for _, tc := range []struct {
		name   string
		mutate func(*operation_setting.CheckinSetting)
	}{
		{"minimum", func(s *operation_setting.CheckinSetting) { s.MinQuota = 6 }},
		{"maximum", func(s *operation_setting.CheckinSetting) { s.MaxQuota = 6 }},
		{"topCap", func(s *operation_setting.CheckinSetting) {
			s.Last10PercentConsumeQuota = 299
			s.TwentyToTenPercentConsumeQuota = 0
		}},
		{"secondCap", func(s *operation_setting.CheckinSetting) { s.TwentyToTenPercentConsumeQuota = 266 }},
		{"tierOrder", func(s *operation_setting.CheckinSetting) { s.TwentyToTenPercentConsumeQuota = 700 }},
		{"negativeGate", func(s *operation_setting.CheckinSetting) { s.MinPreviousDayRequests = -1 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := checkinTestSettings()
			tc.mutate(&s)
			assert.Error(t, s.Validate())
		})
	}
}

func setupCheckinDatabase(t *testing.T) (*gorm.DB, *gorm.DB) {
	t.Helper()
	require.False(t, common.RedisEnabled, "check-in tests require Redis disabled")
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	logDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	previousDB, previousLogDB, previousType := DB, LOG_DB, common.MainDatabaseType()
	common.OptionMapRWMutex.Lock()
	previousSetting := *operation_setting.GetCheckinSetting()
	previousOptions := maps.Clone(common.OptionMap)
	*operation_setting.GetCheckinSetting() = checkinTestSettings()
	common.OptionMap = map[string]string{}
	common.OptionMapRWMutex.Unlock()
	DB, LOG_DB = db, logDB
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	initCol()
	t.Cleanup(func() {
		DB, LOG_DB = previousDB, previousLogDB
		common.SetMainDatabaseType(previousType)
		initCol()
		common.OptionMapRWMutex.Lock()
		*operation_setting.GetCheckinSetting() = previousSetting
		common.OptionMap = previousOptions
		common.OptionMapRWMutex.Unlock()
	})
	for _, connection := range []*gorm.DB{db, logDB} {
		sqlDB, err := connection.DB()
		require.NoError(t, err)
		sqlDB.SetMaxOpenConns(1)
		t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	}
	require.NoError(t, db.AutoMigrate(&Checkin{}, &User{}, &Redemption{}, &Option{}))
	require.NoError(t, logDB.AutoMigrate(&Log{}))
	return db, logDB
}

func TestCheckinEligibilityAndDuplicateAward(t *testing.T) {
	db, logs := setupCheckinDatabase(t)
	require.NoError(t, db.Create(&User{Id: 1, Username: "checkin", Quota: 100}).Error)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterday := today.AddDate(0, 0, -1).Unix()
	// Other users, other log types, today's logs and older logs do not qualify.
	require.NoError(t, logs.Create(&[]Log{
		{UserId: 1, Type: LogTypeConsume, Quota: 9, CreatedAt: yesterday},
		{UserId: 2, Type: LogTypeConsume, Quota: 700, CreatedAt: yesterday},
		{UserId: 1, Type: LogTypeError, Quota: 700, CreatedAt: yesterday},
		{UserId: 1, Type: LogTypeConsume, Quota: 700, CreatedAt: today.Unix()},
		{UserId: 1, Type: LogTypeConsume, Quota: 700, CreatedAt: yesterday - 1},
	}).Error)
	_, err := UserCheckin(1)
	require.ErrorContains(t, err, "昨日消耗不足10")
	require.NoError(t, logs.Create(&Log{UserId: 1, Type: LogTypeConsume, Quota: 1, CreatedAt: today.Unix() - 1}).Error)
	require.NoError(t, UpdateOption("checkin_setting.min_previous_day_requests", "3"))
	_, err = UserCheckin(1)
	require.ErrorContains(t, err, "昨日请求次数不足")
	require.NoError(t, UpdateOption("checkin_setting.min_previous_day_requests", "2"))
	require.NoError(t, UpdateOption("checkin_setting.min_single_redemption_quota", "100"))
	require.NoError(t, db.Create(&[]Redemption{
		{Key: "small-a", UsedUserId: 1, Status: common.RedemptionCodeStatusUsed, Quota: 60},
		{Key: "small-b", UsedUserId: 1, Status: common.RedemptionCodeStatusUsed, Quota: 60},
		{Key: "other-user", UsedUserId: 2, Status: common.RedemptionCodeStatusUsed, Quota: 100},
		{Key: "not-used", UsedUserId: 1, Quota: 100},
	}).Error)
	_, err = UserCheckin(1)
	require.ErrorContains(t, err, "未使用过达到最低额度")
	redemption := Redemption{Key: "qualified", UsedUserId: 1, Status: common.RedemptionCodeStatusUsed, Quota: 100}
	require.NoError(t, db.Create(&redemption).Error)
	require.NoError(t, db.Delete(&redemption).Error)
	checkin, err := UserCheckin(1)
	require.NoError(t, err)
	assert.Equal(t, 5, checkin.QuotaAwarded)
	_, err = UserCheckin(1)
	require.ErrorContains(t, err, "今日已签到")
	var user User
	require.NoError(t, db.First(&user, 1).Error)
	assert.Equal(t, 105, user.Quota)
	var count int64
	require.NoError(t, db.Model(&Checkin{}).Count(&count).Error)
	assert.EqualValues(t, 1, count)
}

func TestCheckinSettingsAtomicSave(t *testing.T) {
	db, _ := setupCheckinDatabase(t)
	next := checkinTestSettings()
	next.Last10PercentConsumeQuota = 800
	next.TwentyToTenPercentConsumeQuota = 600
	data, err := common.Marshal(next)
	require.NoError(t, err)
	require.NoError(t, UpdateOption("checkin_setting", string(data)))
	assert.Equal(t, next, *operation_setting.GetCheckinSetting())
	for _, value := range []string{"1.5", "null", "5,\"enabled\":false"} {
		assert.Error(t, UpdateOption("checkin_setting.min_quota", value))
	}
	assert.Equal(t, next, *operation_setting.GetCheckinSetting())
	require.NoError(t, db.Callback().Update().Before("gorm:update").Register("checkin_fail", func(tx *gorm.DB) {
		tx.AddError(errors.New("save rejected"))
	}))
	assert.Error(t, UpdateOption("checkin_setting.last_10_percent_consume_quota", "900"))
	require.NoError(t, db.Callback().Update().Remove("checkin_fail"))
	assert.Equal(t, next, *operation_setting.GetCheckinSetting())
	var option Option
	require.NoError(t, db.Where(map[string]any{"key": "checkin_setting.last_10_percent_consume_quota"}).First(&option).Error)
	assert.Equal(t, strconv.Itoa(next.Last10PercentConsumeQuota), option.Value)
	assert.Equal(t, option.Value, common.OptionMap[option.Key])
	// A legacy incompatible setting can be disabled, but cannot be enabled.
	common.OptionMapRWMutex.Lock()
	operation_setting.GetCheckinSetting().MinQuota = 1000
	common.OptionMapRWMutex.Unlock()
	require.NoError(t, UpdateOption("checkin_setting.enabled", "false"))
	assert.Error(t, UpdateOption("checkin_setting.enabled", "true"))
	assert.False(t, operation_setting.GetCheckinSetting().Enabled)
}
