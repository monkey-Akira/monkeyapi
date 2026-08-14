package setting

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/common"
)

var ModelRequestRateLimitEnabled = false
var ModelRequestRateLimitDurationMinutes = 1
var ModelRequestRateLimitCount = 0
var ModelRequestRateLimitSuccessCount = 1000
var ModelRequestRateLimitGroup = map[string][2]int{}
var ModelRequestRateLimitMutex sync.RWMutex

const (
	ModelRequestRateLimitScopeAll      = "all"
	ModelRequestRateLimitScopeSelected = "selected"
)

type ModelRequestRateLimitModelsConfig struct {
	Mode   string   `json:"mode"`
	Models []string `json:"models"`
}

var modelRequestRateLimitModels = ModelRequestRateLimitModelsConfig{
	Mode:   ModelRequestRateLimitScopeAll,
	Models: []string{},
}
var modelRequestRateLimitModelsMutex sync.RWMutex

func ModelRequestRateLimitGroup2JSONString() string {
	ModelRequestRateLimitMutex.RLock()
	defer ModelRequestRateLimitMutex.RUnlock()

	jsonBytes, err := common.Marshal(ModelRequestRateLimitGroup)
	if err != nil {
		common.SysLog("error marshalling model ratio: " + err.Error())
	}
	return string(jsonBytes)
}

func UpdateModelRequestRateLimitGroupByJSONString(jsonStr string) error {
	ModelRequestRateLimitMutex.RLock()
	defer ModelRequestRateLimitMutex.RUnlock()

	ModelRequestRateLimitGroup = make(map[string][2]int)
	return common.UnmarshalJsonStr(jsonStr, &ModelRequestRateLimitGroup)
}

func GetGroupRateLimit(group string) (totalCount, successCount int, found bool) {
	ModelRequestRateLimitMutex.RLock()
	defer ModelRequestRateLimitMutex.RUnlock()

	if ModelRequestRateLimitGroup == nil {
		return 0, 0, false
	}

	limits, found := ModelRequestRateLimitGroup[group]
	if !found {
		return 0, 0, false
	}
	return limits[0], limits[1], true
}

func CheckModelRequestRateLimitGroup(jsonStr string) error {
	checkModelRequestRateLimitGroup := make(map[string][2]int)
	err := common.UnmarshalJsonStr(jsonStr, &checkModelRequestRateLimitGroup)
	if err != nil {
		return err
	}
	for group, limits := range checkModelRequestRateLimitGroup {
		if limits[0] < 0 || limits[1] < 1 {
			return fmt.Errorf("group %s has negative rate limit values: [%d, %d]", group, limits[0], limits[1])
		}
		if limits[0] > math.MaxInt32 || limits[1] > math.MaxInt32 {
			return fmt.Errorf("group %s [%d, %d] has max rate limits value 2147483647", group, limits[0], limits[1])
		}
	}

	return nil
}

func NormalizeModelRequestRateLimitModels(jsonStr string) (string, error) {
	isPreviousFormat := false
	var payload struct {
		Mode    string         `json:"mode"`
		Models  []string       `json:"models"`
		Enabled *bool          `json:"enabled"`
		Limits  map[string]int `json:"limits"`
	}
	if err := common.UnmarshalJsonStr(jsonStr, &payload); err != nil {
		return "", fmt.Errorf("模型限流范围配置必须是有效的 JSON 对象: %w", err)
	}

	// Compatible with the previous per-model RPM format. The old RPM values are
	// intentionally discarded because selected models now share the global limits.
	if payload.Mode == "" && payload.Enabled != nil && payload.Limits != nil {
		isPreviousFormat = true
		if *payload.Enabled {
			payload.Mode = ModelRequestRateLimitScopeSelected
			payload.Models = make([]string, 0, len(payload.Limits))
			for modelName := range payload.Limits {
				payload.Models = append(payload.Models, modelName)
			}
		} else {
			payload.Mode = ModelRequestRateLimitScopeAll
			payload.Models = []string{}
		}
	}

	if payload.Mode != ModelRequestRateLimitScopeAll && payload.Mode != ModelRequestRateLimitScopeSelected {
		return "", fmt.Errorf("模型限流范围必须是 all 或 selected")
	}

	normalizedModels := make([]string, 0, len(payload.Models))
	seenModels := make(map[string]struct{}, len(payload.Models))
	for _, rawModelName := range payload.Models {
		modelName := strings.TrimSpace(rawModelName)
		if modelName == "" {
			return "", fmt.Errorf("模型限流范围中的模型名不能为空")
		}
		if _, exists := seenModels[modelName]; exists {
			return "", fmt.Errorf("模型限流范围包含重复模型: %s", modelName)
		}
		seenModels[modelName] = struct{}{}
		normalizedModels = append(normalizedModels, modelName)
	}
	if payload.Mode == ModelRequestRateLimitScopeSelected && len(normalizedModels) == 0 && !isPreviousFormat {
		return "", fmt.Errorf("选择指定模型限流时至少需要选择一个模型")
	}
	if payload.Mode == ModelRequestRateLimitScopeAll {
		normalizedModels = []string{}
	}
	sort.Strings(normalizedModels)

	config := ModelRequestRateLimitModelsConfig{
		Mode:   payload.Mode,
		Models: normalizedModels,
	}

	jsonBytes, err := common.Marshal(config)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

func ModelRequestRateLimitModels2JSONString() string {
	modelRequestRateLimitModelsMutex.RLock()
	defer modelRequestRateLimitModelsMutex.RUnlock()

	jsonBytes, err := common.Marshal(modelRequestRateLimitModels)
	if err != nil {
		common.SysLog("error marshalling model request rate limit scope: " + err.Error())
		return `{"mode":"all","models":[]}`
	}
	return string(jsonBytes)
}

func UpdateModelRequestRateLimitModelsByJSONString(jsonStr string) error {
	normalizedJSON, err := NormalizeModelRequestRateLimitModels(jsonStr)
	if err != nil {
		return err
	}

	var config ModelRequestRateLimitModelsConfig
	if err := common.UnmarshalJsonStr(normalizedJSON, &config); err != nil {
		return err
	}

	modelRequestRateLimitModelsMutex.Lock()
	modelRequestRateLimitModels = config
	modelRequestRateLimitModelsMutex.Unlock()
	return nil
}

func ModelRequestRateLimitAppliesToAllModels() bool {
	modelRequestRateLimitModelsMutex.RLock()
	defer modelRequestRateLimitModelsMutex.RUnlock()

	return modelRequestRateLimitModels.Mode != ModelRequestRateLimitScopeSelected
}

func ShouldApplyModelRequestRateLimit(modelName string) bool {
	modelRequestRateLimitModelsMutex.RLock()
	defer modelRequestRateLimitModelsMutex.RUnlock()

	if modelRequestRateLimitModels.Mode != ModelRequestRateLimitScopeSelected {
		return true
	}
	for _, selectedModel := range modelRequestRateLimitModels.Models {
		if selectedModel == modelName {
			return true
		}
	}
	return false
}
