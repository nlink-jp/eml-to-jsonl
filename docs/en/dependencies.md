# Dependencies

## Runtime

| Package | Version | Purpose | Why not in-house |
|---------|---------|---------|-----------------|
| `github.com/nlink-jp/mailparse` | v0.2.0 | Reads the message: RFC 5322 headers and addresses, MIME, transfer encodings, charsets, attached messages | It is in-house: the organization's mail parser, shared with msg-to-jsonl, mail-analyzer and mail-analyzer-local so they read mail the same way. |
| `golang.org/x/text` | (through mailparse) | Charset conversion (ISO-2022-JP, Shift_JIS, EUC-JP, Windows code pages) | The canonical Go charset library, maintained by the Go team. |
| `golang.org/x/net` | (through mailparse) | `html` charset detection from `<meta charset>` | Maintained by the Go team. |

No community package is used.

## Standard library packages used

| Package | Purpose |
|---------|---------|
| `encoding/json` | JSONL output |
| `path/filepath` | Directory globbing |

## Development

| Tool | Purpose |
|------|---------|
| `golangci-lint` | Static analysis |
| `govulncheck` | Dependency vulnerability scanning |
