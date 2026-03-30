# eml-to-jsonl

EML parser for shell pipelines.
Reads `.eml` files and outputs structured JSONL — one JSON object per message — to stdout.
Designed to compose with [lite-llm](https://github.com/nlink-jp/lite-llm) and other tools for email analysis pipelines.

## Features

- Extracts headers: `from`, `to`, `cc`, `bcc`, `subject`, `date`, `message_id`, `in_reply_to`, `x_mailer`
- Decodes all content to **UTF-8**; records original charset in the `encoding` field
- Handles multipart bodies: `text/plain` preferred, `text/html` included when present
- Decodes `base64`, `quoted-printable`, `7bit`, and `8bit` transfer encodings
- Supports Japanese charsets: ISO-2022-JP, Shift_JIS, EUC-JP (and any IANA-registered charset)
- Attachment metadata (filename, MIME type, size) without embedding binary content
- Input: stdin, file arguments, or directory (processes all `*.eml` in the directory)

## Installation

```sh
git clone https://github.com/nlink-jp/eml-to-jsonl.git
cd eml-to-jsonl
make build
# Add dist/ to PATH or copy dist/eml-to-jsonl to a directory on PATH
```

## Usage

```sh
# Single file
eml-to-jsonl message.eml

# Multiple files
eml-to-jsonl mail1.eml mail2.eml

# Directory (processes all *.eml)
eml-to-jsonl ~/exported-mail/

# Stdin
cat message.eml | eml-to-jsonl

# Pretty-print for inspection
eml-to-jsonl --pretty message.eml

# Pipe into lite-llm for analysis
eml-to-jsonl inbox/ | lite-llm -p "Summarise each email in one sentence."
```

## Output format

Each message produces one JSON line:

```json
{
  "source": "inbox/message.eml",
  "message_id": "<abc123@example.com>",
  "in_reply_to": "<xyz@example.com>",
  "from": "Alice <alice@example.com>",
  "to": ["Bob <bob@example.com>"],
  "cc": ["Carol <carol@example.com>"],
  "bcc": [],
  "subject": "Hello World",
  "date": "2026-03-27T10:00:00+09:00",
  "x_mailer": "Apple Mail",
  "encoding": "ISO-2022-JP",
  "body": [
    {"type": "text/plain", "content": "Hello..."},
    {"type": "text/html",  "content": "<html>...</html>"}
  ],
  "attachments": [
    {"filename": "report.pdf", "mime_type": "application/pdf", "size": 102400}
  ]
}
```

**Body ordering rules:**
1. `text/plain` is always listed first if present.
2. `text/html` follows.
3. If the message is HTML-only, a single `text/html` part is output.

**Encoding field:**
Set to the original charset of the primary text part (e.g. `ISO-2022-JP`).
Omitted when the message is already UTF-8 or ASCII.

**Attachments:**
Only metadata is included (filename, MIME type, decoded byte size).
Binary content is not embedded in the output.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-pretty` | false | Pretty-print JSON instead of JSONL |
| `-version` | — | Print version and exit |

## Building

```sh
make build       # current platform
make build-all   # all release platforms → dist/
make test        # run tests
make check       # vet + lint + test + build + govulncheck
```

## Documentation

- [docs/design/overview.md](docs/design/overview.md) — architecture and design decisions
- [docs/setup.md](docs/setup.md) — detailed setup guide
- [docs/dependencies.md](docs/dependencies.md) — third-party dependencies

## Working with PST files

Outlook PST (Personal Storage Table) files can be converted to EML using
[readpst](https://www.five-ten-sg.com/libpst/) and then processed with eml-to-jsonl:

```sh
# Install readpst
# macOS:        brew install libpst
# Debian/Ubuntu: apt install pst-utils

# Convert PST to EML files
readpst -e -o /tmp/pst-output/ archive.pst

# Parse all extracted EML files
eml-to-jsonl /tmp/pst-output/

# Pipeline: PST → EML → JSONL → LLM analysis
readpst -e -o /tmp/pst-output/ archive.pst && \
  eml-to-jsonl /tmp/pst-output/ | gem-cli --batch -s "Summarize each email"
```

This approach also works with [msg-to-jsonl](https://github.com/nlink-jp/msg-to-jsonl) —
use `readpst -m` to output MSG format instead of EML.

## Part of util-series

eml-to-jsonl is part of the [util-series](https://github.com/nlink-jp/util-series) —
a collection of lightweight CLI tools for working with local and cloud LLMs.
