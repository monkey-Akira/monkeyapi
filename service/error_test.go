package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestResetStatusCode(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		statusCode       int
		statusCodeConfig string
		expectedCode     int
	}{
		{
			name:             "map string value",
			statusCode:       429,
			statusCodeConfig: `{"429":"503"}`,
			expectedCode:     503,
		},
		{
			name:             "map int value",
			statusCode:       429,
			statusCodeConfig: `{"429":503}`,
			expectedCode:     503,
		},
		{
			name:             "skip invalid string value",
			statusCode:       429,
			statusCodeConfig: `{"429":"bad-code"}`,
			expectedCode:     429,
		},
		{
			name:             "skip status code 200",
			statusCode:       200,
			statusCodeConfig: `{"200":503}`,
			expectedCode:     200,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			newAPIError := &types.NewAPIError{
				StatusCode: tc.statusCode,
			}
			ResetStatusCode(newAPIError, tc.statusCodeConfig)
			require.Equal(t, tc.expectedCode, newAPIError.StatusCode)
		})
	}
}

func TestRelayErrorHandlerTruncatesInvalidJSONBodyInLog(t *testing.T) {
	withDebugEnabled(t, false)

	body := strings.Repeat("b", common.LocalLogContentLimit+256)
	var logBuffer bytes.Buffer

	common.LogWriterMu.Lock()
	oldWriter := gin.DefaultErrorWriter
	gin.DefaultErrorWriter = &logBuffer
	common.LogWriterMu.Unlock()
	t.Cleanup(func() {
		common.LogWriterMu.Lock()
		gin.DefaultErrorWriter = oldWriter
		common.LogWriterMu.Unlock()
	})

	resp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	newAPIError := RelayErrorHandler(context.Background(), resp, false)

	require.NotNil(t, newAPIError)
	require.Equal(t, "bad response status code 500", newAPIError.Error())
	require.Contains(t, logBuffer.String(), "[truncated")
	require.Contains(t, logBuffer.String(), fmt.Sprintf("original_length=%d", len(body)))
	require.NotContains(t, logBuffer.String(), strings.Repeat("b", common.LocalLogContentLimit+1))
}

func TestRelayErrorHandlerKeepsStructuredErrorMessage(t *testing.T) {
	message := strings.Repeat("c", common.LocalLogContentLimit+256)
	body := `{"message":"` + message + `"}`
	resp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	newAPIError := RelayErrorHandler(context.Background(), resp, false)

	require.NotNil(t, newAPIError)
	require.Equal(t, message, newAPIError.Error())
}

func TestRelayErrorHandlerKeepsOpenAIErrorMessage(t *testing.T) {
	message := strings.Repeat("d", common.LocalLogContentLimit+256)
	body := `{"error":{"message":"` + message + `","type":"server_error","code":"server_error"}}`
	resp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	newAPIError := RelayErrorHandler(context.Background(), resp, false)

	require.NotNil(t, newAPIError)
	require.Equal(t, message, newAPIError.Error())
}

func TestRelayErrorHandlerSeparatesLocalAndUpstreamQuotaMessages(t *testing.T) {
	const enabledKey = "error_message_setting.enabled"
	const mappingsKey = "error_message_setting.mappings"
	const detailedMessage = "预扣费额度失败, 用户剩余额度: 🍚1.000000, 需要预扣费额度: 🍚9.000000"
	const localMessage = "额度不足，请充值后再试。"
	const upstreamMessage = "当前模型线路暂时不可用，请稍后再试。"

	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	originalEnabled, hadEnabled := common.OptionMap[enabledKey]
	originalMappings, hadMappings := common.OptionMap[mappingsKey]
	common.OptionMap[enabledKey] = "true"
	common.OptionMap[mappingsKey] = `{"insufficient_user_quota":"额度不足，请充值后再试。","upstream:insufficient_user_quota":"当前模型线路暂时不可用，请稍后再试。"}`
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

	localErr := types.NewErrorWithStatusCode(
		errors.New(detailedMessage),
		types.ErrorCodeInsufficientUserQuota,
		http.StatusForbidden,
	)
	require.Equal(t, localMessage, localErr.ToOpenAIError().Message)

	body := `{"error":{"message":"` + detailedMessage + `","type":"new_api_error","param":"","code":"insufficient_user_quota"}}`
	resp := &http.Response{
		StatusCode: http.StatusForbidden,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	upstreamErr := RelayErrorHandler(context.Background(), resp, false)

	require.Equal(t, detailedMessage, upstreamErr.Error())
	require.Equal(t, http.StatusForbidden, upstreamErr.StatusCode)
	openAIError := upstreamErr.ToOpenAIError()
	require.Equal(t, upstreamMessage, openAIError.Message)
	require.Equal(t, string(types.ErrorCodeInsufficientUserQuota), openAIError.Code)
	require.Equal(t, upstreamMessage, upstreamErr.ToClaudeError().Message)
}

func TestRelayErrorHandlerMapsNoAvailableAccountsWithoutUpstreamCode(t *testing.T) {
	withCustomErrorMessages(t, `{
		"upstream:no_available_accounts":"当前模型线路暂时没有可用账号，请稍后再试。",
		"upstream:http_503":"上游服务暂时不可用，请稍后再试。"
	}`)

	body := `{"error":{"type":"<nil>","message":"No available accounts (request id: test-request-id)"},"type":"error"}`
	resp := &http.Response{
		StatusCode: http.StatusServiceUnavailable,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	upstreamErr := RelayErrorHandler(context.Background(), resp, false)

	require.Equal(t, http.StatusServiceUnavailable, upstreamErr.StatusCode)
	require.Equal(t, types.ErrorCode("unknown_error"), upstreamErr.GetErrorCode())
	require.Equal(t, "no_available_accounts", upstreamErr.GetErrorMessageCode())
	require.Equal(t, "当前模型线路暂时没有可用账号，请稍后再试。", upstreamErr.ToOpenAIError().Message)
	require.Equal(t, "当前模型线路暂时没有可用账号，请稍后再试。", upstreamErr.ToClaudeError().Message)
}

func TestRelayErrorHandlerUsesHTTPFallbackWithoutUpstreamCode(t *testing.T) {
	withCustomErrorMessages(t, `{"upstream:http_503":"上游服务暂时不可用，请稍后再试。"}`)

	body := `{"error":{"type":"<nil>","message":"Service temporarily overloaded"},"type":"error"}`
	resp := &http.Response{
		StatusCode: http.StatusServiceUnavailable,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	upstreamErr := RelayErrorHandler(context.Background(), resp, false)

	require.Equal(t, types.ErrorCode("unknown_error"), upstreamErr.GetErrorCode())
	require.Equal(t, "http_503", upstreamErr.GetErrorMessageCode())
	require.Equal(t, "上游服务暂时不可用，请稍后再试。", upstreamErr.ToOpenAIError().Message)
}

func TestRelayErrorHandlerKeepsGenericFallbackWithoutHTTPMapping(t *testing.T) {
	withCustomErrorMessages(t, `{"bad_response_status_code":"上游响应异常，请稍后再试。"}`)

	body := `{"message":"Service temporarily overloaded"}`
	resp := &http.Response{
		StatusCode: http.StatusServiceUnavailable,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	upstreamErr := RelayErrorHandler(context.Background(), resp, false)

	require.Equal(t, types.ErrorCodeBadResponseStatusCode, upstreamErr.GetErrorCode())
	require.Equal(t, "上游响应异常，请稍后再试。", upstreamErr.ToOpenAIError().Message)
}

func TestRelayErrorHandlerPrefersRealUpstreamCode(t *testing.T) {
	withCustomErrorMessages(t, `{
		"upstream:server_error":"上游返回服务器错误。",
		"upstream:no_available_accounts":"当前模型线路暂时没有可用账号，请稍后再试。",
		"upstream:http_503":"上游服务暂时不可用，请稍后再试。"
	}`)

	body := `{"error":{"message":"No available accounts","type":"server_error","code":"server_error"}}`
	resp := &http.Response{
		StatusCode: http.StatusServiceUnavailable,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	upstreamErr := RelayErrorHandler(context.Background(), resp, false)

	require.Equal(t, types.ErrorCode("server_error"), upstreamErr.GetErrorCode())
	require.Equal(t, "server_error", upstreamErr.GetErrorMessageCode())
	require.Equal(t, "上游返回服务器错误。", upstreamErr.ToOpenAIError().Message)
}

func TestRelayErrorHandlerFallsBackToPlainRealErrorCodeBeforeHTTPStatus(t *testing.T) {
	withCustomErrorMessages(t, `{
		"server_error":"服务器错误，请稍后再试。",
		"upstream:http_503":"上游服务暂时不可用，请稍后再试。"
	}`)

	body := `{"error":{"message":"Service temporarily overloaded","type":"server_error","code":"server_error"}}`
	resp := &http.Response{
		StatusCode: http.StatusServiceUnavailable,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	upstreamErr := RelayErrorHandler(context.Background(), resp, false)

	require.Equal(t, "服务器错误，请稍后再试。", upstreamErr.ToOpenAIError().Message)
}

func TestRelayErrorHandlerKeepsInvalidJSONBodyInDebugLog(t *testing.T) {
	withDebugEnabled(t, true)

	body := strings.Repeat("e", common.LocalLogContentLimit+256)
	var logBuffer bytes.Buffer

	common.LogWriterMu.Lock()
	oldWriter := gin.DefaultErrorWriter
	gin.DefaultErrorWriter = &logBuffer
	common.LogWriterMu.Unlock()
	t.Cleanup(func() {
		common.LogWriterMu.Lock()
		gin.DefaultErrorWriter = oldWriter
		common.LogWriterMu.Unlock()
	})

	resp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	newAPIError := RelayErrorHandler(context.Background(), resp, false)

	require.NotNil(t, newAPIError)
	require.NotContains(t, logBuffer.String(), "[truncated")
	require.Contains(t, logBuffer.String(), body)
}

func withDebugEnabled(t *testing.T, enabled bool) {
	t.Helper()

	oldDebug := common.DebugEnabled
	common.DebugEnabled = enabled
	t.Cleanup(func() {
		common.DebugEnabled = oldDebug
	})
}

func withCustomErrorMessages(t *testing.T, mappings string) {
	t.Helper()

	const enabledKey = "error_message_setting.enabled"
	const mappingsKey = "error_message_setting.mappings"
	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	originalEnabled, hadEnabled := common.OptionMap[enabledKey]
	originalMappings, hadMappings := common.OptionMap[mappingsKey]
	common.OptionMap[enabledKey] = "true"
	common.OptionMap[mappingsKey] = mappings
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
}
