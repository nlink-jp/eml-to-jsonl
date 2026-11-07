# Dependencies

## Runtime

| Package | Version | Purpose | Why not in-house |
|---------|---------|---------|-----------------|
| `github.com/nlink-jp/mailparse` | v0.2.0 | Reading the message: headers in order with duplicates, addresses, the MIME tree, transfer encodings, charsets, nested messages, warnings | It is in-house: the org's shared mail library (lib-series), written for attacker-composed mail. This tool's own parser had the defects mailparse's Phase 0 record measured. |
| `golang.org/x/text` | v0.35.0 | Charset conversion (ISO-2022-JP, Shift_JIS, EUC-JP, and all IANA-registered charsets) — through mailparse | Go team; the canonical Go solution. |
| `golang.org/x/net` | v0.52.0 | HTML `<meta charset>` — through mailparse | Go team. |

## Standard library packages used

| Package | Purpose |
|---------|---------|
| `encoding/json` | JSONL output |
| `path/filepath` | Directory globbing and path manipulation |

## Development

| Tool | Purpose |
|------|---------|
| `golangci-lint` | Static analysis |
| `govulncheck` | Dependency vulnerability scanning |
