package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

type CheckinSetting struct {
	Enabled                        bool `json:"enabled"`
	MinQuota                       int  `json:"min_quota"`
	MaxQuota                       int  `json:"max_quota"`
	MinPreviousDayRequests         int  `json:"min_previous_day_requests"`
	MinSingleRedemptionQuota       int  `json:"min_single_redemption_quota"`
	Last10PercentConsumeQuota      int  `json:"last_10_percent_consume_quota"`
	TwentyToTenPercentConsumeQuota int  `json:"twenty_to_ten_percent_consume_quota"`
}

var checkinSetting = CheckinSetting{
	Enabled:                        false,
	MinQuota:                       1000,
	MaxQuota:                       10000,
	MinPreviousDayRequests:         0,
	MinSingleRedemptionQuota:       0,
	Last10PercentConsumeQuota:      0,
	TwentyToTenPercentConsumeQuota: 0,
}

func init() {
	config.GlobalConfig.Register("checkin_setting", &checkinSetting)
}

func GetCheckinSetting() *CheckinSetting {
	return &checkinSetting
}

func IsCheckinEnabled() bool {
	return checkinSetting.Enabled
}

func GetCheckinQuotaRange() (min, max int) {
	return checkinSetting.MinQuota, checkinSetting.MaxQuota
}

func GetLast10PercentConsumeQuota() int {
	if checkinSetting.Last10PercentConsumeQuota > 0 {
		return checkinSetting.Last10PercentConsumeQuota
	}
	if checkinSetting.MaxQuota > 0 {
		return checkinSetting.MaxQuota * 5
	}
	return 0
}
