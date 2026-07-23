package testkit

import (
	_ "termorize/src/utils"

	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"termorize/src/config"
	"termorize/src/data/db"
	apphttp "termorize/src/http"
	"termorize/src/logger"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
)

var router *gin.Engine

func Main(m *testing.M) {
	Setup()
	os.Exit(m.Run())
}

func Setup() {
	logger.UseNop()
	gin.SetMode(gin.TestMode)

	setTestEnv()

	chdirRepoRoot()

	config.LoadEnv()

	if err := ensureTestDatabase(); err != nil {
		panic(fmt.Sprintf("testkit: failed to ensure test database: %v", err))
	}

	if err := db.Connect(); err != nil {
		panic(fmt.Sprintf("testkit: failed to connect to test database: %v", err))
	}

	if err := db.Migrate(); err != nil {
		panic(fmt.Sprintf("testkit: failed to run migrations: %v", err))
	}

	installDefaultExternalFakes()

	router = apphttp.BuildRouter()
}

func Router() *gin.Engine {
	if router == nil {
		panic("testkit: Router() called before Setup(); wire testkit.Main into TestMain")
	}
	return router
}

func testDBName() string {
	if name := os.Getenv("TEST_DB_NAME"); name != "" {
		return name
	}
	return "termorize_test"
}

func setTestEnv() {
	setIfEmpty("ENV", "local")
	setIfEmpty("SECRET", "test-secret-do-not-use-in-production")

	setIfEmpty("DB_HOST", "127.0.0.1")
	setIfEmpty("DB_PORT", "5432")
	setIfEmpty("DB_USER", "root")
	setIfEmpty("DB_PASSWORD", "password")

	setIfEmpty("TELEGRAM_BOT_TOKEN", "test-telegram-bot-token")
	setIfEmpty("TELEGRAM_LOGIN_CLIENT_ID", "test-telegram-client-id")
	setIfEmpty("TELEGRAM_LOGIN_CLIENT_SECRET", "test-telegram-client-secret")

	setIfEmpty("GOOGLE_API_KEY", "test-google-api-key")

	os.Setenv("DB_NAME", testDBName())
}

func setIfEmpty(key, value string) {
	if os.Getenv(key) == "" {
		os.Setenv(key, value)
	}
}

func chdirRepoRoot() {
	dir, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("testkit: getwd failed: %v", err))
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			if chErr := os.Chdir(dir); chErr != nil {
				panic(fmt.Sprintf("testkit: chdir to repo root failed: %v", chErr))
			}
			return
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			panic("testkit: could not locate repo root (go.mod not found walking up)")
		}
		dir = parent
	}
}

func ensureTestDatabase() error {
	name := testDBName()

	adminDSN := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=postgres sslmode=disable",
		config.GetDBHost(),
		config.GetDBPort(),
		config.GetDBUser(),
		config.GetDBPassword(),
	)

	sqlDB, err := sql.Open("pgx", adminDSN)
	if err != nil {
		return fmt.Errorf("open admin connection: %w", err)
	}
	defer sqlDB.Close()

	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("ping admin connection (is postgres running on %s:%s?): %w",
			config.GetDBHost(), config.GetDBPort(), err)
	}

	var exists bool
	if err := sqlDB.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", name,
	).Scan(&exists); err != nil {
		return fmt.Errorf("check database existence: %w", err)
	}

	if exists {
		return nil
	}

	if _, err := sqlDB.Exec(fmt.Sprintf(`CREATE DATABASE %q`, name)); err != nil {
		return fmt.Errorf("create database %q: %w", name, err)
	}

	return nil
}
