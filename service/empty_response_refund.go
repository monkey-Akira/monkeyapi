package service

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
)

func isTextGenerationForEmptyResponseRefund(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, usage *dto.Usage) bool {
	if ctx == nil || relayInfo == nil || usage == nil || ctx.GetBool("image_generation_call") {
		return false
	}
	format := relayInfo.GetFinalRequestRelayFormat()
	if format == "" {
		format = relayInfo.RelayFormat
	}
	switch format {
	case types.RelayFormatOpenAI, types.RelayFormatClaude, types.RelayFormatGemini, types.RelayFormatOpenAIResponses:
	default:
		return false
	}
	if relayInfo.Request == nil {
		return false
	}

	if usage.PromptTokensDetails.AudioTokens > 0 ||
		usage.CompletionTokenDetails.ImageTokens > 0 ||
		usage.CompletionTokenDetails.AudioTokens > 0 ||
		usage.PromptTokensDetails.ImageTokens > 0 ||
		usage.PromptTokensDetails.AudioTokens > 0 ||
		usage.InputTokensDetails != nil && (usage.InputTokensDetails.AudioTokens > 0 || usage.InputTokensDetails.ImageTokens > 0) ||
		usage.OutputTokensDetails != nil && (usage.OutputTokensDetails.AudioTokens > 0 || usage.OutputTokensDetails.ImageTokens > 0) {
		return false
	}

	switch request := relayInfo.Request.(type) {
	case *dto.GeneralOpenAIRequest:
		if request == nil {
			return false
		}
		if len(request.Modalities) > 0 {
			var modalities []string
			if err := common.Unmarshal(request.Modalities, &modalities); err != nil {
				return false
			}
			for _, modality := range modalities {
				if !strings.EqualFold(strings.TrimSpace(modality), "text") {
					return false
				}
			}
		}
		if request.ResponseFormat != nil &&
			((request.ResponseFormat.Type != "" && !strings.EqualFold(request.ResponseFormat.Type, "text")) || len(request.ResponseFormat.JsonSchema) > 0) {
			return false
		}
		if isForcedToolChoice(request.ToolChoice) || rawToolChoiceIsForced(request.FunctionCall) {
			return false
		}
	case *dto.OpenAIResponsesRequest:
		if request == nil || responsesRequestUsesImageGeneration(request) || responsesRequestUsesStructuredText(request.Text) || rawToolChoiceIsForced(request.ToolChoice) {
			return false
		}
	case *dto.ClaudeRequest:
		if request == nil || len(request.OutputConfig) > 0 || len(request.OutputFormat) > 0 || isForcedToolChoice(request.ToolChoice) {
			return false
		}
	case *dto.GeminiChatRequest:
		if request == nil {
			return false
		}
		for _, modality := range request.GenerationConfig.ResponseModalities {
			if !strings.EqualFold(strings.TrimSpace(modality), "text") {
				return false
			}
		}
		if request.GenerationConfig.ResponseMimeType != "" &&
			!strings.EqualFold(request.GenerationConfig.ResponseMimeType, "text/plain") {
			return false
		}
		if request.GenerationConfig.ResponseSchema != nil || len(request.GenerationConfig.ResponseJsonSchema) > 0 {
			return false
		}
		if request.ToolConfig != nil && request.ToolConfig.FunctionCallingConfig != nil &&
			strings.EqualFold(string(request.ToolConfig.FunctionCallingConfig.Mode), "ANY") {
			return false
		}
	}

	return !model_setting.IsGeminiModelSupportImagine(relayInfo.RequestedModelName) &&
		!model_setting.IsGeminiModelSupportImagine(relayInfo.OriginModelName) &&
		!model_setting.IsGeminiModelSupportImagine(relayInfo.GetUpstreamModelName())
}

func responsesRequestUsesImageGeneration(request *dto.OpenAIResponsesRequest) bool {
	if request == nil || len(request.Tools) == 0 {
		return false
	}
	var tools []map[string]any
	if err := common.Unmarshal(request.Tools, &tools); err != nil {
		return true
	}
	for _, tool := range tools {
		if strings.EqualFold(strings.TrimSpace(common.Interface2String(tool["type"])), "image_generation") {
			return true
		}
	}
	return false
}

func getEmptyResponseRefundMode(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, usage *dto.Usage) string {
	if relayInfo == nil || usage == nil || relayInfo.RequestedZeroMaxOutput ||
		usage.CompletionTokens != 0 || usage.OutputTokens != 0 ||
		!isTextGenerationForEmptyResponseRefund(ctx, relayInfo, usage) {
		return operation_setting.EmptyResponseRefundModeOff
	}
	requestedModelName := relayInfo.RequestedModelName
	if requestedModelName == "" {
		requestedModelName = relayInfo.OriginModelName
	}
	return operation_setting.GetEmptyResponseRefundMode(requestedModelName)
}

func emptyResponseRefundRequestAllowsCustomText(relayInfo *relaycommon.RelayInfo) bool {
	if relayInfo == nil {
		return false
	}
	switch request := relayInfo.Request.(type) {
	case *dto.GeneralOpenAIRequest:
		if request == nil {
			return false
		}
		if relayInfo.RelayMode == relayconstant.RelayModeCompletions {
			return false
		}
		if request.ResponseFormat != nil &&
			((request.ResponseFormat.Type != "" && !strings.EqualFold(request.ResponseFormat.Type, "text")) || len(request.ResponseFormat.JsonSchema) > 0) {
			return false
		}
		return !isForcedToolChoice(request.ToolChoice) && !rawToolChoiceIsForced(request.FunctionCall)
	case *dto.OpenAIResponsesRequest:
		if request == nil {
			return false
		}
		if responsesRequestUsesStructuredText(request.Text) {
			return false
		}
		return !rawToolChoiceIsForced(request.ToolChoice)
	case *dto.ClaudeRequest:
		if request == nil {
			return false
		}
		if len(request.OutputConfig) > 0 || len(request.OutputFormat) > 0 {
			return false
		}
		return !isForcedToolChoice(request.ToolChoice)
	case *dto.GeminiChatRequest:
		if request == nil {
			return false
		}
		if request.GenerationConfig.ResponseMimeType != "" &&
			!strings.EqualFold(request.GenerationConfig.ResponseMimeType, "text/plain") {
			return false
		}
		if request.GenerationConfig.ResponseSchema != nil || len(request.GenerationConfig.ResponseJsonSchema) > 0 {
			return false
		}
		return request.ToolConfig == nil || request.ToolConfig.FunctionCallingConfig == nil ||
			!strings.EqualFold(string(request.ToolConfig.FunctionCallingConfig.Mode), "ANY")
	default:
		return false
	}
}

func isForcedToolChoice(toolChoice any) bool {
	if toolChoice == nil {
		return false
	}
	if value, ok := toolChoice.(string); ok {
		value = strings.TrimSpace(value)
		return value != "" && !strings.EqualFold(value, "auto") && !strings.EqualFold(value, "none")
	}
	data, err := common.Marshal(toolChoice)
	return err != nil || rawToolChoiceIsForced(data)
}

func rawToolChoiceIsForced(value []byte) bool {
	trimmed := strings.TrimSpace(string(value))
	if trimmed == "" || trimmed == "null" {
		return false
	}
	var stringValue string
	if err := common.Unmarshal(value, &stringValue); err == nil {
		return isForcedToolChoice(stringValue)
	}
	var objectValue map[string]any
	if err := common.Unmarshal(value, &objectValue); err != nil {
		return true
	}
	choiceType := strings.TrimSpace(common.Interface2String(objectValue["type"]))
	return choiceType == "" || (!strings.EqualFold(choiceType, "auto") && !strings.EqualFold(choiceType, "none"))
}

func responsesRequestUsesStructuredText(value []byte) bool {
	if len(value) == 0 {
		return false
	}
	var textConfig map[string]any
	if err := common.Unmarshal(value, &textConfig); err != nil {
		return true
	}
	format, ok := textConfig["format"].(map[string]any)
	if !ok || format == nil {
		return false
	}
	formatType := strings.TrimSpace(common.Interface2String(format["type"]))
	return formatType != "" && !strings.EqualFold(formatType, "text")
}

func GetEmptyResponseRefundCustomText(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, usage *dto.Usage, hasNonTextOutput bool) string {
	if hasNonTextOutput || !emptyResponseRefundRequestAllowsCustomText(relayInfo) {
		return ""
	}
	if getEmptyResponseRefundMode(ctx, relayInfo, usage) != operation_setting.EmptyResponseRefundModeRefund {
		return ""
	}
	customText := operation_setting.GetEmptyResponseRefundCustomText()
	if strings.TrimSpace(customText) == "" {
		return ""
	}
	return customText
}
