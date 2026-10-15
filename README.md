# env
<img src="docs/img/badges.svg">

Auto-tagged env access: `!wasm` reads `os.Getenv` (+ `.env` fallback + module keyring), `wasm` reads Cloudflare `context.env` via `syscall/js`. No injection — the build tag selects the implementation. `webtyp/fmt` is the only dep, `os` never leaks into wasm.

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

Lookup order on `!wasm` is: process environment → `.env` file in the directory → the module's keyring namespace.
Secrets are declared in the project's `config/deploy.go` and typed in the `webtyp` wizard ([DEPLOY_SECRETS_FLOW](https://github.com/webtyp/docs/blob/main/diagrams/DEPLOY_SECRETS_FLOW.md)).
