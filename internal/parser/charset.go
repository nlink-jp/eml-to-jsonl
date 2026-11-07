package parser

import (
	"io"
	"mime"
	"strings"

	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/transform"
)

// wordDecoder decodes RFC 2047 encoded-words in headers, supporting
// any charset registered in the IANA HTML encoding index (including
// ISO-2022-JP, Shift_JIS, EUC-JP, various ISO-8859-* charsets, etc.).
var wordDecoder = &mime.WordDecoder{
	CharsetReader: func(charset string, input io.Reader) (io.Reader, error) {
		enc, err := htmlindex.Get(charset)
		if err != nil {
			// Unrecognised charset — return the raw bytes as-is.
			return input, nil
		}
		return transform.NewReader(input, enc.NewDecoder()), nil
	},
}

// decodeToUTF8 converts data encoded in charset to a UTF-8 string.
// On failure it returns the raw bytes interpreted as UTF-8 (best effort).
func decodeToUTF8(charset string, data []byte) string {
	if charset == "" {
		return string(data)
	}
	normalized := strings.ToLower(strings.TrimSpace(charset))
	if normalized == "utf-8" || normalized == "us-ascii" {
		return string(data)
	}

	enc, err := htmlindex.Get(charset)
	if err != nil {
		return string(data)
	}
	result, _, err := transform.Bytes(enc.NewDecoder(), data)
	if err != nil {
		return string(data)
	}
	return string(result)
}

// decodeMIMEHeader decodes an RFC 2047 encoded header value to a UTF-8 string.
func decodeMIMEHeader(v string) string {
	if v == "" {
		return ""
	}
	decoded, err := wordDecoder.DecodeHeader(v)
	if err != nil {
		return v
	}
	return decoded
}
