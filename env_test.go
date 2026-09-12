//go:build !wasm

package env

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGet_ReadsFromEnv(t *testing.T) {
	t.Setenv("ENV_TEST_KEY", "from-process")
	if got := Get("ENV_TEST_KEY"); got != "from-process" {
		t.Errorf("got %q, want %q", got, "from-process")
	}
}

func TestGet_MissingReturnsEmpty(t *testing.T) {
	os.Unsetenv("ENV_TEST_MISSING_123")
	if got := Get("ENV_TEST_MISSING_123"); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestLookup_FoundAndMissing(t *testing.T) {
	t.Setenv("ENV_LOOKUP_KEY", "value")
	if v, ok := Lookup("ENV_LOOKUP_KEY"); !ok || v != "value" {
		t.Errorf("Lookup found = %v %q, want true value", ok, v)
	}
	os.Unsetenv("ENV_LOOKUP_MISSING_123")
	if _, ok := Lookup("ENV_LOOKUP_MISSING_123"); ok {
		t.Error("Lookup missing should return not ok")
	}
}

func TestGetOr_FallsBack(t *testing.T) {
	os.Unsetenv("ENV_GETOR_MISSING")
	if got := GetOr("ENV_GETOR_MISSING", "fallback"); got != "fallback" {
		t.Errorf("got %q, want fallback", got)
	}
	t.Setenv("ENV_GETOR_KEY", "primary")
	if got := GetOr("ENV_GETOR_KEY", "fallback"); got != "primary" {
		t.Errorf("got %q, want primary", got)
	}
}

func TestRequire_FoundAndMissing(t *testing.T) {
	t.Setenv("ENV_REQUIRE_KEY", "present")
	if v, err := Require("ENV_REQUIRE_KEY"); err != nil || v != "present" {
		t.Errorf("Require found err %v %q", err, v)
	}
	os.Unsetenv("ENV_REQUIRE_MISSING_123")
	if _, err := Require("ENV_REQUIRE_MISSING_123"); err == nil {
		t.Error("Require missing should error")
	}
}

func TestRequire_MissingErrorNamesEnvPath(t *testing.T) {
	os.Unsetenv("ENV_REQUIRE_DIAGNOSTIC_MISSING")
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	_, err := Require("ENV_REQUIRE_DIAGNOSTIC_MISSING")
	if err == nil {
		t.Fatal("Require missing should error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "ENV_REQUIRE_DIAGNOSTIC_MISSING") {
		t.Errorf("error %q should name the missing key", msg)
	}
	if !strings.Contains(msg, ".env") {
		t.Errorf("error %q should name the .env path it checked", msg)
	}
	if !strings.Contains(msg, dir) {
		t.Errorf("error %q should name the working directory %q it resolved .env against", msg, dir)
	}
}

func TestLookup_FallsBackToDotEnv(t *testing.T) {
	os.Unsetenv("ENV_DOTENV_KEY")
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("OTHER=1\nENV_DOTENV_KEY=\"from-file\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)
	if got := Get("ENV_DOTENV_KEY"); got != "from-file" {
		t.Errorf("got %q, want from-file", got)
	}
}

func TestLookup_EnvVarWinsOverDotEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	os.WriteFile(path, []byte("ENV_WIN_KEY=from-file\n"), 0644)
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)
	t.Setenv("ENV_WIN_KEY", "from-process")
	if got := Get("ENV_WIN_KEY"); got != "from-process" {
		t.Errorf("got %q, want from-process", got)
	}
}
