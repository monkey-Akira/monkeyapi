package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSelectedModelsShareExistingUserRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousConfig := setting.ModelRequestRateLimitModels2JSONString()
	previousEnabled := setting.ModelRequestRateLimitEnabled
	previousDuration := setting.ModelRequestRateLimitDurationMinutes
	previousTotalCount := setting.ModelRequestRateLimitCount
	previousSuccessCount := setting.ModelRequestRateLimitSuccessCount
	previousRedisEnabled := common.RedisEnabled
	t.Cleanup(func() {
		require.NoError(t, setting.UpdateModelRequestRateLimitModelsByJSONString(previousConfig))
		setting.ModelRequestRateLimitEnabled = previousEnabled
		setting.ModelRequestRateLimitDurationMinutes = previousDuration
		setting.ModelRequestRateLimitCount = previousTotalCount
		setting.ModelRequestRateLimitSuccessCount = previousSuccessCount
		common.RedisEnabled = previousRedisEnabled
	})

	modelSuffix := time.Now().UnixNano()
	modelA := fmt.Sprintf("free-model-a-%d", modelSuffix)
	modelB := fmt.Sprintf("free-model-b-%d", modelSuffix)
	unselectedModel := fmt.Sprintf("paid-model-%d", modelSuffix)
	require.NoError(t, setting.UpdateModelRequestRateLimitModelsByJSONString(
		fmt.Sprintf(`{"mode":"selected","models":[%q,%q]}`, modelA, modelB),
	))
	setting.ModelRequestRateLimitEnabled = true
	setting.ModelRequestRateLimitDurationMinutes = 1
	setting.ModelRequestRateLimitCount = 2
	setting.ModelRequestRateLimitSuccessCount = 1000
	common.RedisEnabled = false

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("id", int(modelSuffix%1000000000)+1000000000)
		c.Next()
	})
	router.Use(ModelRequestRateLimit())
	router.POST("/v1/chat/completions", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	requestModel := func(modelName string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(
			http.MethodPost,
			"/v1/chat/completions",
			strings.NewReader(fmt.Sprintf(`{"model":%q}`, modelName)),
		)
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(recorder, request)
		return recorder
	}

	require.Equal(t, http.StatusOK, requestModel(unselectedModel).Code)
	require.Equal(t, http.StatusOK, requestModel(modelA).Code)
	require.Equal(t, http.StatusOK, requestModel(modelB).Code)
	require.Equal(t, http.StatusOK, requestModel(unselectedModel).Code)
	require.Equal(t, http.StatusTooManyRequests, requestModel(modelA).Code)
}
