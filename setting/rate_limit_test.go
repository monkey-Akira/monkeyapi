package setting

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeModelRequestRateLimitModels(t *testing.T) {
	normalized, err := NormalizeModelRequestRateLimitModels(`{
		"enabled": true,
		"limits": {" free-model ": 10, "other-model": 2}
	}`)
	require.NoError(t, err)
	require.JSONEq(t, `{
		"enabled": true,
		"limits": {"free-model": 10, "other-model": 2}
	}`, normalized)
}

func TestNormalizeModelRequestRateLimitModelsRejectsInvalidRPM(t *testing.T) {
	_, err := NormalizeModelRequestRateLimitModels(`{"enabled":true,"limits":{"free-model":0}}`)
	require.EqualError(t, err, "模型 free-model 的每分钟调用次数必须在 1 到 100000000 之间")
}

func TestNormalizeModelRequestRateLimitModelsRequiresCompleteObject(t *testing.T) {
	_, err := NormalizeModelRequestRateLimitModels(`null`)
	require.EqualError(t, err, "指定模型限流配置必须包含 enabled 和 limits")

	_, err = NormalizeModelRequestRateLimitModels(`{"enabled":true}`)
	require.EqualError(t, err, "指定模型限流配置必须包含 enabled 和 limits")
}

func TestGetModelRequestRateLimitRPMMatchesExactModelName(t *testing.T) {
	previous := ModelRequestRateLimitModels2JSONString()
	t.Cleanup(func() {
		require.NoError(t, UpdateModelRequestRateLimitModelsByJSONString(previous))
	})

	require.NoError(t, UpdateModelRequestRateLimitModelsByJSONString(
		`{"enabled":true,"limits":{"free-model":3}}`,
	))

	rpm, found := GetModelRequestRateLimitRPM("free-model")
	require.True(t, found)
	require.Equal(t, 3, rpm)

	_, found = GetModelRequestRateLimitRPM("Free-Model")
	require.False(t, found)
}
