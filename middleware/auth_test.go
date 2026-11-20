package middleware

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestTokenAuthRejectsKeysOfDeletedUser(t *testing.T) {
	t.Setenv("SQL_DSN", "")
	db, sqlitePath, usingSQLite := model.DB, common.SQLitePath, common.UsingSQLite
	isMasterNode, redisEnabled := common.IsMasterNode, common.RedisEnabled
	t.Cleanup(func() {
		model.DB, common.SQLitePath, common.UsingSQLite = db, sqlitePath, usingSQLite
		common.IsMasterNode, common.RedisEnabled = isMasterNode, redisEnabled
	})
	common.SQLitePath = filepath.Join(t.TempDir(), "auth.db")
	common.IsMasterNode = false
	common.RedisEnabled = false
	require.NoError(t, model.InitDB())
	sqlDB, err := model.DB.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	require.NoError(t, model.DB.AutoMigrate(&model.User{}, &model.Token{}))
	require.NoError(t, i18n.Init())

	user := model.User{Username: "deleted-key-owner", Status: common.UserStatusEnabled, Group: "default"}
	require.NoError(t, model.DB.Create(&user).Error)
	const key = "deletedownerkey0123456789abcdefghijklmnopqrstuvwx"
	require.NoError(t, model.DB.Create(&model.Token{
		UserId: user.Id, Key: key, Name: "owner key", Status: common.TokenStatusEnabled,
		ExpiredTime: -1, UnlimitedQuota: true,
	}).Error)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/v1/chat/completions", TokenAuth(), func(c *gin.Context) { c.Status(http.StatusOK) })
	router.GET("/api/usage/token", TokenAuthReadOnly(), func(c *gin.Context) { c.Status(http.StatusOK) })
	checkStatus := func(want int) {
		t.Helper()
		for _, path := range []string{"/v1/chat/completions", "/api/usage/token"} {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.Header.Set("Authorization", "Bearer sk-"+key)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, req)
			require.Equal(t, want, response.Code, "%s: %s", path, response.Body.String())
		}
	}

	checkStatus(http.StatusOK)
	require.NoError(t, model.DeleteUserById(user.Id))
	checkStatus(http.StatusUnauthorized)

	// Fail the user lookup while leaving the token table accessible.
	require.NoError(t, model.DB.Migrator().DropTable(&model.User{}))
	checkStatus(http.StatusInternalServerError)
}
