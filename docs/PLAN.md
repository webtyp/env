---
PLAN: "fix: Require names the .env path it checked, so a missing key is self-diagnosing"
EXECUTOR: jules
REVIEWER: none
---

> This plan is dispatched via the CodeJob workflow. See skill: agents-workflow.

# Plan — a loud diagnostic instead of a bare "$KEY is not set"

Part of `DOTENV_VISIBILITY_MASTER_PLAN.md` (orchestrator: `webtyp.com/docs`,
`docs/DOTENV_VISIBILITY_MASTER_PLAN.md`). **No dependencies — dispatch any
time, independent of the other two modules in that wave.**

## Why

`env.Require` (`env.go:24-29`) returns `fmt.Err(key + " is not set")` when a
key is missing. That message is correct but incomplete: it never says
*where* it looked, so when the real cause is "looked in the wrong directory"
(the exact bug `DOTENV_VISIBILITY_MASTER_PLAN.md` diagnoses in
`webtyp/gorun`/`webtyp/server`) the error gives no hint at all — a junior
reads `DATABASE_URL is not set`, checks the `.env` file, sees the key is
right there, and has no way to tell from the message alone that the process
reading it has a different working directory than they assume.

`CONSTRUCTION_HARNESS.md` principle 6: "fail at compile time, not at
runtime... loud development diagnostic" over a silent/uninformative one.
This closes that gap for the one message every consumer of a required env
var eventually sees.

## What to change

`env/env_native.go` — `Lookup` currently does:

```go
data, err := os.ReadFile(defaultDotEnvPath)
if err != nil {
	return "", false
}
```

`Lookup` only returns `(string, bool)` — it has no path to report through.
Change `env.go`'s `Require` instead, so the behavior change is scoped to the
one function that already owns error construction, without touching
`Lookup`'s signature or its callers (`Get`, `GetOr` keep working exactly as
today):

```go
// Require returns the value of key or an error if unset.
func Require(key string) (string, error) {
	if v, ok := Lookup(key); ok {
		return v, nil
	}
	cwd, _ := os.Getwd()
	return "", fmt.Err(key + " is not set (checked the process environment and " + defaultDotEnvPath + " in " + cwd + ")")
}
```

This needs `defaultDotEnvPath` (declared in `env_native.go`, same package,
already visible) and a new `"os"` import in `env.go` for `os.Getwd()` — `os`
is `!wasm`-only, but so is this whole reasoning; check whether `env.go` is
built for both `wasm` and `!wasm` (it has no build tag today, meaning it is
shared). **If `env.go` compiles under the `wasm` build tag, `os.Getwd()`
will not exist there** — in that case, split `Require` into two files
instead:

- `require_native.go` (`//go:build !wasm`): the version above, with
  `os.Getwd()`.
- `require_wasm.go` (`//go:build wasm`): keep today's exact message,
  `fmt.Err(key + " is not set")` — wasm has no meaningful CWD to report,
  and no `.env` file fallback either (per the package doc comment in
  `env.go:1-4`, wasm reads Cloudflare's `context.env`).

Check `env_wasm.go` first to confirm whether `wasm` even has a working
`os.Getwd()` (TinyGo's `syscall/js` target may or may not) before assuming
the split is required — if `os.Getwd()` compiles fine under the `wasm`
build tag in this codebase's TinyGo target, keep `Require` in the single
shared `env.go` and skip the split.

## Tests

`env/env_test.go` (native) — add a case: unset the key in both process env
and a temp-dir `.env`, call `Require`, assert the returned error's message
contains both the key name and `.env` (exact CWD value is environment-
dependent, so assert on substring, not full equality).

If the wasm split above turns out to be necessary, `env_wasm_test.go` keeps
asserting today's exact message for that build — do not change the wasm
test's expected string.

## Acceptance

- `go test ./...` passes for both build tags (`go test .` and
  `GOOS=js GOARCH=wasm go test .` — check the existing test invocation this
  repo already uses for the wasm target, e.g. in a Makefile or CI config,
  rather than guessing the flags).
- A manual `Require("DOES_NOT_EXIST")` call in a throwaway `main.go` prints
  a message naming both the key and a resolved directory path.

## Stages

| Stage | Files | Done when |
|---|---|---|
| 1 | `env.go` (or `require_native.go` + `require_wasm.go`, per the build-tag check above) | `Require`'s native error names the checked path |
| 2 | `env_test.go` | new substring-matching test passes |
