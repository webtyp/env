//go:build !wasm

package env

import (
	"os"
	"path/filepath"

	"webtyp.com/fmt"
	keyringauto "webtyp.com/keyring/auto"
)

const defaultDotEnvPath = ".env"

func lookupEnvFileAndKeyring(key, path, moduleDir string) (string, bool) {
	data, err := os.ReadFile(path)
	if err == nil {
		prefix := key + "="
		for _, line := range fmt.Split(string(data), "\n") {
			if !fmt.HasPrefix(line, prefix) {
				continue
			}
			v := fmt.Convert(line).TrimPrefix(prefix).String()
			if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
				v = v[1 : len(v)-1]
			}
			return v, true
		}
	}
	kr, err := keyringauto.OpenForModule(moduleDir)
	if err == nil {
		v, err := kr.Get(key)
		if err == nil {
			return v, true
		}
	}
	return "", false
}

// Lookup returns the value of key from the process environment, falling back
// to a .env file in the working directory, and finally to the module's keyring.
// The bool indicates if the key was found.
func Lookup(key string) (string, bool) {
	if v, ok := os.LookupEnv(key); ok {
		return v, true
	}
	return lookupEnvFileAndKeyring(key, defaultDotEnvPath, ".")
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
		return v, true
	}
	return lookupEnvFileAndKeyring(key, path, filepath.Dir(path))
}
