package parser

import (
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/textproto"
	"path/filepath"
	"strings"
)

const (
	// maxMIMEDepth limits recursive MIME nesting to prevent stack exhaustion from
	// maliciously crafted deeply-nested multipart messages.
	maxMIMEDepth = 10

	// maxPartSize is the maximum number of bytes read from a single MIME part
	// body after transfer-decoding. Prevents memory exhaustion from huge parts.
	maxPartSize = 25 * 1024 * 1024 // 25 MiB
)

// parseResult holds the output of recursive MIME part processing.
type parseResult struct {
	bodyParts   []BodyPart
	attachments []Attachment
	encoding    string // first non-UTF-8 charset encountered
}

// parseBody parses the email body, handling simple messages and multipart MIME.
// contentTypeHeader is the value of the Content-Type header for this part.
// cte is the Content-Transfer-Encoding header value for the top-level message.
func parseBody(contentTypeHeader, cte string, body io.Reader) (*parseResult, error) {
	return parseBodyDepth(contentTypeHeader, cte, body, 0)
}

func parseBodyDepth(contentTypeHeader, cte string, body io.Reader, depth int) (*parseResult, error) {
	if contentTypeHeader == "" {
		contentTypeHeader = "text/plain"
	}
	mediaType, params, err := mime.ParseMediaType(contentTypeHeader)
	if err != nil {
		// Unparseable Content-Type — read as plain text.
		data, _ := io.ReadAll(io.LimitReader(body, maxPartSize))
		return &parseResult{
			bodyParts: []BodyPart{{Type: "text/plain", Content: string(data)}},
		}, nil
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		if depth >= maxMIMEDepth {
			return &parseResult{bodyParts: []BodyPart{}, attachments: []Attachment{}}, nil
		}
		return parseMultipart(mediaType, params["boundary"], body, depth)
	}

	return parseSinglePart(mediaType, params["charset"], cte, body)
}

// parseMultipart recursively processes a multipart/* MIME entity.
func parseMultipart(mediaType, boundary string, body io.Reader, depth int) (*parseResult, error) {
	result := &parseResult{
		bodyParts:   []BodyPart{},
		attachments: []Attachment{},
	}

	mr := multipart.NewReader(body, boundary)
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			break // tolerate malformed parts
		}

		sub, err := processRawPart(part.Header, part, depth+1)
		part.Close()
		if err != nil {
			continue
		}

		result.bodyParts = append(result.bodyParts, sub.bodyParts...)
		result.attachments = append(result.attachments, sub.attachments...)
		if result.encoding == "" && sub.encoding != "" {
			result.encoding = sub.encoding
		}
	}

	// For multipart/alternative, keep text/plain first then text/html
	// (already in natural order from the part loop above).

	return result, nil
}

// processRawPart decides whether a raw MIME part is a body section or an attachment.
func processRawPart(h textproto.MIMEHeader, body io.Reader, depth int) (*parseResult, error) {
	ct := h.Get("Content-Type")
	if ct == "" {
		ct = "text/plain"
	}

	mediaType, params, err := mime.ParseMediaType(ct)
	if err != nil {
		mediaType = "application/octet-stream"
		params = map[string]string{}
	}

	cte := h.Get("Content-Transfer-Encoding")
	cd := h.Get("Content-Disposition")

	// Nested multipart: recurse if within depth limit.
	if strings.HasPrefix(mediaType, "multipart/") {
		if depth >= maxMIMEDepth {
			return &parseResult{bodyParts: []BodyPart{}, attachments: []Attachment{}}, nil
		}
		return parseMultipart(mediaType, params["boundary"], body, depth)
	}

	// Determine if this part is an attachment or inline body.
	if isAttachment(mediaType, cd, params) {
		return processAttachment(mediaType, cd, params, cte, body)
	}

	return parseSinglePart(mediaType, params["charset"], cte, body)
}

// isAttachment returns true when the part should be treated as an attachment
// rather than an inline body section.
func isAttachment(mediaType, contentDisposition string, params map[string]string) bool {
	cdLower := strings.ToLower(contentDisposition)
	if strings.HasPrefix(cdLower, "attachment") {
		return true
	}
	// Inline with an explicit filename → treat as attachment.
	if strings.HasPrefix(cdLower, "inline") {
		if fname := attachmentFilename(contentDisposition, params); fname != "" {
			return true
		}
	}
	// Non-text content types without Content-Disposition are attachments.
	if !strings.HasPrefix(mediaType, "text/") {
		return true
	}
	return false
}

// processAttachment reads a MIME part that is an attachment and returns
// an Attachment record (no body content is kept).
func processAttachment(mediaType, cd string, params map[string]string, cte string, body io.Reader) (*parseResult, error) {
	decoded, err := decodeTransfer(cte, body)
	if err != nil {
		decoded = []byte{}
	}

	filename := attachmentFilename(cd, params)

	return &parseResult{
		bodyParts: []BodyPart{},
		attachments: []Attachment{
			{
				Filename: filename,
				MIMEType: mediaType,
				Size:     len(decoded),
			},
		},
	}, nil
}

// parseSinglePart decodes a leaf MIME part and returns it as a body section.
func parseSinglePart(mediaType, charset, cte string, body io.Reader) (*parseResult, error) {
	decoded, err := decodeTransfer(cte, body)
	if err != nil {
		return &parseResult{bodyParts: []BodyPart{}}, nil
	}

	content := decodeToUTF8(charset, decoded)

	var originalEncoding string
	if charset != "" && !strings.EqualFold(charset, "utf-8") && !strings.EqualFold(charset, "us-ascii") {
		originalEncoding = strings.ToUpper(charset)
	}

	return &parseResult{
		bodyParts:   []BodyPart{{Type: mediaType, Content: content}},
		attachments: []Attachment{},
		encoding:    originalEncoding,
	}, nil
}

// decodeTransfer applies the Content-Transfer-Encoding to produce raw bytes.
// The decoded output is capped at maxPartSize to prevent memory exhaustion.
func decodeTransfer(cte string, r io.Reader) ([]byte, error) {
	limited := io.LimitReader(r, maxPartSize)
	switch strings.ToLower(strings.TrimSpace(cte)) {
	case "base64":
		return io.ReadAll(base64.NewDecoder(base64.StdEncoding, newBase64Cleaner(limited)))
	case "quoted-printable":
		return io.ReadAll(quotedprintable.NewReader(limited))
	default:
		// 7bit, 8bit, binary, or empty — read as-is.
		return io.ReadAll(limited)
	}
}

// base64Cleaner strips whitespace (newlines) from a base64 stream, since
// standard.Encoding does not tolerate embedded newlines.
type base64Cleaner struct{ r io.Reader }

func newBase64Cleaner(r io.Reader) io.Reader { return &base64Cleaner{r: r} }

func (c *base64Cleaner) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	j := 0
	for i := 0; i < n; i++ {
		if p[i] != '\n' && p[i] != '\r' && p[i] != ' ' && p[i] != '\t' {
			p[j] = p[i]
			j++
		}
	}
	return j, err
}

// attachmentFilename extracts the filename from Content-Disposition or Content-Type params.
func attachmentFilename(cd string, params map[string]string) string {
	// Try Content-Disposition filename parameter.
	if cd != "" {
		_, cdParams, err := mime.ParseMediaType(cd)
		if err == nil {
			if name := cdParams["filename"]; name != "" {
				return decodeMIMEHeader(name)
			}
		}
	}
	// Fall back to Content-Type name parameter.
	if name := params["name"]; name != "" {
		return decodeMIMEHeader(name)
	}
	// Last resort: use the file extension from the Content-Disposition raw value.
	if idx := strings.Index(strings.ToLower(cd), "filename="); idx >= 0 {
		raw := cd[idx+9:]
		raw = strings.Trim(raw, `"' `)
		if semi := strings.Index(raw, ";"); semi >= 0 {
			raw = raw[:semi]
		}
		return decodeMIMEHeader(strings.TrimSpace(raw))
	}
	return filepath.Base("attachment")
}
