//go:build !wasm

package env

import (
	"os"

	"webtyp.com/fmt"
)

// Require returns the value of key or an error naming both the key and the
// .env path checked, so a missing/misplaced key is self-diagnosing — the
// process environment, defaultDotEnvPath (resolved against the current
// working directory), and the module's keyring are the three places Lookup ever looks.
func Require(key string) (string, error) {
	if v, ok := Lookup(key); ok {
		return v, nil
	}
	cwd, _ := os.Getwd()
	return "", fmt.Err(key + " is not set (checked the process environment, " + defaultDotEnvPath + " in " + cwd + ", and the keyring of this module)")
}
