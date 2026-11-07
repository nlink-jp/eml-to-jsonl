# Design Overview

## Purpose

eml-to-jsonl reads RFC 5322 (.eml) messages and outputs structured JSONL to stdout.
It is designed as a Unix filter that feeds into downstream analysis tools such as lagent.
msg-to-jsonl prints the same schema for Outlook .msg files.

## Input handling

| Source | How |
|--------|-----|
| stdin | Default when no arguments are given |
| File argument | `eml-to-jsonl file.eml` |
| Directory argument | `eml-to-jsonl dir/` — globs `*.eml` directly in the directory (non-recursive) |
| Mixed | `eml-to-jsonl dir/ extra.eml` — processed in order |

Errors on individual files are reported to stderr; processing continues with remaining files.
Exit code is 1 if any file failed.

## Output format

One JSON object per message on stdout. The `--pretty` flag switches to indented multi-line JSON.
`json.Encoder.SetEscapeHTML(false)` is used so that HTML in body fields is not entity-escaped.

## Parsing

All parsing is done by [mailparse](https://github.com/nlink-jp/mailparse); this tool holds none of
its own.

```
reader (file / stdin)
  └─ read up to 64 MiB + 1 byte          — mailparse.DefaultMaxSize; more is ErrTooLarge
       ├─ mailparse.Detect()              — .msg / TNEF → refused with a pointer to msg-to-jsonl
       └─ mailparse.ParseBytes(AssumeEML) — headers, addresses, MIME, charsets, nested messages
            └─ parser.FromMessage()       — the shared mapping below
```

`AssumeEML` is set because the caller says the input is mail: minimal mail that format detection
does not recognize (a cron message with only a Subject, say) is read with a `not_detected_as_mail`
warning instead of being refused.

## Mapping (identical in eml-to-jsonl and msg-to-jsonl)

`internal/parser/email.go` maps a `mailparse.Message` onto the JSONL record. The file is the same in
both tools, byte for byte, so the two print one schema; only the entry (`parse.go`) differs.

| Field | From mailparse |
|---|---|
| `from`, `sender` | every mailbox of every occurrence, `Mailbox.String()`, joined by `, ` |
| `to`, `cc`, `bcc`, `reply_to` | every mailbox of every occurrence, `Mailbox.String()`, one element each |
| `return_path` | `Mailbox.String()`; `<>` for a null reverse path |
| `date` | `Date` in RFC 3339, else `DateRaw` |
| `subject`, `message_id`, `in_reply_to`, `references` | as mailparse gives them |
| `x_mailer` | the first X-Mailer, `Field.Collapsed()` |
| `received` | every Received, `Field.Collapsed()`, in order |
| `encoding` | the first body charset other than UTF-8 / US-ASCII, upper-cased; a .msg Unicode property (UTF-16LE) is the container's storage and does not count |
| `body` | the message's bodies; a body recovered from compressed RTF (`Origin == "rtf"`) only when the message has no other body (in a .msg the RTF normally copies another body) |
| `attachments` | every attachment: `Filename`, `MediaType`, `Size`; an attached message becomes `message`, mapped the same way, without `source` |
| `warnings` | the message's own `Warnings` (a nested message carries its own); always an array |

`Mailbox.String()` is the only way an address becomes text: it quotes a display name that is not
plain atoms, so a name that decodes to an address form stays a name. No address is formatted here.
