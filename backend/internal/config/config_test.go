package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnvFileDoesNotOverrideProcessEnv(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, ".env")
	content := "LAMSZA_CFG_TEST_SET=from-file\nLAMSZA_CFG_TEST_UNSET=from-file\n"
	if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LAMSZA_CFG_TEST_SET", "from-env")
	t.Setenv("LAMSZA_CFG_TEST_UNSET", "")
	os.Unsetenv("LAMSZA_CFG_TEST_UNSET")

	loadEnvFile(file, presetEnv())

	if got := os.Getenv("LAMSZA_CFG_TEST_SET"); got != "from-env" {
		t.Errorf("exported variable: got %q, want from-env", got)
	}
	if got := os.Getenv("LAMSZA_CFG_TEST_UNSET"); got != "from-file" {
		t.Errorf("unset variable: got %q, want from-file", got)
	}
}

func TestLaterEnvFileStillWinsOverEarlierFile(t *testing.T) {
	dir := t.TempDir()
	parent := filepath.Join(dir, "parent.env")
	local := filepath.Join(dir, "local.env")
	if err := os.WriteFile(parent, []byte("LAMSZA_CFG_TEST_ORDER=parent\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(local, []byte("LAMSZA_CFG_TEST_ORDER=local\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LAMSZA_CFG_TEST_ORDER", "")
	os.Unsetenv("LAMSZA_CFG_TEST_ORDER")

	preset := presetEnv()
	loadEnvFile(parent, preset)
	loadEnvFile(local, preset)

	if got := os.Getenv("LAMSZA_CFG_TEST_ORDER"); got != "local" {
		t.Errorf("got %q, want local", got)
	}
}
