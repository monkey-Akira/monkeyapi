package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

// 默认配置
var checkinSetting = CheckinSetting{
	Enabled:                        false, // 默认关闭
	MinQuota:                       5,
	MaxQuota:                       50,
	Last10PercentConsumeQuota:      700,
	TwentyToTenPercentConsumeQuota: 550,
}

func init() {
	// 注册到全局配置管理器
	config.GlobalConfig.Register("checkin_setting", &checkinSetting)
}

// GetCheckinSetting 获取签到配置
func GetCheckinSetting() *CheckinSetting {
	return &checkinSetting
}

// IsCheckinEnabled 是否启用签到功能
func IsCheckinEnabled() bool {
	return checkinSetting.Enabled
}

// GetCheckinQuotaRange 获取签到额度范围
func GetCheckinQuotaRange() (min, max int) {
	return checkinSetting.MinQuota, checkinSetting.MaxQuota
}
