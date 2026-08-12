package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestFormatUserLogsUsesUpstreamCustomMessage(t *testing.T) {
	const enabledKey = "error_message_setting.enabled"
	const mappingsKey = "error_message_setting.mappings"

	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	originalEnabled, hadEnabled := common.OptionMap[enabledKey]
	originalMappings, hadMappings := common.OptionMap[mappingsKey]
	common.OptionMap[enabledKey] = "true"
	common.OptionMap[mappingsKey] = `{"upstream:insufficient_user_quota":"当前模型线路暂时不可用，请稍后再试。"}`
	common.OptionMapRWMutex.Unlock()

	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		defer common.OptionMapRWMutex.Unlock()
		if hadEnabled {
			common.OptionMap[enabledKey] = originalEnabled
		} else {
			delete(common.OptionMap, enabledKey)
		}
		if hadMappings {
			common.OptionMap[mappingsKey] = originalMappings
		} else {
			delete(common.OptionMap, mappingsKey)
		}
	})

	const rawContent = "status_code=403, 预扣费额度失败, 用户剩余额度: 🍚1.000000, 需要预扣费额度: 🍚9.000000"
	logs := []*Log{
		{
			Id:      99,
			Type:    LogTypeError,
			Content: rawContent,
			Other:   `{"error_code":"insufficient_user_quota","status_code":403,"admin_info":{"use_channel":[1]}}`,
		},
		{
			Id:      100,
			Type:    LogTypeConsume,
			Content: rawContent,
			Other:   `{}`,
		},
	}

	formatUserLogs(logs, 10)

	require.Equal(t, "status_code=403, 当前模型线路暂时不可用，请稍后再试。", logs[0].Content)
	require.NotContains(t, logs[0].Other, "admin_info")
	require.Equal(t, 11, logs[0].Id)
	require.Equal(t, rawContent, logs[1].Content)
	require.Equal(t, 12, logs[1].Id)
}

func TestFormatUserLogsMapsNoAvailableAccounts(t *testing.T) {
	const enabledKey = "error_message_setting.enabled"
	const mappingsKey = "error_message_setting.mappings"

	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	originalEnabled, hadEnabled := common.OptionMap[enabledKey]
	originalMappings, hadMappings := common.OptionMap[mappingsKey]
	common.OptionMap[enabledKey] = "true"
	common.OptionMap[mappingsKey] = `{"upstream:no_available_accounts":"当前模型线路暂时没有可用账号，请稍后再试。","upstream:http_503":"上游服务暂时不可用，请稍后再试。"}`
	common.OptionMapRWMutex.Unlock()

	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		defer common.OptionMapRWMutex.Unlock()
		if hadEnabled {
			common.OptionMap[enabledKey] = originalEnabled
		} else {
			delete(common.OptionMap, enabledKey)
		}
		if hadMappings {
			common.OptionMap[mappingsKey] = originalMappings
		} else {
			delete(common.OptionMap, mappingsKey)
		}
	})

	const rawContent = "status_code=503, bad response status code 503, message: No available accounts (request id: secret-id)"
	logs := []*Log{
		{
			Id:      1,
			Type:    LogTypeError,
			Content: rawContent,
			Other:   `{"error_code":"unknown_error","status_code":503,"error_message_code":"no_available_accounts","admin_info":{"use_channel":[1]}}`,
		},
		{
			Id:      2,
			Type:    LogTypeError,
			Content: rawContent,
			Other:   `{"error_code":"unknown_error","status_code":503}`,
		},
	}

	formatUserLogs(logs, 0)

	for _, log := range logs {
		require.Equal(t, "status_code=503, 当前模型线路暂时没有可用账号，请稍后再试。", log.Content)
		require.NotContains(t, log.Content, "secret-id")
		require.NotContains(t, log.Other, "error_message_code")
	}
}

func TestFormatUserLogsFallsBackToPlainRealErrorCode(t *testing.T) {
	const enabledKey = "error_message_setting.enabled"
	const mappingsKey = "error_message_setting.mappings"

	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	originalEnabled, hadEnabled := common.OptionMap[enabledKey]
	originalMappings, hadMappings := common.OptionMap[mappingsKey]
	common.OptionMap[enabledKey] = "true"
	common.OptionMap[mappingsKey] = `{"server_error":"服务器错误，请稍后再试。","upstream:http_503":"上游服务暂时不可用，请稍后再试。"}`
	common.OptionMapRWMutex.Unlock()

	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		defer common.OptionMapRWMutex.Unlock()
		if hadEnabled {
			common.OptionMap[enabledKey] = originalEnabled
		} else {
			delete(common.OptionMap, enabledKey)
		}
		if hadMappings {
			common.OptionMap[mappingsKey] = originalMappings
		} else {
			delete(common.OptionMap, mappingsKey)
		}
	})

	logs := []*Log{
		{
			Type:    LogTypeError,
			Content: "status_code=503, upstream server details",
			Other:   `{"error_code":"server_error","status_code":503}`,
		},
	}

	formatUserLogs(logs, 0)

	require.Equal(t, "status_code=503, 服务器错误，请稍后再试。", logs[0].Content)
}
