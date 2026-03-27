package parser

import (
	"net/mail"
	"strings"
)

// decodeAddress decodes a single RFC 2822 address field (e.g. From) to a
// UTF-8 string of the form "Display Name <addr@example.com>".
func decodeAddress(raw string) string {
	if raw == "" {
		return ""
	}
	// Decode any RFC 2047 encoded words in the display name first.
	decoded := decodeMIMEHeader(raw)
	addr, err := mail.ParseAddress(decoded)
	if err != nil {
		return strings.TrimSpace(decoded)
	}
	if addr.Name != "" {
		return addr.Name + " <" + addr.Address + ">"
	}
	return addr.Address
}

// decodeAddressList decodes a comma-separated RFC 2822 address list field
// (e.g. To, Cc, Bcc) to a slice of UTF-8 formatted address strings.
func decodeAddressList(raw string) []string {
	if raw == "" {
		return nil
	}
	decoded := decodeMIMEHeader(raw)
	addrs, err := mail.ParseAddressList(decoded)
	if err != nil {
		// Fall back to the raw decoded string as a single entry.
		s := strings.TrimSpace(decoded)
		if s == "" {
			return nil
		}
		return []string{s}
	}
	out := make([]string, 0, len(addrs))
	for _, a := range addrs {
		if a.Name != "" {
			out = append(out, a.Name+" <"+a.Address+">")
		} else {
			out = append(out, a.Address)
		}
	}
	return out
}
