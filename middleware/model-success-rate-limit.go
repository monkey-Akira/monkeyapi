package middleware

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/go-redis/redis/v8"
)

const (
	modelSuccessQuotaKeyVersion        = "v2"
	modelSuccessQuotaReservationLease  = 2 * time.Minute
	modelSuccessQuotaHeartbeatInterval = 30 * time.Second
)

var reserveModelSuccessQuotaScript = redis.NewScript(`
local successKey = KEYS[1]
local pendingKey = KEYS[2]
local now = tonumber(ARGV[1])
local windowMs = tonumber(ARGV[2])
local leaseMs = tonumber(ARGV[3])
local maxCount = tonumber(ARGV[4])
local reservationId = ARGV[5]

redis.call('ZREMRANGEBYSCORE', successKey, '-inf', now - windowMs)
redis.call('ZREMRANGEBYSCORE', pendingKey, '-inf', now)
local used = redis.call('ZCARD', successKey) + redis.call('ZCARD', pendingKey)
if used >= maxCount then
    return 0
end
redis.call('ZADD', pendingKey, now + leaseMs, reservationId)
redis.call('PEXPIRE', pendingKey, leaseMs * 2)
return 1
`)

var heartbeatModelSuccessQuotaScript = redis.NewScript(`
local pendingKey = KEYS[1]
local now = tonumber(ARGV[1])
local leaseMs = tonumber(ARGV[2])
local reservationId = ARGV[3]
if redis.call('ZSCORE', pendingKey, reservationId) == false then
    return 0
end
redis.call('ZADD', pendingKey, now + leaseMs, reservationId)
redis.call('PEXPIRE', pendingKey, leaseMs * 2)
return 1
`)

var finishModelSuccessQuotaScript = redis.NewScript(`
local successKey = KEYS[1]
local pendingKey = KEYS[2]
local now = tonumber(ARGV[1])
local windowMs = tonumber(ARGV[2])
local leaseMs = tonumber(ARGV[3])
local reservationId = ARGV[4]
local succeeded = tonumber(ARGV[5])
redis.call('ZREM', pendingKey, reservationId)
redis.call('ZREMRANGEBYSCORE', successKey, '-inf', now - windowMs)
if succeeded == 1 then
    redis.call('ZADD', successKey, now, reservationId)
    redis.call('PEXPIRE', successKey, windowMs + 1000)
end
if redis.call('ZCARD', pendingKey) == 0 then
    redis.call('DEL', pendingKey)
else
    redis.call('PEXPIRE', pendingKey, leaseMs * 2)
end
return 1
`)

type modelSuccessQuotaReservation interface {
	Finish(succeeded bool) error
}

func reserveModelSuccessQuota(ctx context.Context, userID, requestID string, maxCount int, window time.Duration) (modelSuccessQuotaReservation, bool, error) {
	if maxCount <= 0 {
		return nil, true, nil
	}
	if window <= 0 {
		return nil, false, fmt.Errorf("success request rate limit window must be positive")
	}
	if requestID == "" {
		requestID = common.GetRandomString(32)
	} else {
		requestID += ":" + common.GetRandomString(8)
	}

	if common.RedisEnabled {
		if common.RDB == nil {
			return nil, false, fmt.Errorf("Redis is enabled but unavailable")
		}
		return reserveRedisModelSuccessQuota(ctx, common.RDB, userID, requestID, maxCount, window)
	}

	return memoryModelSuccessQuotas.Reserve(userID, requestID, maxCount, window)
}

type redisModelSuccessQuotaReservation struct {
	rdb        *redis.Client
	successKey string
	pendingKey string
	requestID  string
	window     time.Duration
	stop       chan struct{}
	done       chan struct{}
	finishOnce sync.Once
	finishErr  error
}

func reserveRedisModelSuccessQuota(ctx context.Context, rdb *redis.Client, userID, requestID string, maxCount int, window time.Duration) (modelSuccessQuotaReservation, bool, error) {
	successKey, pendingKey := modelSuccessQuotaRedisKeys(userID)
	allowed, err := reserveModelSuccessQuotaScript.Run(
		ctx,
		rdb,
		[]string{successKey, pendingKey},
		time.Now().UnixMilli(),
		window.Milliseconds(),
		modelSuccessQuotaReservationLease.Milliseconds(),
		maxCount,
		requestID,
	).Int()
	if err != nil {
		return nil, false, fmt.Errorf("reserve successful request quota: %w", err)
	}
	if allowed != 1 {
		return nil, false, nil
	}

	reservation := &redisModelSuccessQuotaReservation{
		rdb:        rdb,
		successKey: successKey,
		pendingKey: pendingKey,
		requestID:  requestID,
		window:     window,
		stop:       make(chan struct{}),
		done:       make(chan struct{}),
	}
	reservation.startHeartbeat()
	return reservation, true, nil
}

func modelSuccessQuotaRedisKeys(userID string) (string, string) {
	base := fmt.Sprintf("rateLimit:%s:%s:{%s}", ModelRequestRateLimitSuccessCountMark, modelSuccessQuotaKeyVersion, userID)
	return base + ":success", base + ":pending"
}

func (r *redisModelSuccessQuotaReservation) startHeartbeat() {
	go func() {
		defer close(r.done)
		ticker := time.NewTicker(modelSuccessQuotaHeartbeatInterval)
		defer ticker.Stop()

		for {
			select {
			case <-r.stop:
				return
			case <-ticker.C:
				kept, err := heartbeatModelSuccessQuotaScript.Run(
					context.Background(),
					r.rdb,
					[]string{r.pendingKey},
					time.Now().UnixMilli(),
					modelSuccessQuotaReservationLease.Milliseconds(),
					r.requestID,
				).Int()
				if err != nil {
					common.SysError(fmt.Sprintf("refresh successful request quota reservation: %v", err))
					continue
				}
				if kept != 1 {
					return
				}
			}
		}
	}()
}

func (r *redisModelSuccessQuotaReservation) Finish(succeeded bool) error {
	r.finishOnce.Do(func() {
		close(r.stop)
		<-r.done

		succeededValue := 0
		if succeeded {
			succeededValue = 1
		}
		_, r.finishErr = finishModelSuccessQuotaScript.Run(
			context.Background(),
			r.rdb,
			[]string{r.successKey, r.pendingKey},
			time.Now().UnixMilli(),
			r.window.Milliseconds(),
			modelSuccessQuotaReservationLease.Milliseconds(),
			r.requestID,
			succeededValue,
		).Result()
		if r.finishErr != nil {
			r.finishErr = fmt.Errorf("finish successful request quota reservation: %w", r.finishErr)
		}
	})
	return r.finishErr
}

type memoryModelSuccessQuotaState struct {
	successes []int64
	pending   map[string]struct{}
}

type memoryModelSuccessQuotaStore struct {
	mutex  sync.Mutex
	states map[string]*memoryModelSuccessQuotaState
}

var memoryModelSuccessQuotas = &memoryModelSuccessQuotaStore{
	states: make(map[string]*memoryModelSuccessQuotaState),
}

func (s *memoryModelSuccessQuotaStore) Reserve(userID, requestID string, maxCount int, window time.Duration) (modelSuccessQuotaReservation, bool, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now().UnixMilli()
	state := s.states[userID]
	if state == nil {
		state = &memoryModelSuccessQuotaState{pending: make(map[string]struct{})}
		s.states[userID] = state
	}
	state.successes = pruneModelSuccessQuotaTimestamps(state.successes, now-window.Milliseconds())
	if len(state.successes)+len(state.pending) >= maxCount {
		return nil, false, nil
	}

	state.pending[requestID] = struct{}{}
	return &memoryModelSuccessQuotaReservation{
		store:     s,
		userID:    userID,
		requestID: requestID,
		window:    window,
	}, true, nil
}

type memoryModelSuccessQuotaReservation struct {
	store      *memoryModelSuccessQuotaStore
	userID     string
	requestID  string
	window     time.Duration
	finishOnce sync.Once
}

func (r *memoryModelSuccessQuotaReservation) Finish(succeeded bool) error {
	r.finishOnce.Do(func() {
		r.store.mutex.Lock()
		defer r.store.mutex.Unlock()

		state := r.store.states[r.userID]
		if state == nil {
			return
		}
		delete(state.pending, r.requestID)
		now := time.Now().UnixMilli()
		state.successes = pruneModelSuccessQuotaTimestamps(state.successes, now-r.window.Milliseconds())
		if succeeded {
			state.successes = append(state.successes, now)
		}
		if len(state.successes) == 0 && len(state.pending) == 0 {
			delete(r.store.states, r.userID)
		}
	})
	return nil
}

func pruneModelSuccessQuotaTimestamps(timestamps []int64, cutoff int64) []int64 {
	kept := timestamps[:0]
	for _, timestamp := range timestamps {
		if timestamp > cutoff {
			kept = append(kept, timestamp)
		}
	}
	return kept
}

func resetMemoryModelSuccessQuotas() {
	memoryModelSuccessQuotas.mutex.Lock()
	memoryModelSuccessQuotas.states = make(map[string]*memoryModelSuccessQuotaState)
	memoryModelSuccessQuotas.mutex.Unlock()
}
