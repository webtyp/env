//go:build wasm

package env

import "webtyp.com/fmt"

// Require returns the value of key or an error if unset. On wasm there is no
// CWD and no .env fallback to name — Lookup reads Cloudflare's context.env.
func Require(key string) (string, error) {
	if v, ok := Lookup(key); ok {
		return v, nil
	}
	return "", fmt.Err(key + " is not set")
}
