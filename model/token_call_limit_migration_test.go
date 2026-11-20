package model

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestLegacyTokenCallLimitUpgradePreservesLaterOptOut(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	conn, err := db.DB()
	require.NoError(t, err)
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = conn.Close() })
	require.NoError(t, db.AutoMigrate(&Option{}, &Token{}))
	old := Token{Key: "old-key"}
	custom := Token{Key: "custom-key", MinuteSuccessLimit: 8}
	require.NoError(t, db.Create(&old).Error)
	require.NoError(t, db.Create(&custom).Error)
	readLimit := func(id int) int {
		var token Token
		require.NoError(t, db.First(&token, id).Error)
		return token.MinuteSuccessLimit
	}

	require.NoError(t, migrateLegacyTokenCallLimits(db))
	require.Equal(t, 3, readLimit(old.Id))
	require.Equal(t, 8, readLimit(custom.Id))

	require.NoError(t, db.Model(&old).UpdateColumn("minute_success_limit", 0).Error)
	optOut := Token{Key: "new-unlimited-key"}
	require.NoError(t, db.Create(&optOut).Error)
	require.NoError(t, migrateLegacyTokenCallLimits(db))
	require.Zero(t, readLimit(old.Id))
	require.Zero(t, readLimit(optOut.Id))
}
