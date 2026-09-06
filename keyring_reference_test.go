//go:build !wasm

package env

import (
	"os"
	"path/filepath"
	"testing"

	keyringauto "webtyp.com/keyring/auto"
)

func writeGoMod(t *testing.T, dir, module string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module "+module+"\n\ngo 1.25\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
}

func TestLookupResolvesKeyringReference(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "keyring-ref-test-lookup")
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("SECRET=keyring://\n"), 0644); err != nil {
		t.Fatal(err)
	}
	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(orig)

	kr, err := keyringauto.OpenForModule(".")
	if err != nil {
		t.Skip("no keyring backend available: " + err.Error())
	}
	if err := kr.Set("SECRET", "the-real-value"); err != nil {
		t.Skip("keyring Set unavailable: " + err.Error())
	}
	t.Cleanup(func() { kr.Delete("SECRET") })

	if v, ok := Lookup("SECRET"); !ok || v != "the-real-value" {
		t.Fatalf("Lookup(SECRET) = %q %v, want the-real-value true", v, ok)
	}
}

func TestLookupReferenceNotStoredIsNotFound(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "keyring-ref-test-notfound")
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("SECRET=keyring://\n"), 0644); err != nil {
		t.Fatal(err)
	}
	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(orig)

	// Ensure no leftover value from previous run could make this spuriously pass
	kr, err := keyringauto.OpenForModule(".")
	if err != nil {
		t.Skip("no keyring backend available: " + err.Error())
	}
	// Clean any stale entry before test
	kr.Delete("SECRET")

	if v, ok := Lookup("SECRET"); ok {
		t.Fatalf("Lookup(SECRET) with no stored value = %q %v, want \"\" false", v, ok)
	}
}

func TestLookupLiteralValueUnaffected(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "keyring-ref-test-literal")
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("PLAIN=hello\n"), 0644); err != nil {
		t.Fatal(err)
	}
	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(orig)

	if v, ok := Lookup("PLAIN"); !ok || v != "hello" {
		t.Fatalf("Lookup(PLAIN) = %q %v, want hello true", v, ok)
	}
}

func TestLookupAtResolvesKeyringReference(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "keyring-ref-test-lookupat")
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte("SECRET=keyring://\n"), 0644); err != nil {
		t.Fatal(err)
	}
	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(orig)

	kr, err := keyringauto.OpenForModule(".")
	if err != nil {
		t.Skip("no keyring backend available: " + err.Error())
	}
	if err := kr.Set("SECRET", "the-real-value"); err != nil {
		t.Skip("keyring Set unavailable: " + err.Error())
	}
	t.Cleanup(func() { kr.Delete("SECRET") })

	if v, ok := LookupAt("SECRET", envPath); !ok || v != "the-real-value" {
		t.Fatalf("LookupAt(SECRET) = %q %v, want the-real-value true", v, ok)
	}
}

func TestSetThenLookupResolvesKeyringReference(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "keyring-ref-test-setlookup")
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("PLAIN=unused\n"), 0644); err != nil {
		t.Fatal(err)
	}
	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(orig)

	kr, err := keyringauto.OpenForModule(".")
	if err != nil {
		t.Skip("no keyring backend available: " + err.Error())
	}
	if err := kr.Set("SECRET", "the-real-value"); err != nil {
		t.Skip("keyring Set unavailable: " + err.Error())
	}
	t.Cleanup(func() { kr.Delete("SECRET") })

	t.Setenv("SECRET", "keyring://")
	// Also ensure cleanup of env var handled by t.Setenv
	if v, ok := Lookup("SECRET"); !ok || v != "the-real-value" {
		t.Fatalf("Lookup(SECRET) via os env = %q %v, want the-real-value true", v, ok)
	}
}
