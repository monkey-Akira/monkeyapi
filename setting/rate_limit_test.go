package setting

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeModelRequestRateLimitModels(t *testing.T) {
	normalized, err := NormalizeModelRequestRateLimitModels(`{
		"mode": "selected",
		"models": [" other-model ", "free-model"]
	}`)
	require.NoError(t, err)
	require.JSONEq(t, `{
		"mode": "selected",
		"models": ["free-model", "other-model"]
	}`, normalized)
}

func TestNormalizeModelRequestRateLimitModelsMigratesPreviousFormat(t *testing.T) {
	normalized, err := NormalizeModelRequestRateLimitModels(`{
		"enabled": true,
		"limits": {"free-model": 10, "other-model": 2}
	}`)
	require.NoError(t, err)
	require.JSONEq(t, `{
		"mode": "selected",
		"models": ["free-model", "other-model"]
	}`, normalized)
}

func TestNormalizeModelRequestRateLimitModelsPreservesEmptyPreviousSelection(t *testing.T) {
	normalized, err := NormalizeModelRequestRateLimitModels(`{
		"enabled": true,
		"limits": {}
	}`)
	require.NoError(t, err)
	require.JSONEq(t, `{
		"mode": "selected",
		"models": []
	}`, normalized)
}

func TestNormalizeModelRequestRateLimitModelsRejectsEmptySelection(t *testing.T) {
	_, err := NormalizeModelRequestRateLimitModels(`{"mode":"selected","models":[]}`)
	require.EqualError(t, err, "选择指定模型限流时至少需要选择一个模型")
}

func TestNormalizeModelRequestRateLimitModelsRequiresValidMode(t *testing.T) {
	_, err := NormalizeModelRequestRateLimitModels(`null`)
	require.EqualError(t, err, "模型限流范围必须是 all 或 selected")

	_, err = NormalizeModelRequestRateLimitModels(`{"mode":"custom","models":[]}`)
	require.EqualError(t, err, "模型限流范围必须是 all 或 selected")
}

func TestShouldApplyModelRequestRateLimitMatchesExactModelName(t *testing.T) {
	previous := ModelRequestRateLimitModels2JSONString()
	t.Cleanup(func() {
		require.NoError(t, UpdateModelRequestRateLimitModelsByJSONString(previous))
	})

	require.NoError(t, UpdateModelRequestRateLimitModelsByJSONString(
		`{"mode":"selected","models":["free-model"]}`,
	))

	require.True(t, ShouldApplyModelRequestRateLimit("free-model"))
	require.False(t, ShouldApplyModelRequestRateLimit("Free-Model"))
	require.False(t, ShouldApplyModelRequestRateLimit("other-model"))

	require.NoError(t, UpdateModelRequestRateLimitModelsByJSONString(
		`{"mode":"all","models":[]}`,
	))
	require.True(t, ShouldApplyModelRequestRateLimit("other-model"))
}
