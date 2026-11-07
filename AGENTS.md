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
make verify-release  # gate: notarized, fresh, runs at this version, clean linux archives (run before upload)
go test ./... # tests only
```

## Key structure

```
main.go                        ← entry point, flag parsing, stdin/file reading
internal/parser/
  parse.go                     ← Parse(io.Reader): reads ≤64 MiB+1, refuses .msg/TNEF, mailparse.ParseBytes(AssumeEML)
  parse_test.go                ← entry tests
  email.go                     ← Email schema + FromMessage (mailparse.Message → Email)
  email_test.go                ← schema tests
```

`email.go` and `email_test.go` are **identical, byte for byte, in eml-to-jsonl and msg-to-jsonl**:
the two tools print one schema. Change them in both repositories together (`cmp` them).

## Gotchas

- **All parsing is mailparse** (`github.com/nlink-jp/mailparse`): no own header, MIME, charset or
  CFB code. A parsing defect is fixed in mailparse and released there first, never worked around here.
- **Addresses become text only through `Mailbox.String()`**: never join `Name` and `Address` yourself
  (that re-creates the display-name spoof mailparse prevents).
- **`warnings` is always an array** and belongs to its own message; a nested message carries its own.
- **Size limit is mailparse's** (`DefaultMaxSize`, 64 MiB): read one byte past it and let `ErrTooLarge` speak.
- **Fixtures are fictional** (RFC 2606 domains); .msg/TNEF fixtures are built with `mailparse/msgtest`.
- **Module path**: `github.com/nlink-jp/eml-to-jsonl`.
