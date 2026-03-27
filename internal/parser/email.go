// Package parser implements EML file parsing.
// It extracts headers and body from RFC 2822 email messages,
// decoding all content to UTF-8 regardless of the original charset.
package parser

import (
	"io"
	"net/mail"
	"strings"
	"time"
)

// Email is the structured representation of a parsed EML message.
type Email struct {
	Source      string       `json:"source"`
	MessageID   string       `json:"message_id,omitempty"`
	InReplyTo   string       `json:"in_reply_to,omitempty"`
	From        string       `json:"from,omitempty"`
	To          []string     `json:"to,omitempty"`
	CC          []string     `json:"cc,omitempty"`
	BCC         []string     `json:"bcc,omitempty"`
	Subject     string       `json:"subject,omitempty"`
	Date        string       `json:"date,omitempty"`
	XMailer     string       `json:"x_mailer,omitempty"`
	Encoding    string       `json:"encoding,omitempty"`
	Body        []BodyPart   `json:"body"`
	Attachments []Attachment `json:"attachments"`
}

// BodyPart represents a single decoded body section.
type BodyPart struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

// Attachment holds metadata for a non-body MIME part.
type Attachment struct {
	Filename string `json:"filename"`
	MIMEType string `json:"mime_type"`
	Size     int    `json:"size"`
}

// Parse reads one RFC 2822 EML message from r and returns a structured Email.
// source is recorded as-is in the Source field (e.g. a file path or "stdin").
func Parse(r io.Reader, source string) (*Email, error) {
	msg, err := mail.ReadMessage(r)
	if err != nil {
		return nil, err
	}

	email := &Email{
		Source:      source,
		Body:        []BodyPart{},
		Attachments: []Attachment{},
	}

	if err := extractHeaders(msg, email); err != nil {
		return nil, err
	}

	result, err := parseBody(msg.Header.Get("Content-Type"), msg.Header.Get("Content-Transfer-Encoding"), msg.Body)
	if err != nil {
		return nil, err
	}
	email.Body = result.bodyParts
	email.Attachments = result.attachments
	if result.encoding != "" {
		email.Encoding = result.encoding
	}

	return email, nil
}

// extractHeaders populates the header fields of email from msg.
func extractHeaders(msg *mail.Message, email *Email) error {
	h := msg.Header

	email.MessageID = strings.TrimSpace(h.Get("Message-Id"))
	email.InReplyTo = strings.TrimSpace(h.Get("In-Reply-To"))
	email.XMailer = strings.TrimSpace(h.Get("X-Mailer"))

	email.Subject = decodeMIMEHeader(h.Get("Subject"))

	email.From = decodeAddress(h.Get("From"))
	email.To = decodeAddressList(h.Get("To"))
	email.CC = decodeAddressList(h.Get("Cc"))
	email.BCC = decodeAddressList(h.Get("Bcc"))

	if dateStr := h.Get("Date"); dateStr != "" {
		if t, err := mail.ParseDate(dateStr); err == nil {
			email.Date = t.Format(time.RFC3339)
		} else {
			email.Date = strings.TrimSpace(dateStr)
		}
	}

	return nil
}
