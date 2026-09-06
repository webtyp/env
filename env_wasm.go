//go:build wasm

package env

import (
	"syscall/js"

	"webtyp.com/fmt"
	"webtyp.com/keyring"
)

// Lookup reads from Cloudflare's runtime context (context.env) via syscall/js.
// On non-Cloudflare wasm targets, context.env will be undefined and Lookup returns not found.
func Lookup(key string) (string, bool) {
	ctx := js.Global().Get("context")
	if ctx.IsNull() || ctx.IsUndefined() {
		return "", false
	}
	jsEnv := ctx.Get("env")
	if jsEnv.IsNull() || jsEnv.IsUndefined() {
		return "", false
	}
	val := jsEnv.Get(key)
	if val.IsNull() || val.IsUndefined() {
		return "", false
	}
	v := val.String()
	if keyring.IsReference(v) {
		return "", false // the marker is a local-dev convention; never a real Worker value
	}
	return v, true
}

// Set is not available on wasm — platform env vars are not writable at runtime.
func Set(key, value string) error {
	return fmt.Err("env: Set not available on wasm")
}

// Arg has no wasm equivalent — Workers have no argv.
func Arg(key string) string {
	return ""
}

// LookupAt is not meaningful on wasm — there is no .env file.
func LookupAt(key, path string) (string, bool) {
	return Lookup(key)
}
