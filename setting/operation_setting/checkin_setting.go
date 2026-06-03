package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

type CheckinSetting struct {
	Enabled                bool `json:"enabled"`
	MinQuota               int  `json:"min_quota"`
	MaxQuota               int  `json:"max_quota"`
	MinPreviousDayRequests int  `json:"min_previous_day_requests"`
}

var checkinSetting = CheckinSetting{
	Enabled:                false,
	MinQuota:               1000,
	MaxQuota:               10000,
	MinPreviousDayRequests: 0,
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
