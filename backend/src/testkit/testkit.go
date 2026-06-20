// Package testkit is the foundational integration-testing harness for the
// termorize backend. It is an ordinary (non-_test.go) importable package so that
// every test package can share the same router, database connection and helpers.
//
// Typical usage from a test package:
//
//	// setup_test.go
//	func TestMain(m *testing.M) { testkit.Main(m) }
//
//	// some_test.go
//	func TestSomething(t *testing.T) {
//	    testkit.Truncate(t)
//	    user := testkit.CreateUser(t)
//	    rec := testkit.AuthedRequest(t, user, http.MethodGet, "/api/me", nil)
//	    require.Equal(t, http.StatusOK, rec.Code)
//	}
//
// See TESTING.md at the repo root for the full guide.
package testkit

import (
	// Imported for its side effect of forcing the process timezone to UTC,
	// mirroring main.go. Must be first so it runs before anything else.
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
	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver used to create the test DB
)

var router *gin.Engine

// Main is the entry point for a test package's TestMain. It performs one-time
// global setup (env vars, config, repo-root chdir, test database creation,
// DB connection, migrations, router build), runs the tests, and exits with the
// resulting status code.
//
//	func TestMain(m *testing.M) { testkit.Main(m) }
func Main(m *testing.M) {
	Setup()
	os.Exit(m.Run())
}

// Setup runs the one-time global initialization. Main calls it for you; call it
// directly only if you need custom control over the test lifecycle.
//
// It is safe to call multiple times within a single process (the router and DB
// are only built once), but it is intended to be called once per `go test`
// binary via Main.
func Setup() {
	// Silence logs so they don't pollute test output. Must run before anything
	// else touches the logger (e.g. config.LoadEnv) so initLogger never runs.
	logger.UseNop()
	// Gin's debug logging (route registration, etc.) also goes to stderr; TestMode
	// suppresses it without depending on the production-only ReleaseMode branch.
	gin.SetMode(gin.TestMode)

	setTestEnv()

	// db.Migrate uses a path relative to the repo root, so move the process cwd
	// there. Test binaries run with the package directory as cwd.
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

	// Default the external clients to safe, non-network fakes so that no test can
	// accidentally hit the real Google/OpenRouter APIs. Individual tests can
	// override these via MockGoogleTranslate / MockOpenRouter.
	installDefaultExternalFakes()

	router = apphttp.BuildRouter()
}

// Router returns the shared, fully-configured Gin engine. It is built once
// during Setup.
func Router() *gin.Engine {
	if router == nil {
		panic("testkit: Router() called before Setup(); wire testkit.Main into TestMain")
	}
	return router
}

// testDBName returns the database name used for tests. It can be overridden with
// the TEST_DB_NAME environment variable (default "termorize_test") so concurrent
// or isolated runs can target distinct databases.
func testDBName() string {
	if name := os.Getenv("TEST_DB_NAME"); name != "" {
		return name
	}
	return "termorize_test"
}

// setTestEnv populates all environment variables required by config.LoadEnv with
// safe test values. Existing values are preserved so CI / local overrides win.
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

	// DB_NAME always points at the (possibly overridden) test database. We set it
	// unconditionally so a stray DB_NAME=termorize in the shell can never make
	// tests touch the dev database.
	os.Setenv("DB_NAME", testDBName())
}

func setIfEmpty(key, value string) {
	if os.Getenv(key) == "" {
		os.Setenv(key, value)
	}
}

// chdirRepoRoot walks up from the current working directory until it finds the
// directory containing go.mod and changes into it, so relative paths used by the
// migration runner resolve correctly.
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

// ensureTestDatabase connects to the default "postgres" database and creates the
// test database if it does not already exist. It never touches the dev database.
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

	// Database names cannot be parameterized; the name is sourced from config and
	// validated implicitly by Postgres. Quote it to be safe.
	if _, err := sqlDB.Exec(fmt.Sprintf(`CREATE DATABASE %q`, name)); err != nil {
		return fmt.Errorf("create database %q: %w", name, err)
	}

	return nil
}
