//go:build !wasm

package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"webtyp.com/env"
)

func TestFile_ReadsSecret(t *testing.T) {
	dir := setupModule(t, "test-file-reads-secret")
	path := filepath.Join(dir, "secret")
	if err := os.WriteFile(path, []byte("s3cret\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SECRET_X_FILE", path)

	if got := env.Get("SECRET_X"); got != "s3cret" {
		t.Errorf("Get(SECRET_X) = %q, want \"s3cret\"", got)
	}
}

func TestFile_CRLF(t *testing.T) {
	dir := setupModule(t, "test-file-crlf")
	path := filepath.Join(dir, "secret")
	if err := os.WriteFile(path, []byte("s3cret\r\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SECRET_X_FILE", path)

	if got := env.Get("SECRET_X"); got != "s3cret" {
		t.Errorf("Get(SECRET_X) = %q, want \"s3cret\"", got)
	}
}

func TestFile_KeepsInnerNewlines(t *testing.T) {
	dir := setupModule(t, "test-file-inner-newlines")
	path := filepath.Join(dir, "secret")
	if err := os.WriteFile(path, []byte("a\nb\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SECRET_X_FILE", path)

	if got := env.Get("SECRET_X"); got != "a\nb" {
		t.Errorf("Get(SECRET_X) = %q, want \"a\\nb\"", got)
	}
}

func TestFile_DirectWins(t *testing.T) {
	dir := setupModule(t, "test-file-direct-wins")
	path := filepath.Join(dir, "secret")
	if err := os.WriteFile(path, []byte("other\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SECRET_X", "direct")
	t.Setenv("SECRET_X_FILE", path)

	if got := env.Get("SECRET_X"); got != "direct" {
		t.Errorf("Get(SECRET_X) = %q, want \"direct\"", got)
	}
}

func TestFile_BeforeDotEnv(t *testing.T) {
	dir := setupModule(t, "test-file-before-dotenv")
	path := filepath.Join(dir, "secret")
	if err := os.WriteFile(path, []byte("fromfile\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(".env", []byte("SECRET_X=fromdotenv\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SECRET_X_FILE", path)

	if got := env.Get("SECRET_X"); got != "fromfile" {
		t.Errorf("Get(SECRET_X) = %q, want \"fromfile\"", got)
	}
}

func TestFile_Unreadable(t *testing.T) {
	dir := setupModule(t, "test-file-unreadable")
	path := filepath.Join(dir, "missing")
	if err := os.WriteFile(".env", []byte("SECRET_X=fromdotenv\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SECRET_X_FILE", path)

	if got, ok := env.Lookup("SECRET_X"); ok || got != "" {
		t.Errorf("Lookup(SECRET_X) = %q, %v; want \"\", false", got, ok)
	}

	_, err := env.Require("SECRET_X")
	if err == nil {
		t.Fatal("Require missing should error")
	}

	msg := err.Error()
	if !strings.Contains(msg, "SECRET_X_FILE") {
		t.Errorf("error %q should contain SECRET_X_FILE", msg)
	}
}
