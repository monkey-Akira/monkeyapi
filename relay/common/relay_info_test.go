package common

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/require"
)

func TestRelayInfoGetFinalRequestRelayFormatPrefersExplicitFinal(t *testing.T) {
	info := &RelayInfo{
		RelayFormat:             types.RelayFormatOpenAI,
		RequestConversionChain:  []types.RelayFormat{types.RelayFormatOpenAI, types.RelayFormatClaude},
		FinalRequestRelayFormat: types.RelayFormatOpenAIResponses,
	}

	require.Equal(t, types.RelayFormat(types.RelayFormatOpenAIResponses), info.GetFinalRequestRelayFormat())
}

func TestRelayInfoGetFinalRequestRelayFormatFallsBackToConversionChain(t *testing.T) {
	info := &RelayInfo{
		RelayFormat:            types.RelayFormatOpenAI,
		RequestConversionChain: []types.RelayFormat{types.RelayFormatOpenAI, types.RelayFormatClaude},
	}

	require.Equal(t, types.RelayFormat(types.RelayFormatClaude), info.GetFinalRequestRelayFormat())
}

func TestRelayInfoGetFinalRequestRelayFormatFallsBackToRelayFormat(t *testing.T) {
	info := &RelayInfo{
		RelayFormat: types.RelayFormatGemini,
	}

	require.Equal(t, types.RelayFormat(types.RelayFormatGemini), info.GetFinalRequestRelayFormat())
}

func TestRelayInfoGetFinalRequestRelayFormatNilReceiver(t *testing.T) {
	var info *RelayInfo
	require.Equal(t, types.RelayFormat(""), info.GetFinalRequestRelayFormat())
}

func TestRequestedZeroMaxOutputUsesEffectiveOpenAIField(t *testing.T) {
	zero := uint(0)
	nonZero := uint(128)
	require.True(t, requestedZeroMaxOutput(&dto.GeneralOpenAIRequest{MaxTokens: &zero}))
	require.True(t, requestedZeroMaxOutput(&dto.GeneralOpenAIRequest{MaxCompletionTokens: &zero}))
	require.False(t, requestedZeroMaxOutput(&dto.GeneralOpenAIRequest{
		MaxTokens:           &zero,
		MaxCompletionTokens: &nonZero,
	}))
}

func TestRequestedZeroMaxOutputSupportsResponsesClaudeAndGemini(t *testing.T) {
	zero := uint(0)
	require.True(t, requestedZeroMaxOutput(&dto.OpenAIResponsesRequest{MaxOutputTokens: &zero}))
	require.True(t, requestedZeroMaxOutput(&dto.ClaudeRequest{MaxTokens: &zero}))
	require.True(t, requestedZeroMaxOutput(&dto.GeminiChatRequest{
		GenerationConfig: dto.GeminiChatGenerationConfig{MaxOutputTokens: &zero},
	}))
}
