package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/common/limiter"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

const (
	ModelRequestRateLimitCountMark        = "MRRL"
	ModelRequestRateLimitSuccessCountMark = "MRRLS"
)

func shouldApplyModelRequestRateLimit(c *gin.Context) bool {
	if setting.ModelRequestRateLimitAppliesToAllModels() {
		return true
	}

	modelRequest, _, err := getModelRequest(c)
	if err != nil {
		return false
	}
	requestedModel := modelRequest.Model
	if modelRequest.RequestedModel != "" {
		requestedModel = modelRequest.RequestedModel
	}
	return setting.ShouldApplyModelRequestRateLimit(requestedModel)
}

func checkTotalModelRequestRateLimit(ctx context.Context, userID string, maxCount int, durationSeconds int64) (bool, error) {
	if maxCount <= 0 {
		return true, nil
	}
	if common.RedisEnabled {
		if common.RDB == nil {
			return false, fmt.Errorf("Redis is enabled but unavailable")
		}
		totalKey := fmt.Sprintf("rateLimit:%s", userID)
		tokenBucket := limiter.New(ctx, common.RDB)
		return tokenBucket.Allow(
			ctx,
			totalKey,
			limiter.WithCapacity(int64(maxCount)*durationSeconds),
			limiter.WithRate(int64(maxCount)),
			limiter.WithRequested(durationSeconds),
		)
	}
	inMemoryRateLimiter.Init(time.Duration(durationSeconds) * time.Second)
	return inMemoryRateLimiter.Request(ModelRequestRateLimitCountMark+userID, maxCount, durationSeconds), nil
}

func modelRequestCompletedSuccessfully(c *gin.Context) bool {
	status := c.Writer.Status()
	if status < http.StatusOK || status >= http.StatusMultipleChoices {
		return false
	}
	if c.Request != nil && c.Request.Context().Err() != nil {
		return false
	}
	if value, exists := common.GetContextKey(c, constant.ContextKeyRelayRequestSucceeded); exists {
		succeeded, ok := value.(bool)
		return ok && succeeded
	}
	return true
}

func modelRequestRateLimitHandler(durationSeconds int64, totalMaxCount, successMaxCount int) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := strconv.Itoa(c.GetInt("id"))
		reservation, allowed, err := reserveModelSuccessQuota(
			c.Request.Context(), userID, c.GetString(common.RequestIdKey), successMaxCount,
			time.Duration(durationSeconds)*time.Second,
		)
		if err != nil {
			common.SysError(fmt.Sprintf("reserve successful request quota: %v", err))
			abortWithOpenAiMessage(c, http.StatusInternalServerError, "rate_limit_check_failed")
			return
		}
		if !allowed {
			c.Header("Retry-After", strconv.FormatInt(durationSeconds, 10))
			message := fmt.Sprintf(
				"您已达到成功请求数限制：%d分钟内最多成功请求%d次",
				setting.ModelRequestRateLimitDurationMinutes, successMaxCount,
			)
			abortWithCustomOpenAiMessage(
				c,
				http.StatusTooManyRequests,
				message,
				types.ErrorCodeModelSuccessRateLimitExceeded,
			)
			return
		}

		completed := false
		if reservation != nil {
			defer func() {
				if finishErr := reservation.Finish(completed && modelRequestCompletedSuccessfully(c)); finishErr != nil {
					common.SysError(fmt.Sprintf("finish successful request quota: %v", finishErr))
				}
			}()
		}

		allowed, err = checkTotalModelRequestRateLimit(c.Request.Context(), userID, totalMaxCount, durationSeconds)
		if err != nil {
			common.SysError(fmt.Sprintf("check total request rate limit: %v", err))
			abortWithOpenAiMessage(c, http.StatusInternalServerError, "rate_limit_check_failed")
			return
		}
		if !allowed {
			c.Header("Retry-After", strconv.FormatInt(durationSeconds, 10))
			abortWithOpenAiMessage(c, http.StatusTooManyRequests, fmt.Sprintf(
				"您已达到总请求数限制：%d分钟内最多请求%d次，包括失败次数，请检查您的请求是否正确",
				setting.ModelRequestRateLimitDurationMinutes, totalMaxCount,
			))
			return
		}

		c.Next()
		completed = true
	}
}

// ModelRequestRateLimit applies the configured per-user limits to matching models.
func ModelRequestRateLimit() func(c *gin.Context) {
	return func(c *gin.Context) {
		// 在每个请求时检查是否启用限流
		if !setting.ModelRequestRateLimitEnabled {
			c.Next()
			return
		}
		if !shouldApplyModelRequestRateLimit(c) {
			c.Next()
			return
		}

		// 计算限流参数
		duration := int64(setting.ModelRequestRateLimitDurationMinutes * 60)
		totalMaxCount := setting.ModelRequestRateLimitCount
		successMaxCount := setting.ModelRequestRateLimitSuccessCount

		// 获取分组
		group := common.GetContextKeyString(c, constant.ContextKeyTokenGroup)
		if group == "" {
			group = common.GetContextKeyString(c, constant.ContextKeyUserGroup)
		}

		//获取分组的限流配置
		groupTotalCount, groupSuccessCount, found := setting.GetGroupRateLimit(group)
		if found {
			totalMaxCount = groupTotalCount
			successMaxCount = groupSuccessCount
		}

		// 根据存储类型选择并执行限流处理器
		modelRequestRateLimitHandler(duration, totalMaxCount, successMaxCount)(c)
	}
}
