# eml-to-jsonl

EML parser for shell pipelines.
Reads `.eml` files and outputs structured JSONL — one JSON object per message — to stdout.
Designed to compose with [lagent](https://github.com/nlink-jp/lagent) and other tools for email analysis pipelines.

## Features

- Extracts headers: `from`, `to`, `cc`, `bcc`, `reply_to`, `sender`, `return_path`, `subject`, `date`, `message_id`, `in_reply_to`, `references`, `x_mailer`, `received`
- Keeps every occurrence of an address field: a second `From` is reported (`from_all`), and every `To` / `Cc` field is read
- Parses addresses before decoding display names, so an encoded display name cannot pass for an address
- Opens forwarded messages (`message/rfc822`, or a message attached under another type) as `attachments[].message`
- Decodes all content to **UTF-8**; records original charset in the `encoding` field — declared, taken from HTML `<meta charset>`, or guessed for undeclared Japanese
- Handles multipart bodies: `text/plain` preferred, `text/html` included when present
- Decodes `base64`, `quoted-printable`, `7bit`, and `8bit` transfer encodings, keeping what decodes from damaged data
- Supports Japanese charsets: ISO-2022-JP, Shift_JIS, EUC-JP (and any IANA-registered charset)
- Attachment metadata (filename, MIME type, size) without embedding binary content
- Nothing is dropped silently: a malformed header line, an unclosed boundary, damaged Base64 or a guessed charset is reported in `warnings` and the rest of the message is still read
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

Each message produces one JSON line:

```json
{
  "source": "inbox/message.eml",
  "message_id": "<abc123@example.com>",
  "in_reply_to": "<xyz@example.com>",
  "references": "<xyz@example.com>",
  "from": "Alice <alice@example.com>",
  "to": ["Bob <bob@example.com>"],
  "cc": ["Carol <carol@example.com>"],
  "bcc": [],
  "subject": "Hello World",
  "date": "2026-03-27T10:00:00+09:00",
  "x_mailer": "Apple Mail",
  "received": [
    "from mx2.example.com by mx1.corp.com; Mon, 30 Mar 2026 09:00:00 +0900",
    "from sender.example.org by mx2.example.com; Mon, 30 Mar 2026 08:59:55 +0900"
  ],
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

Fields that appear only when the message has them:

```json
{
  "from_all": ["ceo@example.org", "attacker@example.net"],
  "sender": ["attacker@example.net"],
  "reply_to": ["Help Desk <help@example.net>"],
  "return_path": ["<bounce@example.org>"],
  "attachments": [
    {"filename": "original.eml", "mime_type": "message/rfc822", "size": 855,
     "message": {"from": "Support <support@example.net>", "subject": "...", "body": [], "attachments": []}}
  ],
  "warnings": [
    {"code": "unclosed_boundary", "detail": "a multipart ended without its closing delimiter; the parts read so far are kept", "part": "2"}
  ]
}
```

The message is read by [mailparse](https://github.com/nlink-jp/mailparse), a library written for
messages composed by an attacker.

**Address fields** (`from`, `to`, `cc`, `bcc`, `reply_to`, `sender`, `return_path`):
Each mailbox is written in RFC 5322 form (`Name <address>` or the bare address), so that reading it
again gives the same one mailbox. A display name holding anything but letters, digits, atom
characters and single spaces — `<`, `@`, `,`, a `.` as in `John Q. Public`, an ideographic space
(U+3000) — is written as a quoted string: `"Bank <security@bank.example>" <attacker@example.net>`. A
name cannot read as a second address. A mailbox with a name and no address is the quoted name alone. Every occurrence of a field is read: two `To` fields give
one list with the mailboxes of both. `from` is the first `From` field; `from_all` lists every mailbox of
every `From` field, and appears only when there is more than one. A field that does not parse as
RFC 5322 is read leniently (with an `invalid_address` warning). A null `Return-Path` is `"<>"`.

**Nested messages:**
An attachment that is itself a message — `message/rfc822`, or a message sent as
`application/octet-stream` or a `.eml` name and recognized by its content — carries the parsed message
in `message`, in the same shape (without `source`). Up to three levels are opened.

**Warnings:**
Everything the parse met and handled instead of failing — a header line with no colon, a duplicated
`From`, a repaired encoded-word, an unclosed boundary, Base64 with invalid characters, a guessed charset,
a limit that cut something — is listed in `warnings` with a stable `code` (mailparse's codes, in its
`warnings.go`) and the MIME part it applies to (`part`, omitted for the message header). The field is
omitted when there is nothing to report. One code repeated more than 100 times in a message is cut
there (`warning_limit`).

**Input recognition:**
Outlook `.msg` and TNEF (`winmail.dat`) input is refused with a pointer to msg-to-jsonl, and so is an
input with no header field. Minimal mail that does not carry the fields mail usually has (`From: root`
from cron, `To` and `Subject` alone) is still read, with a `not_detected_as_mail` warning. A UTF-8 byte
order mark before the header is skipped (`leading_bom`). Messages larger than 64 MiB are refused.

**Body ordering rules:**
1. `text/plain` is always listed first if present.
2. `text/html` follows.
3. If the message is HTML-only, a single `text/html` part is output.

Only `text/plain` and `text/html` parts without a filename are bodies; any other part — `text/calendar`,
`text/csv`, a text part with a name — is an attachment.

**Encoding field:**
Set to the original charset of the first body part that is not UTF-8 or ASCII (e.g. `ISO-2022-JP`):
the declared charset, the HTML `<meta charset>`, or a guessed one (a `charset_guessed` warning says so).
Omitted when the message is already UTF-8 or ASCII.

**Attachments:**
Only metadata is included (filename, MIME type, decoded byte size), plus `message` for a nested message.
Binary content is not embedded in the output. An attachment without a name is `"attachment"`.

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
