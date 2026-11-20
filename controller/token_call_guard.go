package controller

import (
	"errors"
	"net/http"

	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
)

func reserveKeyCall(c *gin.Context) (*model.TokenCallSlot, *types.NewAPIError) {
	if c.GetInt("token_id") == 0 {
		return nil, nil
	}
	slot, err := model.ReserveTokenCall(c.GetInt("token_id"))
	if err == nil {
		return slot, nil
	}
	status := http.StatusServiceUnavailable
	switch {
	case errors.Is(err, model.ErrTokenCallCapacity):
		status = http.StatusTooManyRequests
	case errors.Is(err, model.ErrTokenInvalid):
		status = http.StatusForbidden
	default:
		logger.LogError(c, "failed to reserve API key call: "+err.Error())
		err = errors.New("unable to check API key call limit")
	}
	return nil, types.NewErrorWithStatusCode(err, "token_call_limit", status, types.ErrOptionWithSkipRetry())
}

func finishKeyCall(c *gin.Context, slot *model.TokenCallSlot, outcome model.TokenCallOutcome) {
	if err := slot.Finish(outcome); err != nil {
		logger.LogError(c, "failed to finalize API key call: "+err.Error())
	}
}

// Explicit client errors are rejections. Transport errors, server failures and
// interrupted streams may already have incurred cost upstream.
func keyCallOutcome(status int, stream *relaycommon.StreamStatus) model.TokenCallOutcome {
	if status >= 400 && status < 500 && status != http.StatusRequestTimeout && status != 499 {
		return model.CallRejected
	}
	if status < 200 || status >= 400 {
		return model.CallUncertain
	}
	if stream != nil {
		if stream.ResponseFailed() {
			return model.CallRejected
		}
		if !stream.IsNormalEnd() || stream.HasErrors() || (stream.ResponseOutcome() != "" && stream.ResponseOutcome() != "completed") {
			return model.CallUncertain
		}
	}
	return model.CallSucceeded
}
