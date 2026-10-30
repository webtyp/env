---
PLAN: "feat: KEY_FILE — read a secret from the file Docker mounts, before .env and the keyring"
EXECUTOR: jules
REVIEWER: none
STATUS: review
SESSION: 1610340303254169291
PR: https://github.com/webtyp/env/pull/2
REVIEWED_COMMIT: 8c559818680fab09ee31a0d561c3ae51e73c3d74
---

> Read AGENTS.md at the repo root before any change.
> **Finish by yourself:** when the delivery check passes, commit, push and open the pull request. Never ask whether to
> proceed or finalize — nobody answers, and a session that waits ends without your work.

# `KEY_FILE`

## Context

A self-hosted webtyp app runs in Docker Compose. Docker hands each secret to the container as a **file**
(`/run/secrets/<name>`), never as an environment variable, so nobody can read it with `docker inspect`. The app,
however, reads its configuration with `env.Get("DATABASE_URL")`. Decided: `webtyp/env` follows the standard `_FILE`
convention (the official PostgreSQL image does it with `POSTGRES_PASSWORD_FILE`): when `KEY` is not in the process
environment but `KEY_FILE` is, the value is the content of that file. The app does not change a line; its compose sets
`DATABASE_URL_FILE=/run/secrets/database_url`. Native only: the WASM side (Cloudflare `context.env`) has no files.

## Tools

```bash
go install webtyp.com/devflow/cmd/gotest@latest
```

Environment: `GOPRIVATE=webtyp.com/*,github.com/veltylabs/*`, `GOFLAGS=-mod=mod`.

## Design gate

1. **Prior art.** The official `postgres` and `mysql` Docker images read `POSTGRES_PASSWORD_FILE`/`MYSQL_PASSWORD_FILE`;
   Docker's secrets documentation recommends the `_FILE` suffix (unverified: pages not read in this session).
2. **Novice name.** No new API: `env.Get("DATABASE_URL")` finds `DATABASE_URL_FILE`.
3. **Complexity ledger.** Places a value can come from: +1 (between the process environment and `.env`).
4. **Where it belongs.** `env`, the one reader of configuration.
5. **Deletes.** Nothing.

## New lookup order (native)

1. `KEY` in the process environment.
2. `KEY_FILE` in the process environment → the file's content, with **one** trailing `\n` or `\r\n` removed.
   If the file cannot be read: write one line to standard error,
   `env: KEY_FILE=<path> cannot be read: <error>`, and report KEY as not found (never fall through to `.env`: a
   configured secret file that is missing is an error, not a reason to use another value).
3. `.env` in the working directory.
4. The module's keyring.

## Stage 1 — tests first: `tests/file_test.go` (new, `//go:build !wasm`, `package tests`)

Use `t.Setenv` and files in `t.TempDir()`; run each test from an empty temp dir (reuse `setupModule` from
`tests/lookup_order_test.go`) so no `.env` interferes.

| Test | Setup | Expected |
|---|---|---|
| `TestFile_ReadsSecret` | file `s3cret\n`; `t.Setenv("SECRET_X_FILE", path)` | `env.Get("SECRET_X") == "s3cret"` |
| `TestFile_CRLF` | file `s3cret\r\n` | `"s3cret"` |
| `TestFile_KeepsInnerNewlines` | file `a\nb\n` | `"a\nb"` |
| `TestFile_DirectWins` | `SECRET_X=direct` and `SECRET_X_FILE=<file with other>` | `"direct"` |
| `TestFile_BeforeDotEnv` | `.env` with `SECRET_X=fromdotenv` in the temp dir, and `SECRET_X_FILE=<file with fromfile>` | `"fromfile"` |
| `TestFile_Unreadable` | `SECRET_X_FILE=<temp dir>/missing`, and `.env` with `SECRET_X=fromdotenv` | `env.Lookup("SECRET_X")` → `"", false` (the `.env` value is NOT used); `env.Require("SECRET_X")` error contains `SECRET_X_FILE` |

The existing `tests/lookup_order_test.go` must keep passing. Commit after this stage (red).

## Stage 2 — `env_native.go` and `require_native.go`

In `env_native.go` add:

```go
// fileSuffix names the variable that points to a file holding the value (Docker secrets).
const fileSuffix = "_FILE"

// lookupFile reads KEY from the file KEY_FILE names. found reports whether KEY_FILE is set;
// a file that cannot be read is reported on stderr and gives ok == false.
func lookupFile(key string) (value string, found, ok bool) {
	path, found := os.LookupEnv(key + fileSuffix)
	if !found {
		return "", false, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		os.Stderr.WriteString("env: " + key + fileSuffix + "=" + path + " cannot be read: " + err.Error() + "\n")
		return "", true, false
	}
	v := string(data)
	v = fmt.Convert(v).TrimSuffix("\n").String()
	v = fmt.Convert(v).TrimSuffix("\r").String()
	return v, true, true
}
```

and make `Lookup`:

```go
func Lookup(key string) (string, bool) {
	if v, ok := os.LookupEnv(key); ok {
		return v, true
	}
	if v, found, ok := lookupFile(key); found {
		return v, ok
	}
	return lookupEnvFileAndKeyring(key, defaultDotEnvPath, ".")
}
```

Update `Lookup`'s comment to the order above. In `require_native.go`, the error text becomes:
`key + " is not set (checked the process environment, " + key + "_FILE, " + defaultDotEnvPath + " in " + cwd + ", and the keyring of this module)"`
(use `fileSuffix`, not a literal). If `webtyp.com/fmt`'s `Convert(...).TrimSuffix` does not exist, use the stdlib
`strings.TrimSuffix` instead (this file is native-only) and say so in the PR.

Run `gotest`: green. Commit.

## Stage 3 — docs

`README.md:21`: the lookup order gains `KEY_FILE` (step 2) with one sentence on Docker secrets and the stderr line for an
unreadable file. Commit.

## Delivery check

1. `gotest` passes (native and wasm). A PR that does not compile, or whose tests could not run, is a failed delivery.
2. `git diff --stat main` lists only `env_native.go`, `require_native.go`, `tests/file_test.go`, `README.md`.

## Delivery rules

- Commit after every stage. Never run `git reset`, `git restore`, `git checkout -- <file>` or `git stash` on your own work.
- Stuck for 30 minutes: open the PR anyway, listing what fails.
- Do not touch the wasm files.

## Stages

| # | Files | Result |
|---|---|---|
| 1 | `tests/file_test.go` | Every rule of the new order, red |
| 2 | `env_native.go`, `require_native.go` | `KEY_FILE`; green |
| 3 | `README.md` | Documented |
