package common

import "strings"

const (
	errorMessageSettingEnabledKey  = "error_message_setting.enabled"
	errorMessageSettingMappingsKey = "error_message_setting.mappings"
)

func GetCustomErrorMessage(errorCode string) string {
	errorCode = strings.TrimSpace(errorCode)
	if errorCode == "" {
		return ""
	}

	OptionMapRWMutex.RLock()
	enabled := OptionMap[errorMessageSettingEnabledKey] == "true"
	mappingJSON := OptionMap[errorMessageSettingMappingsKey]
	OptionMapRWMutex.RUnlock()

	if !enabled || strings.TrimSpace(mappingJSON) == "" {
		return ""
	}

	var mappings map[string]string
	if err := UnmarshalJsonStr(mappingJSON, &mappings); err != nil {
		return ""
	}

	return strings.TrimSpace(mappings[errorCode])
}

func ApplyCustomErrorMessage(errorCode string, currentMessage string) string {
	if customMessage := GetCustomErrorMessage(errorCode); customMessage != "" {
		return customMessage
	}
	return currentMessage
}
