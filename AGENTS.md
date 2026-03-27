# AGENTS.md — lite-switch

Natural language classifier for shell pipelines.
Reads stdin, outputs one tag to stdout.
Part of [lite-series](https://github.com/nlink-jp/lite-series).

## Rules

- Project rules (security, testing, docs, release, etc.): → [RULES.md](RULES.md)
- Series-wide conventions (config format, Makefile, etc.): → [lite-series CONVENTIONS.md](https://github.com/nlink-jp/lite-series/blob/main/CONVENTIONS.md)

## Build & test

```sh
make build    # bin/lite-switch
make check    # vet → lint → test → build → govulncheck
go test ./... # tests only
```

## Key structure

```
main.go                     ← flag parsing, stdin reading, wiring
internal/config/            ← loads config.toml (TOML) + switches.yaml (YAML), env overrides
internal/llm/               ← HTTP client (retry + backoff), prompt building, input wrapping
internal/classifier/        ← tool calling, tag extraction (4-strategy fallback chain)
```

## Gotchas

- **Two config files, two formats**: `config.toml` (TOML, API settings) and `switches.yaml` (YAML, classification data). Do not mix them up.
- **Switches file is not a system config**: it belongs next to the project, not in `~/.config/`. It is safe to version-control (no secrets).
- **No Cobra**: uses the standard `flag` package. No subcommands.
- **Endpoint normalisation**: `base_url` accepts with or without `/v1`; `client.endpoint()` handles both.
- **Fallback chain**: tool call → JSON in content → tag string in content → last switch. The last switch acts as a catch-all default.
- **Module path**: `github.com/nlink-jp/lite-switch`.
- **Env vars**: `LITE_SWITCH_BASE_URL`, `LITE_SWITCH_API_KEY`, `LITE_SWITCH_MODEL`.
