package controller

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
)

type emptyResponseRefundUpdateRequest struct {
	Mode                  string   `json:"mode"`
	Models                []string `json:"models"`
	CustomResponseEnabled bool     `json:"custom_response_enabled"`
	CustomResponseText    string   `json:"custom_response_text"`
}

func UpdateEmptyResponseRefundSetting(c *gin.Context) {
	var request emptyResponseRefundUpdateRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的参数",
		})
		return
	}

	if request.Mode != operation_setting.EmptyResponseRefundModeOff &&
		request.Mode != operation_setting.EmptyResponseRefundModeObserve &&
		request.Mode != operation_setting.EmptyResponseRefundModeRefund {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "空响应自动退款模式只能是 off、observe 或 refund",
		})
		return
	}
	if request.Models == nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "空响应自动退款模型必须是字符串数组",
		})
		return
	}
	if len([]rune(request.CustomResponseText)) > 4000 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "空响应自定义返回词不能超过 4000 个字符",
		})
		return
	}

	normalizedModels := make([]string, 0, len(request.Models))
	seenModels := make(map[string]struct{}, len(request.Models))
	for _, modelName := range request.Models {
		modelName = strings.TrimSpace(modelName)
		if modelName == "" {
			continue
		}
		if _, exists := seenModels[modelName]; exists {
			continue
		}
		seenModels[modelName] = struct{}{}
		normalizedModels = append(normalizedModels, modelName)
	}
	modelsJSON, err := common.Marshal(normalizedModels)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	values := map[string]string{
		"empty_response_refund_setting.mode":                    request.Mode,
		"empty_response_refund_setting.models":                  string(modelsJSON),
		"empty_response_refund_setting.custom_response_enabled": common.Interface2String(request.CustomResponseEnabled),
		"empty_response_refund_setting.custom_response_text":    request.CustomResponseText,
	}
	if err := model.UpdateOptionsBulk(values); err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}
