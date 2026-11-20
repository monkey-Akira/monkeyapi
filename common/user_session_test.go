package common

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitUserSessionSettingsUsesPositiveFallbacksAndClampsWindow(t *testing.T) {
	previousActiveLimit := UserSessionActiveLimit
	previousIssuanceLimit := UserSessionIssuanceLimit
	previousIssuanceWindow := UserSessionIssuanceWindowSeconds
	previousRevokedRetention := UserSessionRevokedRetentionDays
	previousAlertThreshold := UserSessionHourlyAlertThreshold
	t.Cleanup(func() {
		UserSessionActiveLimit = previousActiveLimit
		UserSessionIssuanceLimit = previousIssuanceLimit
		UserSessionIssuanceWindowSeconds = previousIssuanceWindow
		UserSessionRevokedRetentionDays = previousRevokedRetention
		UserSessionHourlyAlertThreshold = previousAlertThreshold
	})

	t.Setenv("USER_SESSION_ACTIVE_LIMIT", "0")
	t.Setenv("USER_SESSION_ISSUANCE_LIMIT", "-2")
	t.Setenv("USER_SESSION_ISSUANCE_WINDOW_SECONDS", "invalid")
	t.Setenv("USER_SESSION_REVOKED_RETENTION_DAYS", "0")
	t.Setenv("USER_SESSION_HOURLY_ALERT_THRESHOLD", "-1")
	initUserSessionSettings()

	assert.Equal(t, DefaultUserSessionActiveLimit, UserSessionActiveLimit)
	assert.Equal(t, DefaultUserSessionIssuanceLimit, UserSessionIssuanceLimit)
	assert.Equal(t, int64(DefaultUserSessionIssuanceWindowSeconds), UserSessionIssuanceWindowSeconds)
	assert.Equal(t, DefaultUserSessionRevokedRetentionDays, UserSessionRevokedRetentionDays)
	assert.Equal(t, DefaultUserSessionHourlyAlertThreshold, UserSessionHourlyAlertThreshold)

	t.Setenv("USER_SESSION_ACTIVE_LIMIT", "12")
	t.Setenv("USER_SESSION_ISSUANCE_LIMIT", "34")
	t.Setenv("USER_SESSION_ISSUANCE_WINDOW_SECONDS", "172800")
	t.Setenv("USER_SESSION_REVOKED_RETENTION_DAYS", "1")
	t.Setenv("USER_SESSION_HOURLY_ALERT_THRESHOLD", "56")
	initUserSessionSettings()

	assert.Equal(t, 12, UserSessionActiveLimit)
	assert.Equal(t, 34, UserSessionIssuanceLimit)
	assert.Equal(t, int64(24*60*60), UserSessionIssuanceWindowSeconds)
	assert.Equal(t, 1, UserSessionRevokedRetentionDays)
	assert.Equal(t, 56, UserSessionHourlyAlertThreshold)

	t.Setenv("USER_SESSION_ISSUANCE_WINDOW_SECONDS", "43200")
	initUserSessionSettings()
	assert.Equal(t, int64(12*60*60), UserSessionIssuanceWindowSeconds, "a window below retention remains unchanged")

	t.Setenv("USER_SESSION_ISSUANCE_WINDOW_SECONDS", "86400")
	initUserSessionSettings()
	assert.Equal(t, int64(24*60*60), UserSessionIssuanceWindowSeconds, "a window equal to retention remains unchanged")

	t.Setenv("USER_SESSION_REVOKED_RETENTION_DAYS", "9223372036854775807")
	initUserSessionSettings()
	assert.Equal(t, DefaultUserSessionRevokedRetentionDays, UserSessionRevokedRetentionDays)
}

func TestValidateSessionSecret(t *testing.T) {
	validSecret := "0123456789abcdef0123456789abcdef"
	tests := []struct {
		name      string
		secret    string
		wantError bool
	}{
		{name: "minimum length", secret: validSecret},
		{name: "too short", secret: validSecret[:31], wantError: true},
		{name: "empty", secret: "", wantError: true},
		{name: "placeholder", secret: "change_this_session_secret_before_use_0123456789abcdef", wantError: true},
		{name: "repeated character", secret: strings.Repeat("a", 32), wantError: true},
		{name: "leading whitespace", secret: " " + validSecret, wantError: true},
		{name: "trailing whitespace", secret: validSecret + " ", wantError: true},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			err := validateSessionSecret(testCase.secret)
			if testCase.wantError {
				if assert.Error(t, err) && testCase.secret != "" {
					assert.NotContains(t, err.Error(), testCase.secret, "validation errors must not expose the secret")
				}
				return
			}
			require.NoError(t, err)
		})
	}
}
