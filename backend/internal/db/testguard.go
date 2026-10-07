package db

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
)

// TestDatabaseURL returns the database a test binary may use. Tests never read
// DATABASE_URL or .env: those point at the shared dev lamsza database, and the
// suites insert users and sessions. Use `npm run test:backend`, which builds a
// fresh scratch database from lamsza/backend/schema/ and sets TEST_DATABASE_URL.
func TestDatabaseURL() (string, error) {
	raw := os.Getenv("TEST_DATABASE_URL")
	if raw == "" {
		return "", errors.New("TEST_DATABASE_URL is not set. DB-bound tests never use DATABASE_URL or .env, " +
			"which point at the shared dev database. Run `npm run test:backend` from the repo root: it builds a scratch database and sets it")
	}
	if err := checkTestDatabaseURL(raw); err != nil {
		return "", err
	}
	return raw, nil
}

// checkTestDatabaseURL refuses a database that is not a scratch one. Locally the
// name must end in _test. CI (CI=true, set by GitHub Actions) only ever sees a
// throwaway service container, so any name is fine there.
func checkTestDatabaseURL(raw string) error {
	if os.Getenv("CI") == "true" {
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("TEST_DATABASE_URL: %w", err)
	}
	name := strings.TrimPrefix(u.Path, "/")
	if !strings.HasSuffix(name, "_test") {
		return fmt.Errorf("refusing to run tests against database %q: a local test database name must end in _test, so the shared dev database can never be hit by mistake", name)
	}
	return nil
}

// guardTestConnection stops a test binary from opening anything but a scratch
// database, whichever code path it took to get here.
func guardTestConnection(connStr string) error {
	if !testing.Testing() {
		return nil
	}
	return checkTestDatabaseURL(connStr)
}
