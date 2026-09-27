package model

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

// PricingValues contains one model's settings, keyed by the existing option
// names. Numeric zero values remain explicit and are never treated as absent.
type PricingValues map[string]any

type ModelPricingChange struct {
	ModelName       string        `json:"model_name"`
	ExpectedVersion string        `json:"expected_version"`
	Pricing         PricingValues `json:"pricing"`
	Reset           bool          `json:"reset,omitempty"`
}

type ModelPricingDescription struct {
	Effective      PricingValues `json:"effective"`
	CacheWriteMode string        `json:"cache_write_mode,omitempty"`
}

type ModelPricingEntry struct {
	ModelPricingDescription
	ModelName  string        `json:"model_name"`
	Version    string        `json:"version"`
	Configured PricingValues `json:"configured"`
}

type ModelPricingSnapshot struct {
	Entries      []ModelPricingEntry `json:"entries"`
	Options      map[string]string   `json:"options"`
	EmptyVersion string              `json:"empty_version"`
}

type ModelPricingConversion struct {
	ModelPricingDescription
	Expression        string `json:"expression,omitempty"`
	UnsupportedReason string `json:"unsupported_reason,omitempty"`
}

var ErrModelPricingConflict = errors.New("model pricing changed; reload before saving")

var modelPricingOptionKeys = []string{
	"ModelPrice",
	"ModelRatio",
	"CompletionRatio",
	"CacheRatio",
	"CreateCacheRatio",
	"ImageRatio",
	"AudioRatio",
	"AudioCompletionRatio",
	"billing_setting.billing_mode",
	"billing_setting.billing_expr",
	billing_setting.PluginBillingExprOption,
}

var modelPricingMutationMu sync.Mutex

func modelPricingVersion(values PricingValues) string {
	encoded, _ := common.Marshal(values)
	return fmt.Sprintf("%x", sha256.Sum256(encoded))
}

func readModelPricingMaps() (map[string]map[string]any, map[string]string, error) {
	common.OptionMapRWMutex.RLock()
	raw := make(map[string]string, len(modelPricingOptionKeys))
	for _, key := range modelPricingOptionKeys {
		raw[key] = common.OptionMap[key]
	}
	common.OptionMapRWMutex.RUnlock()

	values := make(map[string]map[string]any, len(modelPricingOptionKeys))
	for _, key := range modelPricingOptionKeys {
		encoded := raw[key]
		if strings.TrimSpace(encoded) == "" {
			encoded = "{}"
		}
		entries := make(map[string]any)
		if err := common.UnmarshalJsonStr(encoded, &entries); err != nil {
			return nil, nil, fmt.Errorf("%s: %w", key, err)
		}
		if entries == nil {
			return nil, nil, fmt.Errorf("%s must be a JSON object", key)
		}
		values[key] = entries
		normalized, err := common.Marshal(entries)
		if err != nil {
			return nil, nil, err
		}
		raw[key] = string(normalized)
	}
	return values, raw, nil
}

func modelPricingValues(values map[string]map[string]any, name string) PricingValues {
	result := make(PricingValues)
	for _, key := range modelPricingOptionKeys {
		if key == billing_setting.PluginBillingExprOption {
			variants := make(map[string]any)
			for variant, expression := range values[key] {
				plugin, modelName, ok := billing_setting.SplitPluginBillingExprKey(variant)
				if ok && modelName == name {
					variants[plugin] = expression
				}
			}
			if len(variants) > 0 {
				result[key] = variants
			}
			continue
		}
		if value, ok := values[key][name]; ok {
			result[key] = value
		}
	}
	return result
}

func defaultModelPricing(name string) PricingValues {
	result := make(PricingValues)
	for key, entries := range ratio_setting.GetDefaultPricingMaps() {
		if value, ok := entries[name]; ok {
			result[key] = value
		}
	}
	if len(result) > 0 {
		result["billing_setting.billing_mode"] = billing_setting.BillingModeRatio
	}
	return result
}

func replaceModelPricing(values map[string]map[string]any, name string, draft PricingValues) {
	for _, key := range modelPricingOptionKeys {
		if key == billing_setting.PluginBillingExprOption {
			for variant := range values[key] {
				_, modelName, ok := billing_setting.SplitPluginBillingExprKey(variant)
				if ok && modelName == name {
					delete(values[key], variant)
				}
			}
			if variants, ok := draft[key].(map[string]any); ok {
				for plugin, expression := range variants {
					values[key][billing_setting.PluginBillingExprKey(plugin, name)] = expression
				}
			}
			continue
		}
		delete(values[key], name)
		if value, ok := draft[key]; ok {
			values[key][name] = value
		}
	}
}

func effectiveModelPricing(values map[string]map[string]any, name string) PricingValues {
	result := modelPricingValues(values, name)
	alias := ratio_setting.FormatMatchingModelName(name)
	for _, key := range []string{
		"ModelPrice",
		"ModelRatio",
		"CompletionRatio",
		"AudioRatio",
		"AudioCompletionRatio",
	} {
		delete(result, key)
		if value, exists := values[key][alias]; exists {
			result[key] = value
		}
	}
	if result["billing_setting.billing_mode"] == billing_setting.BillingModeTieredExpr {
		return result
	}
	if _, fixed := result["ModelPrice"]; fixed {
		return result
	}
	if _, exists := result["ModelRatio"]; !exists && operation_setting.SelfUseModeEnabled {
		result["ModelRatio"] = float64(37.5)
	}
	var configuredCompletion *float64
	if ratio, exists := result["CompletionRatio"].(float64); exists {
		configuredCompletion = &ratio
	}
	result["CompletionRatio"] = ratio_setting.ResolveModelPricingCompletionRatio(name, configuredCompletion)
	for key, fallback := range map[string]float64{
		"CacheRatio":       1,
		"CreateCacheRatio": 1.25,
		"ImageRatio":       1,
	} {
		if _, exists := result[key]; !exists {
			result[key] = fallback
		}
	}
	return result
}
func cacheWriteMode(name string, configured PricingValues) string {
	if strings.Contains(strings.ToLower(name), "claude") {
		return "claude_ttl"
	}
	if _, ok := configured["CreateCacheRatio"]; ok {
		return "standard"
	}
	return "none"
}

func GetModelPricingSnapshot(names []string) (*ModelPricingSnapshot, error) {
	values, options, err := readModelPricingMaps()
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		nameSet := make(map[string]bool)
		for key, entries := range values {
			for name := range entries {
				if key == billing_setting.PluginBillingExprOption {
					_, modelName, ok := billing_setting.SplitPluginBillingExprKey(name)
					if !ok {
						continue
					}
					name = modelName
				}
				nameSet[name] = true
			}
		}
		for name := range nameSet {
			names = append(names, name)
		}
	}
	names = uniqueSortedStrings(names)
	result := &ModelPricingSnapshot{
		Entries:      make([]ModelPricingEntry, 0, len(names)),
		Options:      options,
		EmptyVersion: modelPricingVersion(PricingValues{}),
	}
	for _, name := range names {
		configured := modelPricingValues(values, name)
		result.Entries = append(result.Entries, ModelPricingEntry{
			ModelName:  name,
			Version:    modelPricingVersion(configured),
			Configured: configured,
			ModelPricingDescription: ModelPricingDescription{
				Effective:      effectiveModelPricing(values, name),
				CacheWriteMode: cacheWriteMode(name, configured),
			},
		})
	}
	return result, nil
}

func uniqueSortedStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func validateModelPricing(name string, values PricingValues) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("model name is required")
	}
	for key, value := range values {
		supported := false
		for _, candidate := range modelPricingOptionKeys {
			if key == candidate {
				supported = true
				break
			}
		}
		if !supported {
			return fmt.Errorf("unsupported pricing field: %s", key)
		}
		switch key {
		case "billing_setting.billing_mode":
			if value != billing_setting.BillingModeRatio && value != billing_setting.BillingModeTieredExpr {
				return errors.New("invalid billing mode")
			}
		case "billing_setting.billing_expr":
			expression, ok := value.(string)
			if !ok || strings.TrimSpace(expression) == "" {
				return errors.New("billing expression is required")
			}
			if err := billing_setting.SmokeTestExpr(expression); err != nil {
				return fmt.Errorf("model %s: %w", name, err)
			}
		case billing_setting.PluginBillingExprOption:
			variants, ok := value.(map[string]any)
			if !ok {
				return errors.New("plugin billing expressions must be an object")
			}
			for plugin, expression := range variants {
				if strings.TrimSpace(plugin) == "" {
					return errors.New("plugin billing expression key is required")
				}
				if text, ok := expression.(string); !ok || strings.TrimSpace(text) == "" {
					return fmt.Errorf("plugin %s billing expression is required", plugin)
				}
			}
		default:
			number, ok := value.(float64)
			if !ok || math.IsNaN(number) || math.IsInf(number, 0) || number < 0 {
				return fmt.Errorf("%s must be a finite, non-negative number", key)
			}
		}
	}
	if values["billing_setting.billing_mode"] == billing_setting.BillingModeTieredExpr {
		if _, ok := values["billing_setting.billing_expr"]; !ok {
			return errors.New("billing expression is required")
		}
	}
	return nil
}

func UpdateModelPricing(changes []ModelPricingChange) error {
	if len(changes) == 0 {
		return errors.New("select model pricing changes before saving")
	}
	modelPricingMutationMu.Lock()
	defer modelPricingMutationMu.Unlock()

	values, _, err := readModelPricingMaps()
	if err != nil {
		return err
	}
	seen := make(map[string]bool, len(changes))
	for _, change := range changes {
		if seen[change.ModelName] {
			return errors.New("duplicate model pricing change")
		}
		seen[change.ModelName] = true
		previous := modelPricingValues(values, change.ModelName)
		if change.ExpectedVersion == "" || modelPricingVersion(previous) != change.ExpectedVersion {
			return fmt.Errorf("%w: %s", ErrModelPricingConflict, change.ModelName)
		}
		next := change.Pricing
		if change.Reset {
			next = defaultModelPricing(change.ModelName)
		}
		if err := validateModelPricing(change.ModelName, next); err != nil {
			return err
		}
		replaceModelPricing(values, change.ModelName, next)
	}
	updates := make(map[string]string, len(modelPricingOptionKeys))
	for _, key := range modelPricingOptionKeys {
		encoded, err := common.Marshal(values[key])
		if err != nil {
			return err
		}
		updates[key] = string(encoded)
	}
	return UpdateOptionsBulk(updates)
}

func PreviewModelPricing(name string, draft PricingValues) (ModelPricingDescription, error) {
	if draft == nil {
		return ModelPricingDescription{}, errors.New("pricing draft is required")
	}
	values, _, err := readModelPricingMaps()
	if err != nil {
		return ModelPricingDescription{}, err
	}
	if err := validateModelPricing(name, draft); err != nil {
		return ModelPricingDescription{}, err
	}
	replaceModelPricing(values, name, draft)
	configured := modelPricingValues(values, name)
	return ModelPricingDescription{
		Effective:      effectiveModelPricing(values, name),
		CacheWriteMode: cacheWriteMode(name, configured),
	}, nil
}
func PreviewModelPricingConversion(name string, draft PricingValues) (*ModelPricingConversion, error) {
	preview, err := PreviewModelPricing(name, draft)
	if err != nil {
		return nil, err
	}
	if draft["billing_setting.billing_mode"] == billing_setting.BillingModeTieredExpr {
		return &ModelPricingConversion{UnsupportedReason: "This model already uses an expression."}, nil
	}
	if _, fixed := preview.Effective["ModelPrice"]; fixed {
		return &ModelPricingConversion{UnsupportedReason: "Per-request pricing must be converted manually."}, nil
	}
	ratio, ok := numberValue(preview.Effective["ModelRatio"])
	if !ok {
		return &ModelPricingConversion{UnsupportedReason: "Configure an input price before converting this model."}, nil
	}
	if common.QuotaPerUnit <= 0 {
		return nil, errors.New("invalid quota unit")
	}
	base := ratio * 1_000_000 / common.QuotaPerUnit
	terms := []string{"p * " + pricingNumber(base)}
	lanes := []struct {
		variable string
		key      string
	}{
		{variable: "c", key: "CompletionRatio"},
		{variable: "cr", key: "CacheRatio"},
		{variable: "cc", key: "CreateCacheRatio"},
		{variable: "img", key: "ImageRatio"},
		{variable: "ai", key: "AudioRatio"},
	}
	for _, lane := range lanes {
		if lane.variable != "c" {
			if _, explicitlyConfigured := draft[lane.key]; !explicitlyConfigured {
				continue
			}
		}
		multiplier, exists := numberValue(preview.Effective[lane.key])
		if !exists {
			continue
		}
		terms = append(terms, lane.variable+" * "+pricingNumber(base*multiplier))
		if lane.variable == "cc" && preview.CacheWriteMode == "claude_ttl" {
			terms = append(terms, "cc1h * "+pricingNumber(base*multiplier*6/3.75))
		}
	}
	if audioInput, exists := numberValue(draft["AudioRatio"]); exists {
		outputRatio := float64(1)
		if configured, ok := numberValue(preview.Effective["AudioCompletionRatio"]); ok {
			outputRatio = configured
		}
		terms = append(terms, "ao * "+pricingNumber(base*audioInput*outputRatio))
	}
	expression := `tier("base", ` + strings.Join(terms, " + ") + `)`
	if err := billing_setting.SmokeTestExpr(expression); err != nil {
		return nil, err
	}
	return &ModelPricingConversion{
		ModelPricingDescription: preview,
		Expression:              expression,
	}, nil
}

func numberValue(value any) (float64, bool) {
	number, ok := value.(float64)
	return number, ok && !math.IsNaN(number) && !math.IsInf(number, 0)
}

func pricingNumber(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
