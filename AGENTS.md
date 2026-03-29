# AGENTS.md — eml-to-jsonl

EML parser for shell pipelines.
Reads `.eml` files and outputs structured JSONL — one JSON object per message — to stdout.
Part of [util-series](https://github.com/nlink-jp/util-series).

## Rules

- Project rules (security, testing, docs, release, etc.): → [RULES.md](RULES.md)
- Series-wide conventions: → [util-series CONVENTIONS.md](https://github.com/nlink-jp/util-series/blob/main/CONVENTIONS.md)

## Build & test

```sh
make build    # dist/eml-to-jsonl
make check    # vet → lint → test → build → govulncheck
go test ./... # tests only
```

## Key structure

```
main.go                        ← entry point, flag parsing, stdin/file reading
internal/parser/
  email.go                     ← Email struct, ToJSON()
  headers.go                   ← header extraction (From, To, Subject, Date, etc.)
  body.go                      ← MIME body parsing (text/plain, text/html)
  charset.go                   ← character encoding detection and conversion
  parser_test.go               ← unit tests
```

## Gotchas

- **No external dependencies**: uses only the Go standard library for MIME parsing.
- **Charset handling**: auto-detects and converts non-UTF-8 encodings (ISO-2022-JP, Shift_JIS, etc.).
- **Module path**: `github.com/nlink-jp/eml-to-jsonl`.
