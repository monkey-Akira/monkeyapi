package ratio_setting

import (
	"maps"
	"strings"
)

// GetDefaultPricingMaps returns isolated copies of the built-in legacy pricing
// maps so model-level edits can restore one model without resetting all prices.
func GetDefaultPricingMaps() map[string]map[string]float64 {
	return map[string]map[string]float64{
		"ModelPrice":           maps.Clone(defaultModelPrice),
		"ModelRatio":           maps.Clone(defaultModelRatio),
		"CompletionRatio":      maps.Clone(defaultCompletionRatio),
		"CacheRatio":           maps.Clone(defaultCacheRatio),
		"CreateCacheRatio":     maps.Clone(defaultCreateCacheRatio),
		"ImageRatio":           maps.Clone(defaultImageRatio),
		"AudioRatio":           maps.Clone(defaultAudioRatio),
		"AudioCompletionRatio": maps.Clone(defaultAudioCompletionRatio),
	}
}

// ResolveModelPricingCompletionRatio applies the relay's hard-coded completion
// ratio rules to an isolated pricing draft without reading the live option map.
func ResolveModelPricingCompletionRatio(name string, configured *float64) float64 {
	name = FormatMatchingModelName(name)
	if strings.Contains(name, "/") && configured != nil {
		return *configured
	}
	if ratio, exists := getHardcodedCompletionModelRatio(name); exists {
		return ratio
	}
	if configured != nil {
		return *configured
	}
	return 1
}
