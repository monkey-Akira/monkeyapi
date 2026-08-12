package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

const (
	EmptyResponseRefundModeOff     = "off"
	EmptyResponseRefundModeObserve = "observe"
	EmptyResponseRefundModeRefund  = "refund"
)

type EmptyResponseRefundSetting struct {
	Mode                  string   `json:"mode"`
	Models                []string `json:"models"`
	CustomResponseEnabled bool     `json:"custom_response_enabled"`
	CustomResponseText    string   `json:"custom_response_text"`
}

var emptyResponseRefundSetting = EmptyResponseRefundSetting{
	Mode:                  EmptyResponseRefundModeOff,
	Models:                []string{},
	CustomResponseEnabled: false,
	CustomResponseText:    "",
}

func init() {
	config.GlobalConfig.Register("empty_response_refund_setting", &emptyResponseRefundSetting)
}

func GetEmptyResponseRefundMode(modelName string) string {
	mode := emptyResponseRefundSetting.Mode
	if modelName == "" || mode == EmptyResponseRefundModeOff {
		return EmptyResponseRefundModeOff
	}
	if mode != EmptyResponseRefundModeObserve && mode != EmptyResponseRefundModeRefund {
		return EmptyResponseRefundModeOff
	}
	for _, configuredModel := range emptyResponseRefundSetting.Models {
		if configuredModel == modelName {
			return mode
		}
	}
	return EmptyResponseRefundModeOff
}

func GetEmptyResponseRefundCustomText() string {
	if !emptyResponseRefundSetting.CustomResponseEnabled {
		return ""
	}
	return emptyResponseRefundSetting.CustomResponseText
}
