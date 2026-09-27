package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassificationConfigDoesNotRequireApplicationCredentials(t *testing.T) {
	t.Chdir(t.TempDir())
	previous := config
	t.Cleanup(func() { config = previous })
	for key, value := range map[string]string{
		"ENV": "local", "DB_HOST": "127.0.0.1", "DB_PORT": "55432",
		"DB_NAME": "termorize_classification_test", "DB_USER": "root", "DB_PASSWORD": "password",
		"OPENROUTER_API_KEY": "classification-test-key", "SENTRY_DSN": "",
		"SECRET": "", "TELEGRAM_BOT_TOKEN": "", "TELEGRAM_LOGIN_CLIENT_ID": "",
		"TELEGRAM_LOGIN_CLIENT_SECRET": "", "GOOGLE_API_KEY": "",
	} {
		t.Setenv(key, value)
	}
	require.NotPanics(t, LoadClassificationEnv)
	assert.Equal(t, "local", GetEnv())
	assert.Equal(t, "127.0.0.1", GetDBHost())
	assert.Equal(t, "55432", GetDBPort())
	assert.Equal(t, "termorize_classification_test", GetDBName())
	assert.Equal(t, "root", GetDBUser())
	assert.Equal(t, "password", GetDBPassword())
	assert.Equal(t, "classification-test-key", GetOpenRouterApiKey())
	assert.Empty(t, GetSentryDSN())
	require.PanicsWithValue(t, "Required environment variable is missing: SECRET", LoadEnv)
}
