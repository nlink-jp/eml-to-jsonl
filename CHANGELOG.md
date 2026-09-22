# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).


## [Unreleased]

### Fixed

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

### Internal

- `make verify-release` also judges each Linux archive: no AppleDouble or other
  macOS metadata members — listed with `--options 'tar:!mac-ext'`, because a
  plain macOS listing folds `._` members away — no extended attributes as pax
  headers, and exactly the canonical binary, `README.md` and `LICENSE`, compared
  in the C locale.

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
