package model

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Older releases stored both an untouched limit and an explicit opt-out as 0.
// Claim the backfill and update the keys in one transaction, so concurrent
// startups and later restarts preserve any opt-outs made after the upgrade.
func migrateLegacyTokenCallLimits(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		marker := Option{Key: "migration.token_minute_success_limit_v1", Value: "pending"}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&marker).Error; err != nil {
			return err
		}
		claim := tx.Model(&marker).Where(&Option{Value: "pending"}).Update("value", "done")
		if claim.Error != nil || claim.RowsAffected == 0 {
			return claim.Error
		}
		return tx.Model(&Token{}).
			Where("minute_success_limit = 0 OR minute_success_limit IS NULL").
			UpdateColumn("minute_success_limit", 3).Error
	})
}
