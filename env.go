// Package env provides auto-tagged environment access: on !wasm it reads
// the process environment (os.Getenv + .env fallback), on wasm it reads
// Cloudflare's context.env via syscall/js. No injection, no Reader —
// the build tag selects the implementation.
package env

// Get returns the value of key or "" if unset.
func Get(key string) string {
	v, _ := Lookup(key)
	return v
}

// GetOr returns the value of key or fallback if unset.
func GetOr(key, fallback string) string {
	if v, ok := Lookup(key); ok {
		return v
	}
	return fallback
}
