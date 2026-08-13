package setting

import (
	"fmt"
	"math"
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

const MaxModelRequestRateLimitRPM = 100000000

type ModelRequestRateLimitModelsConfig struct {
	Enabled bool           `json:"enabled"`
	Limits  map[string]int `json:"limits"`
}

var modelRequestRateLimitModels = ModelRequestRateLimitModelsConfig{
	Enabled: false,
	Limits:  map[string]int{},
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
	var payload struct {
		Enabled *bool          `json:"enabled"`
		Limits  map[string]int `json:"limits"`
	}
	if err := common.UnmarshalJsonStr(jsonStr, &payload); err != nil {
		return "", fmt.Errorf("指定模型限流配置必须是有效的 JSON 对象: %w", err)
	}
	if payload.Enabled == nil || payload.Limits == nil {
		return "", fmt.Errorf("指定模型限流配置必须包含 enabled 和 limits")
	}

	normalizedLimits := make(map[string]int, len(payload.Limits))
	for rawModelName, rpm := range payload.Limits {
		modelName := strings.TrimSpace(rawModelName)
		if modelName == "" {
			return "", fmt.Errorf("指定模型限流的模型名不能为空")
		}
		if _, exists := normalizedLimits[modelName]; exists {
			return "", fmt.Errorf("指定模型限流包含重复模型: %s", modelName)
		}
		if rpm < 1 || rpm > MaxModelRequestRateLimitRPM {
			return "", fmt.Errorf("模型 %s 的每分钟调用次数必须在 1 到 %d 之间", modelName, MaxModelRequestRateLimitRPM)
		}
		normalizedLimits[modelName] = rpm
	}
	config := ModelRequestRateLimitModelsConfig{
		Enabled: *payload.Enabled,
		Limits:  normalizedLimits,
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
		common.SysLog("error marshalling model request rate limits: " + err.Error())
		return `{"enabled":false,"limits":{}}`
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

func GetModelRequestRateLimitRPM(modelName string) (int, bool) {
	modelRequestRateLimitModelsMutex.RLock()
	defer modelRequestRateLimitModelsMutex.RUnlock()

	if !modelRequestRateLimitModels.Enabled {
		return 0, false
	}
	rpm, found := modelRequestRateLimitModels.Limits[modelName]
	return rpm, found && rpm > 0
}
