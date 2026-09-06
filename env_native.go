//go:build !wasm

package env

import (
	"os"

	"webtyp.com/fmt"
	"webtyp.com/keyring"
	keyringauto "webtyp.com/keyring/auto"
)

// resolveIfReference returns raw unchanged unless it is exactly
// keyring.Scheme ("keyring://"), in which case it looks key up in the
// keyring scoped to the current module (keyringauto.OpenForModule(".")) and
// returns that instead. ok is false whenever the real value cannot be
// produced — a keyring open failure or a key never stored is exactly as
// "not found" as the key being absent from .env in the first place.
func resolveIfReference(key, raw string) (string, bool) {
	if !keyring.IsReference(raw) {
		return raw, true
	}
	kr, err := keyringauto.OpenForModule(".")
	if err != nil {
		return "", false
	}
	v, err := kr.Get(key)
	if err != nil {
		return "", false
	}
	return v, true
}

const defaultDotEnvPath = ".env"

// Lookup returns the value of key from the process environment, falling back
// to a .env file in the working directory. The bool indicates if the key was found.
func Lookup(key string) (string, bool) {
	if v, ok := os.LookupEnv(key); ok {
		return resolveIfReference(key, v)
	}
	data, err := os.ReadFile(defaultDotEnvPath)
	if err != nil {
		return "", false
	}
	prefix := key + "="
	for _, line := range fmt.Split(string(data), "\n") {
		if !fmt.HasPrefix(line, prefix) {
			continue
		}
		v := fmt.Convert(line).TrimPrefix(prefix).String()
		if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
			v = v[1 : len(v)-1]
		}
		return resolveIfReference(key, v)
	}
	return "", false
}

// Set sets key to value in the process environment. Only available on !wasm.
func Set(key, value string) error {
	return os.Setenv(key, value)
}

// Arg returns the value for -key=value or -key value in os.Args (skipping argv[0]).
// Only available on !wasm; wasm has no argv.
func Arg(key string) string {
	prefix := "-" + key + "="
	args := os.Args[1:]
	for i, arg := range args {
		if fmt.HasPrefix(arg, prefix) {
			return fmt.Convert(arg).TrimPrefix(prefix).String()
		}
		if arg == "-"+key && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

// LookupAt is Lookup with an explicit .env path, for callers not running from the project root.
func LookupAt(key, path string) (string, bool) {
	if v, ok := os.LookupEnv(key); ok {
		return resolveIfReference(key, v)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	prefix := key + "="
	for _, line := range fmt.Split(string(data), "\n") {
		if !fmt.HasPrefix(line, prefix) {
			continue
		}
		v := fmt.Convert(line).TrimPrefix(prefix).String()
		if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
			v = v[1 : len(v)-1]
		}
		return resolveIfReference(key, v)
	}
	return "", false
}
