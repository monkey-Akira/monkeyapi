package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSelectedModelsShareSuccessfulRequestLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetMemoryModelSuccessQuotas()
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
		resetMemoryModelSuccessQuotas()
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
	setting.ModelRequestRateLimitCount = 0
	setting.ModelRequestRateLimitSuccessCount = 2
	common.RedisEnabled = false

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("id", int(modelSuffix%1000000000)+1000000000)
		c.Next()
	})
	router.Use(ModelRequestRateLimit())
	router.POST("/v1/chat/completions", func(c *gin.Context) {
		if c.GetHeader("X-Test-Relay-Failed") == "true" {
			common.SetContextKey(c, constant.ContextKeyRelayRequestSucceeded, false)
			c.Status(http.StatusOK)
			return
		}
		if c.GetHeader("X-Test-Fail") == "true" {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.Status(http.StatusOK)
	})

	requestModel := func(modelName string, headers ...string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(
			http.MethodPost,
			"/v1/chat/completions",
			strings.NewReader(fmt.Sprintf(`{"model":%q}`, modelName)),
		)
		request.Header.Set("Content-Type", "application/json")
		for index := 0; index+1 < len(headers); index += 2 {
			request.Header.Set(headers[index], headers[index+1])
		}
		router.ServeHTTP(recorder, request)
		return recorder
	}

	require.Equal(t, http.StatusOK, requestModel(modelA, "X-Test-Relay-Failed", "true").Code)
	require.Equal(t, http.StatusInternalServerError, requestModel(modelA, "X-Test-Fail", "true").Code)
	require.Equal(t, http.StatusOK, requestModel(unselectedModel).Code)
	require.Equal(t, http.StatusOK, requestModel(modelA).Code)
	require.Equal(t, http.StatusOK, requestModel(modelB).Code)
	require.Equal(t, http.StatusOK, requestModel(unselectedModel).Code)
	require.Equal(t, http.StatusTooManyRequests, requestModel(modelA).Code)
}

func TestSuccessfulRequestLimitReservesConcurrentCapacity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetMemoryModelSuccessQuotas()
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
		resetMemoryModelSuccessQuotas()
	})

	modelSuffix := time.Now().UnixNano()
	selectedModel := fmt.Sprintf("limited-model-%d", modelSuffix)
	require.NoError(t, setting.UpdateModelRequestRateLimitModelsByJSONString(
		fmt.Sprintf(`{"mode":"selected","models":[%q]}`, selectedModel),
	))
	setting.ModelRequestRateLimitEnabled = true
	setting.ModelRequestRateLimitDurationMinutes = 1
	setting.ModelRequestRateLimitCount = 0
	setting.ModelRequestRateLimitSuccessCount = 2
	common.RedisEnabled = false

	const requestCount = 8
	entered := make(chan struct{}, requestCount)
	release := make(chan struct{})
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("id", int(modelSuffix%1000000000)+1000000000)
		c.Next()
	})
	router.Use(ModelRequestRateLimit())
	router.POST("/v1/chat/completions", func(c *gin.Context) {
		entered <- struct{}{}
		<-release
		c.Status(http.StatusOK)
	})

	start := make(chan struct{})
	statuses := make(chan int, requestCount)
	var waitGroup sync.WaitGroup
	waitGroup.Add(requestCount)
	for range requestCount {
		go func() {
			defer waitGroup.Done()
			<-start
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(
				http.MethodPost,
				"/v1/chat/completions",
				strings.NewReader(fmt.Sprintf(`{"model":%q}`, selectedModel)),
			)
			request.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(recorder, request)
			statuses <- recorder.Code
		}()
	}
	close(start)

	for range 2 {
		select {
		case <-entered:
		case <-time.After(2 * time.Second):
			close(release)
			t.Fatal("timed out waiting for successful request quota reservations")
		}
	}
	close(release)
	waitGroup.Wait()
	close(statuses)

	successCount := 0
	rateLimitedCount := 0
	for status := range statuses {
		switch status {
		case http.StatusOK:
			successCount++
		case http.StatusTooManyRequests:
			rateLimitedCount++
		default:
			t.Fatalf("unexpected response status: %d", status)
		}
	}
	require.Equal(t, 2, successCount)
	require.Equal(t, requestCount-2, rateLimitedCount)
}
