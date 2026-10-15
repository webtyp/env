---
PLAN: "feat!: Lookup reads the project keyring after the environment and .env; the keyring:// marker is gone"
EXECUTOR: jules
REVIEWER: none
STATUS: running
SESSION: 7810215043722152745
---

> Read AGENTS.md at the repo root before any change.

# Plan — env: secrets come from the keyring, never from a `.env` marker

Part of the "Despliegue guiado" wave (`https://github.com/webtyp/docs/blob/main/DEPLOY_WIZARD_MASTER_PLAN.md`,
decision D15). A secret is never written in `.env`, not even as a placeholder. Today a project had
to write `KEY=keyring://` in `.env` for `env` to look the key up in the keyring
(`env_native.go:19-35`). From now on the project declares its secrets in `config/deploy.go`
(read by `webtyp/app`, not by this module) and `env` simply looks in one more place, in a fixed
order.

No project uses the marker today (`grep -r "=keyring://"` over every `.env` → empty), so nothing
migrates.

## Tools

```bash
go install webtyp.com/devflow/cmd/gotest@latest
```

## Design gate

1. **Prior art.** direnv / dotenv load a file into the environment; Kamal 2 and 1Password CLI
   (`op run`) resolve secrets from a vault at process start; Rails credentials read an encrypted
   store after `ENV`. All define one lookup order. Here: environment → `.env` → the project's
   keyring.
2. **Names.** No new exported names.
3. **Ledger.** Concepts −1 (the `keyring://` marker). Ways to provide a secret locally: marker in
   `.env` + keyring → keyring only.
4. **Where.** In `env`: it already owns "where does a value come from".
5. **Deletes.** `resolveIfReference` and every use of `keyring.IsReference` / `keyring.Scheme`.

## Stage 1 — native lookup order

`env_native.go`:

- Delete `resolveIfReference`.
- `Lookup(key)`: process environment → `.env` in the working directory → keyring scoped to the
  module of the working directory (`keyringauto.OpenForModule(".")`, then `kr.Get(key)`). The first
  place that has the key wins and its value is returned as is. A keyring that cannot be opened, or
  that lacks the key, means "not found" (`"", false`), exactly as today.
- `LookupAt(key, path)`: same order, with `.env` read from `path` and the keyring scoped to the
  module found from `filepath.Dir(path)`.
- Extract the shared ".env file then keyring" part into one unexported function used by both
  (today the `.env` parsing is duplicated in `Lookup` and `LookupAt`; it must exist once).
- `require_native.go`: the error message lists the three places checked:
  `key + " is not set (checked the process environment, " + path + " in " + cwd + ", and the keyring of this module)"`
  as a named constant format.

## Stage 2 — wasm

`env_wasm.go`: delete the `keyring.IsReference` check and the `webtyp.com/keyring` import. A Worker
reads only its bindings.

## Stage 3 — tests

Replace `keyring_reference_test.go` with `tests/lookup_order_test.go` (package `tests`, external),
reusing its helpers (temporary module dir with `go.mod`, `os.Chdir`, `keyringauto.OpenForModule`).
If the platform keyring is unavailable in the test environment, skip with `t.Skip` and the exact
reason, as the current test does.

| Branch | Test |
|---|---|
| O1 | key in the process environment, `.env` and keyring → environment value |
| O2 | key in `.env` and keyring → `.env` value |
| O3 | key only in the keyring → keyring value |
| O4 | key nowhere → `"", false`; `Require` error names the three places |
| O5 | `.env` line `KEY=keyring://` → returns the literal `keyring://` (no special meaning any more) |
| O6 | `LookupAt` with a `.env` in another module dir reads that module's keyring |

Move `env_test.go` and `env_wasm_test.go` to `tests/` only if they use exported API alone; leave
them otherwise.

## Stage 4 — docs

`README.md`: the lookup order; remove every mention of `keyring://`. Add one line: secrets are
declared in the project's `config/deploy.go` and typed in the `webtyp` wizard (link
`https://github.com/webtyp/docs/blob/main/diagrams/DEPLOY_SECRETS_FLOW.md`).

| Stage | Files |
|---|---|
| 1 | `env_native.go`, `require_native.go` |
| 2 | `env_wasm.go`, `go.mod`, `go.sum` |
| 3 | `tests/lookup_order_test.go` (new); delete `keyring_reference_test.go` |
| 4 | `README.md` |

## Acceptance

- `gotest` green (native and wasm).
- `grep -rn "IsReference\|keyring.Scheme\|keyring://" --include=*.go --include=*.md . | grep -v "tests/lookup_order_test.go"` → empty.
- `git diff --stat main` lists every file of the stages table; only `keyring_reference_test.go` deleted.
