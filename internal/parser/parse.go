package parser

import (
	"errors"
	"fmt"
	"io"

	"github.com/nlink-jp/mailparse"
)

// ErrNotEML is returned for an Outlook .msg or a winmail.dat (TNEF), which
// msg-to-jsonl reads.
var ErrNotEML = errors.New("not an .eml message: this is an Outlook .msg or winmail.dat file; use msg-to-jsonl")

// Parse reads one RFC 5322 message from r and returns a structured Email.
// source is recorded as-is in the Source field (e.g. a file path or "stdin").
func Parse(r io.Reader, source string) (*Email, error) {
	// One byte past the limit, so mailparse reports ErrTooLarge instead of
	// parsing a cut message.
	data, err := io.ReadAll(io.LimitReader(r, mailparse.DefaultMaxSize+1))
	if err != nil {
		return nil, fmt.Errorf("reading: %w", err)
	}
	switch mailparse.Detect(data) {
	case mailparse.FormatMSG, mailparse.FormatTNEF:
		return nil, ErrNotEML
	}
	// The caller says the input is mail, so minimal mail that Detect does
	// not recognize is read anyway (with a not_detected_as_mail warning).
	m, err := mailparse.ParseBytes(data, mailparse.Options{AssumeEML: true})
	if err != nil {
		return nil, err
	}
	return FromMessage(m, source), nil
}
