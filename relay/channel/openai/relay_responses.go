package openai

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

func OaiResponsesHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	defer service.CloseResponseBodyGracefully(resp)

	// read response body
	var responsesResponse dto.OpenAIResponsesResponse
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}
	err = common.Unmarshal(responseBody, &responsesResponse)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	if oaiError := responsesResponse.GetOpenAIError(); oaiError != nil && oaiError.Type != "" {
		return nil, types.WithOpenAIError(*oaiError, resp.StatusCode)
	}

	if responsesResponse.HasImageGenerationCall() {
		c.Set("image_generation_call", true)
		c.Set("image_generation_call_quality", responsesResponse.GetQuality())
		c.Set("image_generation_call_size", responsesResponse.GetSize())
	}

	// compute usage
	usage := dto.Usage{}
	if responsesResponse.Usage != nil {
		usage.PromptTokens = responsesResponse.Usage.InputTokens
		usage.CompletionTokens = responsesResponse.Usage.OutputTokens
		usage.TotalTokens = responsesResponse.Usage.TotalTokens
		if responsesResponse.Usage.InputTokensDetails != nil {
			usage.PromptTokensDetails.CachedTokens = responsesResponse.Usage.InputTokensDetails.CachedTokens
		}
	}
	if customText := service.GetEmptyResponseRefundCustomText(c, info, &usage, responsesResponse.HasImageGenerationCall() || responsesResponseHasNonTextOutput(&responsesResponse)); customText != "" && responsesResponseText(&responsesResponse) == "" {
		responsesResponse.Output = []dto.ResponsesOutput{buildResponsesTextOutput(
			"msg_"+strings.TrimPrefix(helper.GetResponseID(c), "chatcmpl-"),
			customText,
		)}
		responseBody, err = common.Marshal(responsesResponse)
		if err != nil {
			return nil, types.NewOpenAIError(err, types.ErrorCodeJsonMarshalFailed, http.StatusInternalServerError)
		}
	}

	// 写入新的 response body
	service.IOCopyBytesGracefully(c, resp, responseBody)
	if info == nil || info.ResponsesUsageInfo == nil || info.ResponsesUsageInfo.BuiltInTools == nil {
		return &usage, nil
	}
	// 解析 Tools 用量
	for _, tool := range responsesResponse.Tools {
		buildToolinfo, ok := info.ResponsesUsageInfo.BuiltInTools[common.Interface2String(tool["type"])]
		if !ok || buildToolinfo == nil {
			logger.LogError(c, fmt.Sprintf("BuiltInTools not found for tool type: %v", tool["type"]))
			continue
		}
		buildToolinfo.CallCount++
	}
	return &usage, nil
}

func responsesResponseText(response *dto.OpenAIResponsesResponse) string {
	var text strings.Builder
	for _, output := range response.Output {
		for _, content := range output.Content {
			if content.Type == "output_text" {
				text.WriteString(content.Text)
			}
		}
	}
	return text.String()
}

func responsesResponseHasNonTextOutput(response *dto.OpenAIResponsesResponse) bool {
	for _, output := range response.Output {
		if output.Type != "" && output.Type != "message" {
			return true
		}
		for _, content := range output.Content {
			if content.Type != "" && content.Type != "output_text" {
				return true
			}
		}
	}
	return false
}

func OaiResponsesStreamHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	if resp == nil || resp.Body == nil {
		logger.LogError(c, "invalid response or response body")
		return nil, types.NewError(fmt.Errorf("invalid response"), types.ErrorCodeBadResponse)
	}

	defer service.CloseResponseBodyGracefully(resp)

	var usage = &dto.Usage{}
	var responseTextBuilder strings.Builder
	var completedStreamResponse *dto.ResponsesStreamResponse
	var completedStreamData string
	var completedSequenceNumber int
	var hasNonTextOutput bool

	helper.StreamScannerHandler(c, resp, info, func(data string, sr *helper.StreamResult) {

		// 检查当前数据是否包含 completed 状态和 usage 信息
		var streamResponse dto.ResponsesStreamResponse
		if err := common.UnmarshalJsonStr(data, &streamResponse); err != nil {
			logger.LogError(c, "failed to unmarshal stream response: "+err.Error())
			sr.Error(err)
			return
		}
		if streamResponse.Type == "response.completed" {
			completedStreamResponse = &streamResponse
			completedStreamData = data
			var sequenceEnvelope struct {
				SequenceNumber int `json:"sequence_number"`
			}
			if common.UnmarshalJsonStr(data, &sequenceEnvelope) == nil {
				completedSequenceNumber = sequenceEnvelope.SequenceNumber
			}
		} else {
			sendResponsesStreamData(c, streamResponse, data)
		}
		switch streamResponse.Type {
		case "response.completed":
			if streamResponse.Response != nil {
				if streamResponse.Response.Usage != nil {
					if streamResponse.Response.Usage.InputTokens != 0 {
						usage.PromptTokens = streamResponse.Response.Usage.InputTokens
					}
					if streamResponse.Response.Usage.OutputTokens != 0 {
						usage.CompletionTokens = streamResponse.Response.Usage.OutputTokens
					}
					if streamResponse.Response.Usage.TotalTokens != 0 {
						usage.TotalTokens = streamResponse.Response.Usage.TotalTokens
					}
					if streamResponse.Response.Usage.InputTokensDetails != nil {
						usage.PromptTokensDetails.CachedTokens = streamResponse.Response.Usage.InputTokensDetails.CachedTokens
					}
				}
				if streamResponse.Response.HasImageGenerationCall() {
					c.Set("image_generation_call", true)
					c.Set("image_generation_call_quality", streamResponse.Response.GetQuality())
					c.Set("image_generation_call_size", streamResponse.Response.GetSize())
				}
				hasNonTextOutput = hasNonTextOutput || responsesResponseHasNonTextOutput(streamResponse.Response)
			}
		case "response.output_text.delta":
			// 处理输出文本
			responseTextBuilder.WriteString(streamResponse.Delta)
		case dto.ResponsesOutputTypeItemDone:
			// 函数调用处理
			if streamResponse.Item != nil {
				if streamResponse.Item.Type != "" && streamResponse.Item.Type != "message" {
					hasNonTextOutput = true
				}
				switch streamResponse.Item.Type {
				case dto.BuildInCallWebSearchCall:
					if info != nil && info.ResponsesUsageInfo != nil && info.ResponsesUsageInfo.BuiltInTools != nil {
						if webSearchTool, exists := info.ResponsesUsageInfo.BuiltInTools[dto.BuildInToolWebSearchPreview]; exists && webSearchTool != nil {
							webSearchTool.CallCount++
						}
					}
				}
			}
		}
	})

	if usage.CompletionTokens == 0 {
		// 计算输出文本的 token 数量
		tempStr := responseTextBuilder.String()
		if len(tempStr) > 0 {
			// 非正常结束，使用输出文本的 token 数量
			completionTokens := service.CountTextToken(tempStr, info.UpstreamModelName)
			usage.CompletionTokens = completionTokens
		}
	}

	if usage.PromptTokens == 0 && usage.CompletionTokens != 0 {
		usage.PromptTokens = info.GetEstimatePromptTokens()
	}

	usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens

	completedResponseText := ""
	if completedStreamResponse != nil && completedStreamResponse.Response != nil {
		completedResponseText = responsesResponseText(completedStreamResponse.Response)
	}
	customText := service.GetEmptyResponseRefundCustomText(c, info, usage, hasNonTextOutput)
	if customText != "" && responseTextBuilder.Len() == 0 && completedResponseText == "" {
		messageID := "msg_" + strings.TrimPrefix(helper.GetResponseID(c), "chatcmpl-")
		nextSequenceNumber := sendResponsesCustomTextEvents(c, messageID, customText, completedSequenceNumber)
		if completedStreamResponse != nil && completedStreamResponse.Response != nil {
			completedStreamResponse.Response.Output = []dto.ResponsesOutput{buildResponsesTextOutput(messageID, customText)}
			var completedPayload map[string]interface{}
			if common.UnmarshalJsonStr(completedStreamData, &completedPayload) == nil {
				completedPayload["response"] = completedStreamResponse.Response
				completedPayload["sequence_number"] = nextSequenceNumber
				if data, err := common.Marshal(completedPayload); err == nil {
					completedStreamData = string(data)
				}
			}
		}
	}
	if completedStreamResponse != nil {
		sendResponsesStreamData(c, *completedStreamResponse, completedStreamData)
	}

	return usage, nil
}

func buildResponsesTextOutput(messageID string, text string) dto.ResponsesOutput {
	return dto.ResponsesOutput{
		Type:   "message",
		ID:     messageID,
		Status: "completed",
		Role:   "assistant",
		Content: []dto.ResponsesOutputContent{{
			Type:        "output_text",
			Text:        text,
			Annotations: []interface{}{},
		}},
	}
}

func sendResponsesCustomTextEvents(c *gin.Context, messageID string, customText string, sequenceNumber int) int {
	outputItem := buildResponsesTextOutput(messageID, customText)
	outputItem.Status = "in_progress"
	outputItem.Content = []dto.ResponsesOutputContent{}
	completedItem := buildResponsesTextOutput(messageID, customText)
	part := map[string]interface{}{
		"type":        "output_text",
		"text":        customText,
		"annotations": []interface{}{},
	}
	events := []struct {
		typeName string
		payload  map[string]interface{}
	}{
		{"response.output_item.added", map[string]interface{}{"output_index": 0, "item": outputItem}},
		{"response.content_part.added", map[string]interface{}{"item_id": messageID, "output_index": 0, "content_index": 0, "part": map[string]interface{}{"type": "output_text", "text": "", "annotations": []interface{}{}}}},
		{"response.output_text.delta", map[string]interface{}{"item_id": messageID, "output_index": 0, "content_index": 0, "delta": customText}},
		{"response.output_text.done", map[string]interface{}{"item_id": messageID, "output_index": 0, "content_index": 0, "text": customText}},
		{"response.content_part.done", map[string]interface{}{"item_id": messageID, "output_index": 0, "content_index": 0, "part": part}},
		{"response.output_item.done", map[string]interface{}{"output_index": 0, "item": completedItem}},
	}
	for _, event := range events {
		event.payload["type"] = event.typeName
		event.payload["sequence_number"] = sequenceNumber
		sequenceNumber++
		data, err := common.Marshal(event.payload)
		if err != nil {
			continue
		}
		sendResponsesStreamData(c, dto.ResponsesStreamResponse{Type: event.typeName}, string(data))
	}
	return sequenceNumber
}
