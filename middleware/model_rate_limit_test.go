package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newModelRateLimitContext(userId int) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	context.Set("id", userId)
	return context, recorder
}

func TestModelRequestRateLimitCountsByUserAndModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousConfig := setting.ModelRequestRateLimitModels2JSONString()
	previousRedisEnabled := common.RedisEnabled
	t.Cleanup(func() {
		require.NoError(t, setting.UpdateModelRequestRateLimitModelsByJSONString(previousConfig))
		common.RedisEnabled = previousRedisEnabled
	})

	common.RedisEnabled = false
	modelName := fmt.Sprintf("free-model-test-user-model-%d", time.Now().UnixNano())
	require.NoError(t, setting.UpdateModelRequestRateLimitModelsByJSONString(
		fmt.Sprintf(`{"enabled":true,"limits":{%q:2}}`, modelName),
	))

	context, _ := newModelRateLimitContext(10101)
	require.True(t, checkModelRequestRateLimit(context, modelName))
	context, _ = newModelRateLimitContext(10101)
	require.True(t, checkModelRequestRateLimit(context, modelName))
	context, recorder := newModelRateLimitContext(10101)
	require.False(t, checkModelRequestRateLimit(context, modelName))
	require.Equal(t, http.StatusTooManyRequests, recorder.Code)
	require.Equal(t, "60", recorder.Header().Get("Retry-After"))
	require.Contains(t, recorder.Body.String(), `"code":"rate_limit_reached"`)

	context, _ = newModelRateLimitContext(20202)
	require.True(t, checkModelRequestRateLimit(context, modelName))
	context, _ = newModelRateLimitContext(10101)
	require.True(t, checkModelRequestRateLimit(context, "unconfigured-model"))
}
