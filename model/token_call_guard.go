package model

import (
	"errors"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrTokenCallCapacity = errors.New("API key has no available call slots in the current 60-second window")

// FinishedAt is Unix milliseconds. Zero reserves a slot for the entire upstream
// call, including long streams. Only an explicit reset releases orphaned slots.
type TokenCallSlot struct {
	ID string `gorm:"type:varchar(40);primaryKey"`
	TokenID int `gorm:"index:idx_token_call_window,priority:1"`
	FinishedAt int64 `gorm:"index:idx_token_call_window,priority:2"`
	Successful bool
}

type TokenCallOutcome int

const (
	CallUncertain TokenCallOutcome = iota
	CallRejected
	CallSucceeded
)

func lockForTokenCall(tx *gorm.DB) *gorm.DB {
	if common.UsingSQLite {
		return tx
	}
	return tx.Clauses(clause.Locking{Strength: "UPDATE"})
}

func ReserveTokenCall(tokenID int) (*TokenCallSlot, error) {
	// An old Redis snapshot may outlive an edit. Read the guard configuration
	// from the database even for unlimited keys; only protected keys need a lock.
	var config Token
	if err := DB.Select("id", "status", "minute_success_limit").First(&config, tokenID).Error; err != nil {
		return nil, err
	}
	if config.Status != common.TokenStatusEnabled {
		return nil, ErrTokenInvalid
	}
	if config.MinuteSuccessLimit == 0 {
		return nil, nil
	}
	var slot *TokenCallSlot
	err := DB.Transaction(func(tx *gorm.DB) error {
		var token Token
		if err := lockForTokenCall(tx).Select("id", "status", "minute_success_limit").First(&token, tokenID).Error; err != nil {
			return err
		}
		if token.Status != common.TokenStatusEnabled {
			return ErrTokenInvalid
		}
		if token.MinuteSuccessLimit == 0 {
			return nil
		}
		cutoff := time.Now().Add(-time.Minute).UnixMilli()
		if err := tx.Where("token_id = ? AND finished_at > 0 AND finished_at <= ?", tokenID, cutoff).Delete(&TokenCallSlot{}).Error; err != nil {
			return err
		}
		var used int64
		if err := tx.Model(&TokenCallSlot{}).Where("token_id = ?", tokenID).Count(&used).Error; err != nil {
			return err
		}
		if used > int64(token.MinuteSuccessLimit) {
			return ErrTokenCallCapacity
		}
		slot = &TokenCallSlot{ID: common.GetUUID(), TokenID: tokenID}
		return tx.Create(slot).Error
	})
	return slot, err
}

func (slot *TokenCallSlot) Finish(outcome TokenCallOutcome) error {
	if slot == nil {
		return nil
	}
	var disabledKey string
	err := DB.Transaction(func(tx *gorm.DB) error {
		var token Token
		if err := lockForTokenCall(tx).Select("id", "key", "status", "minute_success_limit").First(&token, slot.TokenID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		pending := tx.Model(&TokenCallSlot{}).Where("id = ? AND finished_at = 0", slot.ID)
		if outcome == CallRejected {
			return pending.Delete(&TokenCallSlot{}).Error
		}
		now := time.Now()
		updated := pending.Updates(map[string]any{"finished_at": now.UnixMilli(), "successful": outcome == CallSucceeded})
		if updated.Error != nil || updated.RowsAffected == 0 || outcome != CallSucceeded || token.MinuteSuccessLimit == 0 || token.Status != common.TokenStatusEnabled {
			return updated.Error
		}
		var successes int64
		if err := tx.Model(&TokenCallSlot{}).Where("token_id = ? AND successful = ? AND finished_at > ?", token.Id, true, now.Add(-time.Minute).UnixMilli()).Count(&successes).Error; err != nil {
			return err
		}
		if successes <= int64(token.MinuteSuccessLimit) {
			return nil
		}
		disabledKey = token.Key
		return tx.Model(&token).Update("status", common.TokenStatusDisabled).Error
	})
	if err == nil && common.RedisEnabled && disabledKey != "" {
		err = cacheDeleteToken(disabledKey)
	}
	return err
}

func (token *Token) UpdateCallGuardStatus() error {
	err := DB.Transaction(func(tx *gorm.DB) error {
		var current Token
		if err := lockForTokenCall(tx).Select("id", "status").First(&current, token.Id).Error; err != nil {
			return err
		}
		if current.Status != common.TokenStatusEnabled && token.Status == common.TokenStatusEnabled {
			if err := tx.Where("token_id = ?", token.Id).Delete(&TokenCallSlot{}).Error; err != nil {
				return err
			}
		}
		return tx.Model(token).Update("status", token.Status).Error
	})
	if err == nil && common.RedisEnabled {
		err = cacheDeleteToken(token.Key)
	}
	return err
}
