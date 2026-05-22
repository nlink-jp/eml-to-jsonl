# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).


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
