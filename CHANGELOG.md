# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [0.1.0] - 2026-03-27

### Added

- Initial release.
- `lite-eml`: reads EML files from stdin, file arguments, or directories and outputs structured JSONL.
- Extracts headers: From, To, Cc, Bcc, Subject, Date, Message-Id, In-Reply-To, X-Mailer.
- Handles multipart/alternative (text/plain preferred, text/html included), multipart/mixed, and nested multipart.
- Decodes all content to UTF-8; records original charset in the `encoding` field.
- Supports Content-Transfer-Encoding: base64, quoted-printable, 7bit, 8bit.
- Supports Japanese charsets: ISO-2022-JP, Shift_JIS, EUC-JP (and all IANA-registered charsets via golang.org/x/text).
- Attachment metadata (filename, MIME type, decoded size) included in output without embedding content.
- `--pretty` flag for human-readable JSON output.


[0.1.0]: https://github.com/nlink-jp/lite-eml/releases/tag/v0.1.0
