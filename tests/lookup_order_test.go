//go:build !wasm

package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"webtyp.com/env"
	keyringauto "webtyp.com/keyring/auto"
)

func writeGoMod(t *testing.T, dir, module string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module "+module+"\n\ngo 1.25\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
}

func setupModule(t *testing.T, moduleName string) string {
	t.Helper()
	dir := t.TempDir()
	writeGoMod(t, dir, moduleName)
	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) })
	return dir
}

func setupKeyring(t *testing.T, key, value string) {
	t.Helper()
	kr, err := keyringauto.OpenForModule(".")
	if err != nil {
		t.Skip("no keyring backend available: " + err.Error())
	}
	if err := kr.Set(key, value); err != nil {
		t.Skip("keyring Set unavailable: " + err.Error())
	}
	t.Cleanup(func() { kr.Delete(key) })
}

func TestLookupOrder_O1_EnvWinsOverDotEnvAndKeyring(t *testing.T) {
	setupModule(t, "test-o1")
	setupKeyring(t, "KEY", "from-keyring")
	if err := os.WriteFile(".env", []byte("KEY=from-dotenv\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KEY", "from-process")

	if got, ok := env.Lookup("KEY"); !ok || got != "from-process" {
		t.Errorf("Lookup(KEY) = %q, %v, want from-process, true", got, ok)
	}
}

func TestLookupOrder_O2_DotEnvWinsOverKeyring(t *testing.T) {
	setupModule(t, "test-o2")
	setupKeyring(t, "KEY", "from-keyring")
	if err := os.WriteFile(".env", []byte("KEY=from-dotenv\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if got, ok := env.Lookup("KEY"); !ok || got != "from-dotenv" {
		t.Errorf("Lookup(KEY) = %q, %v, want from-dotenv, true", got, ok)
	}
}

func TestLookupOrder_O3_KeyringUsedWhenNotFoundElse(t *testing.T) {
	setupModule(t, "test-o3")
	setupKeyring(t, "KEY", "from-keyring")

	if got, ok := env.Lookup("KEY"); !ok || got != "from-keyring" {
		t.Errorf("Lookup(KEY) = %q, %v, want from-keyring, true", got, ok)
	}
}

func TestLookupOrder_O4_KeyNowhereReturnsEmptyAndRequireErrorsWithThreePlaces(t *testing.T) {
	dir := setupModule(t, "test-o4")
	// Ensure no leftover value from previous run could make this spuriously pass
	if kr, err := keyringauto.OpenForModule("."); err == nil {
		kr.Delete("KEY")
	}

	if got, ok := env.Lookup("KEY"); ok || got != "" {
		t.Errorf("Lookup(KEY) = %q, %v, want \"\", false", got, ok)
	}

	_, err := env.Require("KEY")
	if err == nil {
		t.Fatal("Require missing should error")
	}

	msg := err.Error()
	if !strings.Contains(msg, "KEY is not set") {
		t.Errorf("error %q should name the missing key", msg)
	}
	if !strings.Contains(msg, "checked the process environment") {
		t.Errorf("error %q should mention process environment", msg)
	}
	if !strings.Contains(msg, ".env in "+dir) {
		t.Errorf("error %q should mention .env in CWD", msg)
	}
	if !strings.Contains(msg, "and the keyring of this module") {
		t.Errorf("error %q should mention the keyring of this module", msg)
	}
}

func TestLookupOrder_O5_KeyringMarkerInDotEnvReturnsLiteral(t *testing.T) {
	setupModule(t, "test-o5")
	if err := os.WriteFile(".env", []byte("KEY=keyring://\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if got, ok := env.Lookup("KEY"); !ok || got != "keyring://" {
		t.Errorf("Lookup(KEY) = %q, %v, want keyring://, true", got, ok)
	}
}

func TestLookupOrder_O6_LookupAtReadsKeyringOfOtherModuleDir(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "test-o6-other")

	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	kr, err := keyringauto.OpenForModule(".")
	if err != nil {
		t.Skip("no keyring backend available: " + err.Error())
	}
	if err := kr.Set("KEY", "from-other-keyring"); err != nil {
		t.Skip("keyring Set unavailable: " + err.Error())
	}
	t.Cleanup(func() { kr.Delete("KEY") })
	os.Chdir(orig) // Go back to original dir

	envPath := filepath.Join(dir, ".env")
	// don't create .env file, so it falls back to keyring

	if got, ok := env.LookupAt("KEY", envPath); !ok || got != "from-other-keyring" {
		t.Errorf("LookupAt(KEY) = %q, %v, want from-other-keyring, true", got, ok)
	}
}
