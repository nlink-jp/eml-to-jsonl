# Dependencies

## Runtime

| Package | Version | Purpose | Why not in-house |
|---------|---------|---------|-----------------|
| `golang.org/x/text` | v0.35.0 | Charset detection and conversion (ISO-2022-JP, Shift_JIS, EUC-JP, and all IANA-registered charsets) | Implementing correct charset conversion for dozens of encodings is a significant undertaking; the x/text package is the canonical Go solution, maintained by the Go team. |

## Standard library packages used

| Package | Purpose |
|---------|---------|
| `net/mail` | RFC 2822 message parsing (headers + body split) |
| `mime` | MIME media type parsing and RFC 2047 word decoding |
| `mime/multipart` | MIME multipart body parsing |
| `mime/quotedprintable` | Quoted-printable transfer encoding |
| `encoding/base64` | Base64 transfer encoding |
| `encoding/json` | JSONL output |
| `path/filepath` | Directory globbing and path manipulation |

## Development

| Tool | Purpose |
|------|---------|
| `golangci-lint` | Static analysis |
| `govulncheck` | Dependency vulnerability scanning |
