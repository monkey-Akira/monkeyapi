package service

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/go-redis/redis/v8"
)

const (
	AdminSessionTokenKey    = "admin_session_token"
	adminSessionPrefix      = "admin_session:"
	adminUserSessionsPrefix = "admin_user_sessions:"
	adminSessionTTL         = 30 * 24 * time.Hour

	adminSessionUserIDField             = "user_id"
	adminSessionVerifiedAtField         = "secure_verified_at"
	adminSessionVerificationMethodField = "secure_verified_method"
	adminSessionPasskeyReadyAtField     = "secure_passkey_ready_at"
)

var ErrAdminSessionUnavailable = errors.New("administrator session is unavailable")

type AdminSecureVerification struct {
	VerifiedAt int64
	Method     string
}

func adminSessionKey(token string) (string, error) {
	token = strings.TrimSpace(token)
	if len(token) != 64 {
		return "", ErrAdminSessionUnavailable
	}
	sum := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%s%x", adminSessionPrefix, sum), nil
}

func adminSessionClient() (*redis.Client, error) {
	if !common.RedisEnabled || common.RDB == nil {
		return nil, ErrAdminSessionUnavailable
	}
	return common.RDB, nil
}

func CreateAdminSession(userID int) (string, error) {
	if userID <= 0 {
		return "", ErrAdminSessionUnavailable
	}
	client, err := adminSessionClient()
	if err != nil {
		return "", err
	}
	token, err := common.GenerateRandomCharsKey(64)
	if err != nil {
		return "", err
	}
	key, err := adminSessionKey(token)
	if err != nil {
		return "", err
	}
	ctx := context.Background()
	userSessionsKey := adminUserSessionsKey(userID)
	_, err = client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.HSet(ctx, key, adminSessionUserIDField, userID)
		pipe.Expire(ctx, key, adminSessionTTL)
		pipe.SAdd(ctx, userSessionsKey, key)
		pipe.Expire(ctx, userSessionsKey, adminSessionTTL)
		return nil
	})
	if err != nil {
		return "", err
	}
	return token, nil
}

func GetAdminSessionUserID(token string) (int, error) {
	client, err := adminSessionClient()
	if err != nil {
		return 0, err
	}
	key, err := adminSessionKey(token)
	if err != nil {
		return 0, err
	}
	value, err := client.HGet(context.Background(), key, adminSessionUserIDField).Result()
	if err != nil {
		return 0, ErrAdminSessionUnavailable
	}
	userID, err := strconv.Atoi(value)
	if err != nil || userID <= 0 {
		return 0, ErrAdminSessionUnavailable
	}
	return userID, nil
}

func DeleteAdminSession(token string) error {
	if strings.TrimSpace(token) == "" {
		return nil
	}
	client, err := adminSessionClient()
	if err != nil {
		return err
	}
	key, err := adminSessionKey(token)
	if err != nil {
		return err
	}
	ctx := context.Background()
	userID, err := client.HGet(ctx, key, adminSessionUserIDField).Int()
	if errors.Is(err, redis.Nil) {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Del(ctx, key)
		pipe.SRem(ctx, adminUserSessionsKey(userID), key)
		return nil
	})
	return err
}

func DeleteAdminSessionsForUser(userID int) error {
	if userID <= 0 {
		return nil
	}
	client, err := adminSessionClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	userSessionsKey := adminUserSessionsKey(userID)
	sessionKeys, err := client.SMembers(ctx, userSessionsKey).Result()
	if err != nil {
		return err
	}
	keys := append(sessionKeys, userSessionsKey)
	return client.Del(ctx, keys...).Err()
}

func SetAdminSecureVerification(token string, verifiedAt int64, method string) error {
	client, key, err := existingAdminSession(token)
	if err != nil {
		return err
	}
	return client.HSet(
		context.Background(),
		key,
		adminSessionVerifiedAtField, verifiedAt,
		adminSessionVerificationMethodField, method,
	).Err()
}

func GetAdminSecureVerification(token string) (AdminSecureVerification, error) {
	client, key, err := existingAdminSession(token)
	if err != nil {
		return AdminSecureVerification{}, err
	}
	values, err := client.HMGet(
		context.Background(),
		key,
		adminSessionVerifiedAtField,
		adminSessionVerificationMethodField,
	).Result()
	if err != nil || len(values) != 2 || values[0] == nil {
		return AdminSecureVerification{}, nil
	}
	verifiedAt, err := strconv.ParseInt(fmt.Sprint(values[0]), 10, 64)
	if err != nil {
		return AdminSecureVerification{}, ErrAdminSessionUnavailable
	}
	method := ""
	if values[1] != nil {
		method = fmt.Sprint(values[1])
	}
	return AdminSecureVerification{VerifiedAt: verifiedAt, Method: method}, nil
}

func ClearAdminSecureVerification(token string) error {
	client, key, err := existingAdminSession(token)
	if err != nil {
		return err
	}
	return client.HDel(
		context.Background(),
		key,
		adminSessionVerifiedAtField,
		adminSessionVerificationMethodField,
	).Err()
}

func SetAdminPasskeyReady(token string, readyAt int64) error {
	client, key, err := existingAdminSession(token)
	if err != nil {
		return err
	}
	return client.HSet(context.Background(), key, adminSessionPasskeyReadyAtField, readyAt).Err()
}

func ConsumeAdminPasskeyReady(token string) (int64, error) {
	client, key, err := existingAdminSession(token)
	if err != nil {
		return 0, err
	}
	ctx := context.Background()
	value, err := client.HGet(ctx, key, adminSessionPasskeyReadyAtField).Result()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if err := client.HDel(ctx, key, adminSessionPasskeyReadyAtField).Err(); err != nil {
		return 0, err
	}
	readyAt, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, ErrAdminSessionUnavailable
	}
	return readyAt, nil
}

func existingAdminSession(token string) (*redis.Client, string, error) {
	client, err := adminSessionClient()
	if err != nil {
		return nil, "", err
	}
	key, err := adminSessionKey(token)
	if err != nil {
		return nil, "", err
	}
	exists, err := client.Exists(context.Background(), key).Result()
	if err != nil || exists != 1 {
		return nil, "", ErrAdminSessionUnavailable
	}
	return client, key, nil
}

func adminUserSessionsKey(userID int) string {
	return fmt.Sprintf("%s%d", adminUserSessionsPrefix, userID)
}
