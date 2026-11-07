# Defect fixtures

Byte-exact copies of `testdata/eml/` fixtures from
[mailparse](https://github.com/nlink-jp/mailparse) v0.1.0 — fictional messages (RFC 2606
domains, RFC 5737 addresses) on which mailparse's Phase 0 record measured this tool's old
parser. `internal/parser/defects_test.go` holds the behaviour each one now has. They hold raw
ISO-2022-JP / Shift_JIS bytes and CRLF line ends on purpose; do not re-save them.
