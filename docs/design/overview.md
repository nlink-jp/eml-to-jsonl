# Design Overview

## Purpose

lite-eml parses RFC 2822 EML files and outputs structured JSONL to stdout.
It is designed as a Unix filter that feeds into downstream analysis tools such as lite-llm.

## Input handling

| Source | How |
|--------|-----|
| stdin | Default when no arguments are given |
| File argument | `lite-eml file.eml` |
| Directory argument | `lite-eml dir/` — globs `*.eml` directly in the directory (non-recursive) |
| Mixed | `lite-eml dir/ extra.eml` — processed in order |

Errors on individual files are reported to stderr; processing continues with remaining files.
Exit code is 1 if any file failed.

## Output format

One JSON object per message on stdout. The `--pretty` flag switches to indented multi-line JSON.
`json.Encoder.SetEscapeHTML(false)` is used so that HTML in body fields is not entity-escaped.

## Parsing pipeline

```
reader (file / stdin)
  └─ net/mail.ReadMessage()       — RFC 2822 header + body split
       ├─ extractHeaders()        — decode RFC 2047 headers to UTF-8
       │    └─ mime.WordDecoder   — custom CharsetReader via golang.org/x/text
       └─ parseBody()             — MIME body dispatch
            ├─ simple part        — decodeTransfer() → decodeToUTF8() → BodyPart
            └─ multipart/*        — recursive parseMultipart()
                 ├─ text parts    → BodyPart (text/plain first, text/html second)
                 └─ attachments   → Attachment (filename, MIME type, decoded size)
```

## Charset handling

All text content is converted to UTF-8 before output.

**Headers:** RFC 2047 encoded words (`=?charset?encoding?text?=`) are decoded by
`mime.WordDecoder` with a custom `CharsetReader` that uses `golang.org/x/text/encoding/htmlindex`
to look up any IANA-registered charset including ISO-2022-JP, Shift_JIS, EUC-JP.

**Body parts:** The `charset` parameter from `Content-Type` is used to construct a
decoder from `htmlindex.Get()`, which is applied via `golang.org/x/text/transform`.

**Encoding field:** Set to the uppercased charset of the first non-ASCII text body part
encountered. Omitted if all parts are UTF-8 or ASCII.

## Transfer encoding

| Encoding | Handler |
|----------|---------|
| `base64` | `base64.NewDecoder` with a whitespace-stripping wrapper |
| `quoted-printable` | `mime/quotedprintable.NewReader` |
| `7bit` / `8bit` / `binary` / empty | read as-is |

## Attachment detection

A MIME part is treated as an attachment when:
- `Content-Disposition` starts with `attachment`, or
- `Content-Disposition` starts with `inline` and carries a `filename` parameter, or
- the media type is not `text/*`.

Only metadata is collected (filename, MIME type, decoded byte size).
Binary content is not included in the output to keep JSONL lines compact.

## Body ordering

For `multipart/alternative`, parts are emitted in the order they appear in the message.
The EML specification requires `text/plain` to appear before `text/html`, so this
ordering is preserved naturally. No reordering is performed.
