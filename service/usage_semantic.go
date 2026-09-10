package service

import relaycommon "github.com/QuantumNous/new-api/relay/common"

// InputTokensExcludeCache reports whether the selected channel/model uses an
// upstream input_tokens value that already excludes cached input tokens.
func InputTokensExcludeCache(relayInfo *relaycommon.RelayInfo) bool {
	if relayInfo == nil || relayInfo.ChannelMeta == nil {
		return false
	}
	settings := relayInfo.ChannelOtherSettings
	for _, modelName := range []string{
		relayInfo.UpstreamModelName,
		relayInfo.OriginModelName,
		relayInfo.RequestedModelName,
	} {
		if settings.IsInputTokensExcludeCacheModel(modelName) {
			return true
		}
	}
	return false
}
