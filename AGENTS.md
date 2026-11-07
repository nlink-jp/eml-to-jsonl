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
  email.go                     ← Email struct; Parse() maps mailparse.Message onto it
  parser_test.go               ← unit tests (the original fixtures)
  defects_test.go              ← the defects mailparse fixed, and the mapping rules
testdata/                      ← fixtures; testdata/defects/ are copies of mailparse's
```

## Gotchas

- **Reading the message is mailparse's job** (`github.com/nlink-jp/mailparse`, lib-series; it
  depends on `golang.org/x/text` and `golang.org/x/net`). Fix a parsing defect there, not here;
  this package only maps the result onto the JSONL shape.
- **The JSONL is read by downstream tools**: keep existing fields and their meaning; put new
  information in new, omitempty fields. Compare before/after output when changing the mapping.
- **Never format a mailbox by joining `Name <addr>`; use `Mailbox.String()`**: a decoded display
  name can hold an address form, and only the quoting keeps it from reading as a second address.
- **Received, X-Mailer, In-Reply-To, References, Message-ID use `Field.Collapsed()`** (whitespace
  runs made one space), as net/textproto did; mailparse's `Value` keeps the folding tab.
- **Detection**: `Options{AssumeEML: true}` reads minimal mail `mailparse.Detect` does not recognize
  (`From: root`), with a `not_detected_as_mail` warning. .msg/TNEF/OLE are refused before parsing.
- **Module path**: `github.com/nlink-jp/eml-to-jsonl`.
