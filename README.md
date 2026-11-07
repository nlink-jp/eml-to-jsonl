# eml-to-jsonl

EML parser for shell pipelines.
Reads `.eml` files and outputs structured JSONL — one JSON object per message — to stdout.
Designed to compose with [lagent](https://github.com/nlink-jp/lagent) and other tools for email analysis pipelines.

## Features

- Reads the message with [mailparse](https://github.com/nlink-jp/mailparse), the organization's
  mail parser written for messages composed by an attacker; msg-to-jsonl prints the same schema
- Headers: `from`, `sender`, `reply_to`, `return_path`, `to`, `cc`, `bcc`, `subject`, `date`,
  `message_id`, `in_reply_to`, `references`, `x_mailer`, `received` — every occurrence of an address
  field, mailboxes written in RFC 5322 form so an encoded display name cannot pass for an address
- Decodes all content to **UTF-8**; records the original charset in the `encoding` field
- Japanese charsets: ISO-2022-JP, Shift_JIS, EUC-JP (and any charset `golang.org/x/text` knows)
- Attachment metadata (filename, MIME type, size); a forwarded message is parsed into the same shape
- `warnings`: what the parser met and handled instead of failing (stable mailparse codes)
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

# Pipe into lagent for analysis (the whole stream is one attachment, up to 256 KiB)
eml-to-jsonl inbox/ | lagent -p "Summarise each email in one sentence."
```

## Output format

Each message produces one JSON line (eml-to-jsonl and msg-to-jsonl print the same schema):

```json
{
  "source": "inbox/message.eml",
  "message_id": "<abc123@example.com>",
  "in_reply_to": "<xyz@example.com>",
  "references": "<first@example.com> <xyz@example.com>",
  "from": "Alice <alice@example.com>",
  "sender": "list@example.org",
  "reply_to": ["\"Sales, Tokyo\" <sales@example.com>"],
  "return_path": "bounce@example.com",
  "to": ["Bob <bob@example.com>"],
  "cc": ["Carol <carol@example.com>"],
  "subject": "Hello World",
  "date": "2026-03-27T10:00:00+09:00",
  "x_mailer": "Apple Mail",
  "received": [
    "from mx2.example.com by mx1.example.net; Mon, 30 Mar 2026 09:00:00 +0900",
    "from sender.example.org by mx2.example.com; Mon, 30 Mar 2026 08:59:55 +0900"
  ],
  "encoding": "ISO-2022-JP",
  "body": [
    {"type": "text/plain", "content": "Hello..."},
    {"type": "text/html",  "content": "<html>...</html>"}
  ],
  "attachments": [
    {"filename": "report.pdf", "mime_type": "application/pdf", "size": 102400},
    {"filename": "", "mime_type": "message/rfc822", "size": 2048,
     "message": {"message_id": "<fwd@example.org>", "subject": "Original", "body": [], "attachments": [], "warnings": []}}
  ],
  "warnings": [
    {"code": "duplicate_field", "detail": "From appears 2 times; every occurrence is kept"}
  ]
}
```

**Addresses:** every mailbox of every occurrence of the field. `from` and `sender` are one string
(mailboxes joined by `, `); `to`, `cc`, `bcc` and `reply_to` are arrays. A display name holding
anything but atom characters and spaces is written as a quoted string, so a name that decodes to
`Bank <security@bank.example>` stays a name. `return_path` is `<>` for a bounce.

**Body:** the `text/plain` and `text/html` parts, in message order. Other text parts
(`text/calendar`, `text/csv`) are attachments.

**Encoding field:** the original charset of the first body that is not UTF-8 or US-ASCII
(e.g. `ISO-2022-JP`); omitted otherwise.

**Attachments:** metadata only (filename, MIME type, decoded byte size); binary content is not
embedded. A filename is empty when the message gives none. An attached message (`message/rfc822`,
or a message recognized by its content) is parsed into `message`, in the same shape (without
`source`), up to three levels deep.

**Warnings:** what mailparse met and handled in this message instead of failing — a duplicated
header, a malformed address, a broken encoding, a limit — as `{code, detail, part}`. The codes are
stable; see the [mailparse README](https://github.com/nlink-jp/mailparse). Always an array.
`not_detected_as_mail` marks minimal mail that was read although it was not recognized as mail.

**Refused input:** an Outlook `.msg` or a `winmail.dat` (use msg-to-jsonl), input with no header
field, and messages larger than 64 MiB. The error goes to stderr and the other files are still
processed.

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

- [docs/en/design/overview.md](docs/en/design/overview.md) — architecture and design decisions
- [docs/en/setup.md](docs/en/setup.md) — detailed setup guide
- [docs/en/dependencies.md](docs/en/dependencies.md) — third-party dependencies

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

# Pipeline: PST → EML → JSONL → LLM analysis, one gem-agent run per email
# (a mailbox outgrows the 256 KiB a single piped attachment can carry)
readpst -e -o /tmp/pst-output/ archive.pst && \
  eml-to-jsonl /tmp/pst-output/ | while IFS= read -r mail; do
    printf '%s\n' "$mail" | gem-agent -p "Summarize this email"
  done
```

This approach also works with [msg-to-jsonl](https://github.com/nlink-jp/msg-to-jsonl) —
use `readpst -m` to output MSG format instead of EML.

## Part of util-series

eml-to-jsonl is part of the [util-series](https://github.com/nlink-jp/util-series) —
a collection of lightweight CLI tools for working with local and cloud LLMs.
