# Design Overview

## Purpose

eml-to-jsonl parses RFC 2822 EML files and outputs structured JSONL to stdout.
It is designed as a Unix filter that feeds into downstream analysis tools such as lagent.

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

## Parsing pipeline

```
reader (file / stdin)
  └─ parser.Parse()
       ├─ mailparse.Detect()      — msg / TNEF / OLE: refuse (use msg-to-jsonl)
       ├─ mailparse.ParseBytes()  — Options{AssumeEML: true}: minimal mail Detect does not
       │                            recognize is read with not_detected_as_mail; headers in order with duplicates, addresses parsed before
       │                            display names are decoded, the MIME tree, transfer encodings,
       │                            charsets, nested messages, warnings (limits shared across a parse)
       └─ fromMessage()           — maps mailparse.Message onto the JSONL record (recursively for
                                    nested messages)
```

Parsing defects are fixed in mailparse, not here. This package decides only the JSONL shape:

| JSONL | From mailparse |
|-------|----------------|
| `from` | the mailboxes of the first `From` field |
| `from_all` | every mailbox of every `From` field, when more than one |
| `to`, `cc`, `bcc`, `sender`, `reply_to`, `return_path` | every mailbox of every occurrence |
| `subject` | `Message.Subject` (first occurrence, RFC 2047 decoded and repaired) |
| `date` | `Message.Date` as RFC 3339, else `DateRaw` |
| `message_id`, `in_reply_to`, `references`, `x_mailer`, `received` | `Field.Collapsed()` (whitespace runs made one space, as net/textproto gave) |
| `body` | `Message.Bodies` (text/plain and text/html without a filename) |
| `encoding` | the first body charset that is not UTF-8 or US-ASCII, uppercased |
| `attachments` | `Message.Attachments`; `message` from `Attachment.Message` |
| `warnings` | `Message.Warnings` |

A mailbox is `Mailbox.String()`: RFC 5322 form that reads back as the same one mailbox, with a
display name quoted when it holds anything but atom characters and single spaces, so a decoded name
cannot read as a second address.

## Charset handling

All text content is converted to UTF-8 by mailparse: the declared charset, the HTML `<meta charset>`
when none is declared, or a guess for undeclared Japanese text — each guess or fallback with a warning.

## Limits

All limits are mailparse's defaults: messages over 64 MiB are refused (a 25 MB attachment is about
34 MB in Base64), 3 nested message layers, MIME depth 32, 1,000
parts and 120 MiB decoded across a parse.

## Body ordering

For `multipart/alternative`, parts are emitted in the order they appear in the message.
The EML specification requires `text/plain` to appear before `text/html`, so this
ordering is preserved naturally. No reordering is performed.
