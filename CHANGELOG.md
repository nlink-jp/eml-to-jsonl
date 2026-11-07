# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).


## [0.6.1] - 2026-11-07

### Fixed

- **mailparse v0.3.0.** An address field that does not parse as RFC 5322 is
  now read by a lexer and a small grammar: every address in it is kept (a
  missing comma no longer drops one), and none is made up from a comment, a
  quoted display name or words glued together. A crafted .msg whose compressed
  RTF nests `\field` groups no longer panics, and unclosed parentheses in an
  address field no longer take quadratic time. Well-formed mail reads as
  before (18 real messages compared, no difference).

## [0.6.0] - 2026-11-07

The move to [mailparse](https://github.com/nlink-jp/mailparse) is redone, together with
msg-to-jsonl, mail-analyzer and mail-analyzer-local, after the separate v0.5.0 attempt was
withdrawn (it had no binaries; the tag stays). The changes below are against v0.4.0.

### Changed

- **The message is read by mailparse v0.2.0** instead of this tool's own parser on `net/mail`,
  `mime` and `mime/multipart`. eml-to-jsonl and msg-to-jsonl now print one schema from one shared
  mapping (`internal/parser/email.go`, identical in both). For the same input:
  - Address fields hold every mailbox of every occurrence of the field; `from` joins them with `, `.
  - Mailboxes are written in RFC 5322 form that reads back as the same mailbox: a display name with
    anything but atom characters and spaces is quoted, so a name that decodes to an address form stays
    a name. A comment in a From field is part of the display name.
  - Display names are decoded after the address is parsed (the old parser decoded first).
  - Only `text/plain` and `text/html` parts are bodies; `text/calendar`, `text/csv` and other text
    parts are attachments.
  - Bodies from quoted-printable keep the message's CRLF line ends, as other encodings did.
  - An attachment with no filename has an empty `filename` (it was `"attachment"`).
  - `source` is omitted for an attached message (always present at the top level).
- Input that is an Outlook `.msg` or a `winmail.dat` is refused with a pointer to msg-to-jsonl;
  input with no header field and messages larger than 64 MiB are refused.

### Added

- `sender`, `reply_to` (array), `return_path` (`<>` for a bounce) and `references`.
- `attachments[].message`: an attached message (`message/rfc822`, an embedded `.msg`, or a message
  recognized by its content) parsed into the same shape, without `source`, up to three levels deep.
- `warnings`: what the parser met and handled instead of failing, as `{code, detail, part}` with
  mailparse's stable codes. Always an array; a nested message carries its own.

### Fixed

- A header line without a colon no longer fails the whole message (it is skipped with a warning).
- An HTML part whose charset is declared only by `<meta charset>`, and an RFC 2047 word broken by
  whitespace, are decoded.
- A UTF-8 byte order mark before the header no longer swallows the first field.
- Minimal mail that is not recognized as mail (a cron message with only a Subject) is read, with a
  `not_detected_as_mail` warning.
- The deferred `Close` calls pass `errcheck`, so `make lint` is clean.

## [0.5.0] - 2026-11-07

### Changed

- **The message is read by [mailparse](https://github.com/nlink-jp/mailparse)
  v0.2.0** instead of this tool's own parser on `net/mail`, `mime` and
  `mime/multipart`. The JSONL keeps its fields and their meaning; what changes
  for the same input:
  - `to`, `cc`, `bcc` hold the mailboxes of every occurrence of the field, not
    only the first.
  - Mailboxes are written in RFC 5322 form that reads back as the same one
    mailbox: a display name holding anything but atom characters and single
    spaces is a quoted string, so a name that decodes to an address form no
    longer reads as a second address. This also quotes names with a `.`
    (`"John Q. Public"`) or an ideographic space (U+3000), which were bare.
    A name with no address is the quoted name alone.
  - A display name is decoded after the address is parsed: the old parser
    decoded first and could lose part of a name (a parenthesis read as a
    comment) or take an address out of it.
  - Bodies from quoted-printable keep the message's CRLF line ends, as the
    other transfer encodings already did (they were LF).
  - Only `text/plain` and `text/html` parts without a filename are bodies;
    `text/calendar`, `text/csv` and other text parts, and text parts with a
    `name`, are now attachments.
  - `encoding` also reports a charset taken from HTML `<meta charset>` or
    guessed for undeclared Japanese.
  - An attachment's `size` counts what decodes from damaged Base64 (it was 0).
  - Outlook `.msg`, TNEF and other OLE files, input with no header field, and
    messages larger than 64 MiB are refused.

### Added

- `from_all` (every mailbox of every `From` field, when there is more than
  one), `sender`, `reply_to`, `return_path` and `references`.
- `attachments[].message`: a forwarded message (`message/rfc822`, or a message
  attached under another type and recognized by its content) parsed in the
  same shape.
- `warnings`: what the parse met and handled instead of failing, with
  mailparse's stable codes (`not_detected_as_mail` for minimal mail that is
  read although it is not recognized as mail). Omitted when empty.

### Fixed

- A header line without a colon no longer fails the whole message; the line is
  skipped with a `malformed_header_line` warning.
- A multipart whose closing boundary never comes keeps the parts after the
  first (they were dropped silently).
- An HTML part whose charset is declared only by `<meta charset>` is decoded
  (it was garbled).
- An RFC 2047 encoded-word broken by whitespace is repaired (the subject was
  left encoded).
- A multipart nested too deep is kept as an attachment with a warning (it was
  dropped silently).
- A UTF-8 byte order mark before the header is skipped with a `leading_bom`
  warning (it swallowed the first field).
- The deferred `Close` calls pass `errcheck`, so `make lint` is clean.

- **`make verify-release` now fails closed.** Its last block chained unzip, the
  packaged binary's `--version` and `spctl` with `&&` and ended the whole chain
  in `|| true`, so a zip that did not unpack or a binary that did not run exited
  0 and the upload proceeded. Each step is now judged on its own, the packaged
  binary's `--version` must contain the tag being released, and only the
  informational `spctl` line may be ignored. Matches the org template
  (CONVENTIONS.md §Code Signing → Verifying a release).
- **The Linux archives no longer carry macOS file metadata.** macOS `tar` wrote
  each bundled file's extended attributes (`com.apple.provenance`, and a Dropbox
  attribute where the tree is synced) into the `.tar.gz` twice: as AppleDouble
  `._` members, which GNU tar extracts as stray `._<name>` files beside the real
  ones, and as `LIBARCHIVE.xattr.*` / `SCHILY.xattr.*` pax headers, which it
  reports as unknown keywords. `make package` now archives with
  `COPYFILE_DISABLE=1 tar --no-xattrs`; each setting stops one of the two.
  Archives already published still carry them; the files themselves are
  unaffected.

### Documentation

- The pipe examples and comparisons named lite-llm, which is archived; they name
  its successor, llm-cli, in the form its README documents for piped data
  (`… | llm-cli -s "<instruction>"`).
- The pipe examples named llm-cli and gem-cli, which are now archived too; they
  name their successors, lagent and gem-agent, in the form those document for
  piped data (`… | lagent -p "<instruction>"`). The PST pipeline runs gem-agent
  once per email, as `gem-cli --batch` did, because one piped attachment holds
  at most 256 KiB.

### Internal

- `make verify-release` also judges each Linux archive: no AppleDouble or other
  macOS metadata members — listed with `--options 'tar:!mac-ext'`, because a
  plain macOS listing folds `._` members away — no extended attributes as pax
  headers, and exactly the canonical binary, `README.md` and `LICENSE`, compared
  in the C locale.
- The Linux-archive check in `make verify-release` reads each archive's pax
  headers with Python's `tarfile` instead of grepping the decompressed stream,
  which also matched file text that names the keywords (a bundled CHANGELOG,
  for one).

## [0.4.0] - 2026-07-12

### Added

- **`LICENSE` (MIT).** The repository was missing a license file; added MIT to
  match the org convention. `LICENSE` is now bundled in every release archive.

### Removed

- **darwin/amd64 (Intel) pre-built binary.** macOS releases now ship
  **arm64 only**, per the org-wide policy (darwin is Apple-Silicon only; no
  universal binaries). Intel Mac users can build from source.

### Changed

- **Linux release archives are now `.tar.gz`** (darwin/windows remain `.zip`),
  per `nlink-jp/.github` CONVENTIONS.md §Release Archive Standard.
- **darwin code-signature identifier** is now the canonical `eml-to-jsonl`.

No change to the binary's behaviour — a packaging / build-config release.

## [0.3.1] - 2026-05-23

### Added

- **`package` Makefile target.** Builds all 5 platforms, signs darwin
  binaries with Developer ID, zips each with README.md using
  versioned naming (`eml-to-jsonl-vX.Y.Z-<os>-<arch>.zip`), and
  notarizes the darwin zips.

### Changed

- **Darwin releases are now Developer ID signed and Apple-notarized.**
  `eml-to-jsonl-v0.3.1-darwin-{amd64,arm64}.zip` carry full Apple
  Developer ID Application signatures and notarization tickets from
  Apple. End users on macOS no longer need to bypass Gatekeeper
  with right-click → Open or `xattr -d com.apple.quarantine` on
  first launch; local users who place `eml-to-jsonl` under
  Dropbox-synced (or any other FileProvider-managed) paths are no
  longer killed by macOS's ad-hoc + provenance distrust policy.
  Pipeline: `scripts/codesign-darwin.sh` +
  `scripts/notarize-darwin.sh`, driven by `make package`. Adopts
  the org-wide convention in `nlink-jp/.github` CONVENTIONS.md
  §Code Signing.
- **Release zip filenames now embed the version**
  (`eml-to-jsonl-vX.Y.Z-<os>-<arch>.zip`), aligning with sibling
  util-series tools. v0.3.0 assets used version-less names.

No behaviour change to the binary itself — feature-wise this is
identical to v0.3.0.

## [0.3.0] - 2026-03-30

### Added

- **`received` field** — All Received headers are now included in the output as a string array, preserving mail delivery hop order.
- **PST file handling** — README documents how to use `readpst` + eml-to-jsonl for Outlook PST files.

## [0.2.0] - 2026-03-28

### Changed

- **Breaking:** renamed from `lite-eml` to `eml-to-jsonl`.
  - Repository: `github.com/nlink-jp/eml-to-jsonl`
  - Module path: `github.com/nlink-jp/eml-to-jsonl`
  - Binary name: `eml-to-jsonl`
  - Moved from lite-series to util-series.

## [0.1.1] - 2026-03-27

### Security

- Added MIME recursion depth limit (`maxMIMEDepth = 10`) to prevent stack exhaustion
  from maliciously crafted deeply-nested multipart messages.
- Added per-part memory cap (`maxPartSize = 25 MiB`) using `io.LimitReader` in the
  transfer-encoding decoder to prevent memory exhaustion from oversized body parts.


## [0.1.0] - 2026-03-27

### Added

- Initial release.
- `eml-to-jsonl`: reads EML files from stdin, file arguments, or directories and outputs structured JSONL.
- Extracts headers: From, To, Cc, Bcc, Subject, Date, Message-Id, In-Reply-To, X-Mailer.
- Handles multipart/alternative (text/plain preferred, text/html included), multipart/mixed, and nested multipart.
- Decodes all content to UTF-8; records original charset in the `encoding` field.
- Supports Content-Transfer-Encoding: base64, quoted-printable, 7bit, 8bit.
- Supports Japanese charsets: ISO-2022-JP, Shift_JIS, EUC-JP (and all IANA-registered charsets via golang.org/x/text).
- Attachment metadata (filename, MIME type, decoded size) included in output without embedding content.
- `--pretty` flag for human-readable JSON output.

### Fixed

- `<` and `>` characters in message IDs and email addresses were HTML-escaped (`\u003c`, `\u003e`) in `--pretty` mode. Both JSONL and pretty modes now use `SetEscapeHTML(false)`.


[0.1.1]: https://github.com/nlink-jp/eml-to-jsonl/releases/tag/v0.1.1
[0.1.0]: https://github.com/nlink-jp/eml-to-jsonl/releases/tag/v0.1.0
