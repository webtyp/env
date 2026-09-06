---
PLAN: "feat: resolve the keyring:// .env marker through webtyp/keyring (native only)"
EXECUTOR: jules
REVIEWER: none
---

> This plan is dispatched via the CodeJob workflow. See skill: agents-workflow.

# Plan — one resolution point for every consumer of `env.Get`/`env.Lookup`

## Part of a multi-repo wave — depends on `webtyp/keyring` publishing first

This is one piece of `KEYRING_DOTENV_MASTER_PLAN.md` (orchestrator:
`webtyp.com/app-releases`, `docs/KEYRING_DOTENV_MASTER_PLAN.md`).

**Do not dispatch this plan until `webtyp.com/keyring` has published**
`Scheme`/`IsReference` (root package) and `auto.OpenForModule` (its own
`docs/PLAN.md` in the same wave).

## Why

A `.env` value that is a credential must never sit in the file in plaintext —
the convention is `KEY=keyring://`, and `webtyp/keyring` now owns that
marker (`keyring.Scheme`/`keyring.IsReference`) and knows how to resolve it
for "whatever Go module the current project is"
(`keyring/auto.OpenForModule`). This package is the **one** place every
consumer (`veltylabs/misitio`, `veltylabs/iam`, any future project) already
calls to read configuration — resolving the marker here means no consumer
changes a single line of its own code; `env.Get("IAM_CLIENT_SECRET")` just
starts returning the real value once it is stored in the keyring, regardless
of whether the process was launched by `webtyp -mcp`, `webtyp -tui`, or a
bare `go run`.

**This must never reach a Cloudflare Worker build.** `env_wasm.go`'s `Lookup`
already reads from `context.env` (real Worker secrets/bindings), never from a
`.env` file — there is no marker to resolve there. But this plan still adds a
defensive check on that path too: see "wasm side" below.

## The change

### Native side — file **[`env_native.go`](env_native.go)**

Add one unexported helper, and call it from both existing lookup paths:

```go
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
```

Then in `Lookup`, replace both existing `return v, true` / `return v, true`
result points (the `os.LookupEnv` hit, and the `.env` line match) with a call
through this helper:

```go
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
```

Apply the exact same substitution to `LookupAt(key, path string)` — it is
today's `Lookup` with `defaultDotEnvPath` replaced by the `path` parameter;
change its two `return v, true` points the same way, calling
`resolveIfReference(key, v)`.

`Set(key, value string) error` (`os.Setenv`) is **unchanged** — writing
`os.Setenv("IAM_CLIENT_SECRET", "keyring://")` and later reading it back
through `Lookup` still resolves correctly, since `os.LookupEnv` is the first
branch checked.

### Wasm side — file **[`env_wasm.go`](env_wasm.go)**

Defensive only — this path never reads `.env`, but a Worker secret whose
value was mistakenly copy-pasted as the literal marker string must not be
treated as a real value (returning `"keyring://"` itself as if it were the
secret is a silent misconfiguration, worse than failing loudly):

```go
import (
	"syscall/js"

	"webtyp.com/fmt"
	"webtyp.com/keyring"
)

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
```

`webtyp.com/keyring` (root package only — never `keyring/auto`,
which pulls in OS-specific backends this build must not need) is a plain,
portable leaf package with no wasm-incompatible code — confirm with `GOOS=js
GOARCH=wasm go build ./...` in this repo before considering this stage done.

## Test hygiene — these tests touch the real OS keychain

`auto.OpenForModule` has no injectable fake backend (by design — see
`webtyp/keyring`'s own plan in this wave: `OpenKeyring`/`OpenForModule`
always probe the real platform provider). Every new test in this plan that
calls it must:

- Guard with a probe at the top: `if _, err := keyringauto.OpenForModule(".");
  err != nil { t.Skip("no keyring backend available: " + err.Error()) }` —
  matches the existing pattern this ecosystem already uses for an
  environment-dependent tool (`t.Skip` when unavailable rather than failing),
  e.g. `tinygo` in `veltylabs/misitio/tests/edge_size_test.go`.
- Use a **unique module name per test** in the `go.mod` it writes (e.g.
  `"keyring-ref-test-lookup"`, `"keyring-ref-test-lookupat"` — one per test
  function, not shared) so parallel or re-run test processes never collide on
  the same keychain entry.
- `t.Cleanup(func() { kr.Delete("SECRET") })` right after storing the probe
  value, so a failed run never leaves a stale entry that makes a *later* run
  of `TestLookupReferenceNotStoredIsNotFound` spuriously find a value.

## Anti-footguns

- **`keyring/auto` (the OS-backend-selecting package) is imported only from
  `env_native.go`**, never from `env_wasm.go` — that file imports the root
  `webtyp.com/keyring` package alone, for `IsReference` only.
- **Do not** cache the resolved keyring value across calls — `Lookup` already
  re-reads `.env`/`os.LookupEnv` on every call today (no caching exists), and
  this plan does not change that contract. If a caller wants to avoid
  repeated keyring round-trips, that is its own concern.
- **Do not** change `defaultDotEnvPath` or add a second `.env`-like file for
  "the keyring ones" — the marker lives inline in the same `.env`, next to
  every non-secret value, exactly as designed.

## Tests

File: **`keyring_reference_test.go`** (new, next to `env_test.go`).

`env_test.go`'s existing `.env`-backed tests (`TestLookup_FallsBackToDotEnv`,
`TestLookup_EnvVarWinsOverDotEnv`) isolate CWD with `dir := t.TempDir()` +
`os.Chdir(dir)` + `defer os.Chdir(orig)`, and write only a `.env` file into
`dir`. **`OpenForModule(".")` additionally needs a `go.mod` in that same
`dir`** (it has none by default) — every new test below must also
`os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module keyring-ref-test\n\ngo 1.25\n"), 0644)`
before calling `Lookup`, so the keyring service resolves to a fixed,
predictable `"keyring-ref-test"` instead of failing with
`auto.ErrGoModNotFound`.

| Test | Asserts |
|---|---|
| `TestLookupResolvesKeyringReference` | temp dir with `go.mod` (module `keyring-ref-test`) and `.env` containing `SECRET=keyring://`; before calling `Lookup`, store `"the-real-value"` under key `SECRET` via `keyringauto.OpenForModule(".")` (same CWD, so it resolves to the same service) — `Lookup("SECRET")` returns `("the-real-value", true)` |
| `TestLookupReferenceNotStoredIsNotFound` | `.env` has `SECRET=keyring://`, nothing stored under that key — `Lookup("SECRET")` returns `("", false)`, exactly like a key missing from `.env` altogether |
| `TestLookupLiteralValueUnaffected` | `.env` has `PLAIN=hello` — `Lookup("PLAIN")` returns `("hello", true)`, unchanged from today |
| `TestLookupAtResolvesKeyringReference` | same as the first test but through `LookupAt(key, explicitPath)` |
| `TestSetThenLookupResolvesKeyringReference` (native only) | `os.Setenv("SECRET", "keyring://")` (mirrors `Set`), value stored in keyring beforehand — `Lookup("SECRET")` returns the real value, proving the `os.LookupEnv` branch also resolves |

## Acceptance criteria

- [ ] `go build ./...` and `go vet ./...` clean (native).
- [ ] `GOOS=js GOARCH=wasm go build ./...` and `go vet` clean.
- [ ] `go test ./...` (native) green, including all new tests, and every
      pre-existing test still passes unchanged for literal (non-marker)
      values.
- [ ] `grep -n "keyringauto\|keyring/auto" env_wasm.go` → empty: the wasm
      build never imports the OS-backend-selecting package.
- [ ] `grep -n "resolveIfReference" env_native.go` → three call sites (the
      helper's own definition plus one call each in `Lookup` and
      `LookupAt`).

## Out of scope

`webtyp/app`'s interactive section for actually storing a missing secret
is a separate plan in this wave, dispatched after this one and after
`webtyp/keyring` publish (this plan and that one both depend on
`webtyp/keyring`, not on each other). No consumer project
(`veltylabs/misitio`, etc.) is touched here.
