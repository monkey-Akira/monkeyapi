package common

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/relaykit/types"
)

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

func ApplyCustomErrorMessage(errorCode, currentMessage string) string {
	if customMessage := GetCustomErrorMessage(errorCode); customMessage != "" {
		return customMessage
	}
	return currentMessage
}

func ApplyCustomUpstreamErrorMessage(errorCode, messageCode string, statusCode int, currentMessage string) string {
	errorCode = strings.TrimSpace(errorCode)
	messageCode = strings.TrimSpace(messageCode)
	missingErrorCode := errorCode == "" ||
		errorCode == "unknown_error" ||
		errorCode == string(types.ErrorCodeBadResponseStatusCode)

	candidates := make([]string, 0, 3)
	if messageCode != "" {
		candidates = append(candidates, messageCode)
	}
	if errorCode != "" {
		candidates = append(candidates, errorCode)
	}
	if missingErrorCode && statusCode > 0 {
		candidates = append(candidates, fmt.Sprintf("http_%d", statusCode))
	}

	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		if customMessage := GetCustomErrorMessage("upstream:" + candidate); customMessage != "" {
			return customMessage
		}
	}
	if !missingErrorCode {
		return ApplyCustomErrorMessage(errorCode, currentMessage)
	}
	if statusCode > 0 {
		if customMessage := GetCustomErrorMessage(fmt.Sprintf("upstream:http_%d", statusCode)); customMessage != "" {
			return customMessage
		}
	}
	return ApplyCustomErrorMessage(errorCode, currentMessage)
}

func ApplyCustomErrorMessageToAPIError(apiError *types.NewAPIError, currentMessage string) string {
	if apiError == nil {
		return currentMessage
	}
	errorCode := string(apiError.GetErrorCode())
	if apiError.IsUpstreamError() {
		return ApplyCustomUpstreamErrorMessage(
			errorCode,
			apiError.GetErrorMessageCode(),
			apiError.StatusCode,
			currentMessage,
		)
	}
	return ApplyCustomErrorMessage(errorCode, currentMessage)
}

func ToOpenAIErrorWithCustomMessage(apiError *types.NewAPIError) types.OpenAIError {
	result := apiError.ToOpenAIError()
	result.Message = ApplyCustomErrorMessageToAPIError(apiError, result.Message)
	return result
}

func ToClaudeErrorWithCustomMessage(apiError *types.NewAPIError) types.ClaudeError {
	result := apiError.ToClaudeError()
	result.Message = ApplyCustomErrorMessageToAPIError(apiError, result.Message)
	return result
}
