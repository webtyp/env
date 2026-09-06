# env
<img src="docs/img/badges.svg">

Auto-tagged env access: `!wasm` reads `os.Getenv` (+ `.env` fallback), `wasm` reads Cloudflare `context.env` via `syscall/js`. No injection — the build tag selects the implementation. `webtyp/fmt` is the only dep, `os` never leaks into wasm.

```go
import "webtyp.com/env"

port := env.Get("PORT")                 // "" if unset
dsn, err := env.Require("DATABASE_URL") // error if unset
host := env.GetOr("HOST", "localhost")
if v, ok := env.Lookup("KEY"); ok { ... }
env.Set("KEY", "value") // !wasm only
flag := env.Arg("port") // !wasm only, -port=8080 / -port 8080
```

Low-level `.env` path override: `env.LookupAt("KEY", "/path/to/.env")`.

Wasm target is Cloudflare Workers (`context.env`); other wasm targets get `""`.

A value of exactly `keyring://` (`webtyp.com/keyring`'s `Scheme`) is
a marker, not a literal: `!wasm` `Lookup`/`LookupAt` resolve it through
`keyring/auto.OpenForModule(".")` — the current module's own `go.mod` path is
the keyring namespace, so two projects never collide on the same key name.
Nothing to store it yet → `ok=false`, same as the key being absent from
`.env` altogether. `wasm` never resolves it (there is no OS keyring in a
Worker) — a Worker secret whose value literally is `"keyring://"` is treated
as not found rather than served as-is. Filling in a missing one is
`webtyp/app`'s job (`webtyp -tui`'s SECRETS section), not this package's.
