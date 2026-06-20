package tests

import (
	"testing"

	"termorize/src/testkit"
)

// TestMain wires the shared testkit harness for every test in this package.
// One-time setup (env, config, test DB, migrations, router) happens here.
//
// Downstream per-endpoint test files in this package can simply rely on the
// testkit API; they do not need their own TestMain.
func TestMain(m *testing.M) {
	testkit.Main(m)
}
