// Package parser turns one RFC 5322 message into the eml-to-jsonl record.
//
// Reading the message is github.com/nlink-jp/mailparse's job: header fields in
// order with duplicates kept, addresses parsed before display names are
// decoded, bodies converted to UTF-8, and every repair, guess or cut reported
// as a warning. This package only maps mailparse.Message onto the JSON shape
// eml-to-jsonl has always emitted, and adds new information in new fields.
package parser

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/nlink-jp/mailparse"
)

// ErrNotEML is returned when the input is not recognized as an RFC 5322 message.
var ErrNotEML = errors.New("not recognized as an RFC 5322 message")

// Email is the structured representation of a parsed EML message.
type Email struct {
	Source     string   `json:"source,omitempty"`
	MessageID  string   `json:"message_id,omitempty"`
	InReplyTo  string   `json:"in_reply_to,omitempty"`
	References string   `json:"references,omitempty"`
	From       string   `json:"from,omitempty"`
	FromAll    []string `json:"from_all,omitempty"`
	Sender     []string `json:"sender,omitempty"`
	ReplyTo    []string `json:"reply_to,omitempty"`
	ReturnPath []string `json:"return_path,omitempty"`
	To         []string `json:"to,omitempty"`
	CC         []string `json:"cc,omitempty"`
	BCC        []string `json:"bcc,omitempty"`
	Subject    string   `json:"subject,omitempty"`
	Date       string   `json:"date,omitempty"`
	XMailer    string   `json:"x_mailer,omitempty"`
	Received   []string `json:"received,omitempty"`
	Encoding   string   `json:"encoding,omitempty"`

	Body        []BodyPart   `json:"body"`
	Attachments []Attachment `json:"attachments"`
	Warnings    []Warning    `json:"warnings,omitempty"`
}

// BodyPart represents a single decoded body section.
type BodyPart struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

// Attachment holds metadata for a non-body MIME part. Message is the
// attachment parsed as a message when it is one (message/rfc822, or a message
// recognized by its content).
type Attachment struct {
	Filename string `json:"filename"`
	MIMEType string `json:"mime_type"`
	Size     int    `json:"size"`
	Message  *Email `json:"message,omitempty"`
}

// Warning is something the parse met and handled instead of failing: a
// malformed line, a repair, a guessed charset, a limit that cut something.
// Code is mailparse's stable code; Part is the MIME part path ("" for the
// message header).
type Warning struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
	Part   string `json:"part,omitempty"`
}

// Parse reads one RFC 5322 message from r and returns a structured Email.
// source is recorded as-is in the Source field (e.g. a file path or "stdin").
// Input mailparse does not recognize as mail is still read when it has a
// header block (minimal mail such as cron's "From: root"), with mailparse's
// not_detected_as_mail warning. Outlook .msg and TNEF input are refused with
// a pointer to msg-to-jsonl, and messages over mailparse.DefaultMaxSize
// (64 MiB) are refused.
func Parse(r io.Reader, source string) (*Email, error) {
	b, err := io.ReadAll(io.LimitReader(r, mailparse.DefaultMaxSize+1))
	if err != nil {
		return nil, err
	}
	switch f := mailparse.Detect(b); {
	case f == mailparse.FormatMSG || f == mailparse.FormatTNEF:
		return nil, fmt.Errorf("%w: it is an Outlook %s file (use msg-to-jsonl)", ErrNotEML, f)
	case bytes.HasPrefix(b, cfbSignature):
		return nil, fmt.Errorf("%w: it is an OLE compound file", ErrNotEML)
	}
	m, err := mailparse.ParseBytes(b, mailparse.Options{AssumeEML: true})
	switch {
	case errors.Is(err, mailparse.ErrTooLarge):
		return nil, fmt.Errorf("larger than %d MiB", mailparse.DefaultMaxSize>>20)
	case errors.Is(err, mailparse.ErrNotMessage):
		return nil, ErrNotEML
	case err != nil:
		return nil, err
	}
	email := fromMessage(m)
	email.Source = source
	return email, nil
}

// cfbSignature starts every OLE compound file (.msg, legacy Office documents).
var cfbSignature = []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}

// fromMessage maps one parsed message (top level or nested) onto Email.
func fromMessage(m *mailparse.Message) *Email {
	e := &Email{
		MessageID:   first(m.Header, "Message-ID"),
		InReplyTo:   first(m.Header, "In-Reply-To"),
		References:  first(m.Header, "References"),
		XMailer:     first(m.Header, "X-Mailer"),
		Subject:     m.Subject,
		Sender:      addresses(m.Sender),
		ReplyTo:     addresses(m.ReplyTo),
		ReturnPath:  addresses(m.ReturnPath),
		To:          addresses(m.To),
		CC:          addresses(m.Cc),
		BCC:         addresses(m.Bcc),
		Body:        []BodyPart{},
		Attachments: []Attachment{},
	}

	if len(m.From) > 0 {
		e.From = strings.Join(addresses(m.From[:1]), ", ")
		if all := addresses(m.From); len(all) > 1 {
			e.FromAll = all
		}
	}

	for _, f := range m.Header.All("Received") {
		e.Received = append(e.Received, f.Collapsed())
	}

	switch {
	case !m.Date.IsZero():
		e.Date = m.Date.Format(time.RFC3339)
	case m.DateRaw != "":
		e.Date = m.DateRaw
	}

	for _, b := range m.Bodies {
		e.Body = append(e.Body, BodyPart{Type: b.MediaType, Content: b.Content})
		if e.Encoding == "" && !isASCIICompatible(b.Charset) {
			e.Encoding = strings.ToUpper(b.Charset)
		}
	}

	for _, a := range m.Attachments {
		att := Attachment{Filename: a.Filename, MIMEType: a.MediaType, Size: a.Size}
		if att.Filename == "" {
			att.Filename = "attachment"
		}
		if a.Message != nil {
			att.Message = fromMessage(a.Message)
		}
		e.Attachments = append(e.Attachments, att)
	}

	for _, w := range m.Warnings {
		e.Warnings = append(e.Warnings, Warning{Code: w.Code, Detail: w.Detail, Part: w.Part})
	}
	return e
}

// addresses flattens every occurrence of an address field into RFC 5322
// mailbox strings (mailparse.Mailbox.String: a display name that could read as
// an address is quoted, so it stays a name). A field
// that parsed to no mailbox contributes its display text when it did not
// parse (as before), "<>" when it is a null Return-Path, and nothing when it
// is an empty group such as "undisclosed-recipients:;".
func addresses(lists []mailparse.AddressList) []string {
	var out []string
	for _, l := range lists {
		if len(l.Mailboxes) == 0 {
			raw := strings.TrimSpace(l.Raw)
			switch {
			case raw == "<>":
				out = append(out, raw)
			case !l.Parsed && raw != "":
				out = append(out, strings.Join(strings.Fields(raw), " "))
			}
			continue
		}
		for _, mb := range l.Mailboxes {
			out = append(out, mb.String())
		}
	}
	return out
}

// first returns the first field named name with its whitespace collapsed, as
// net/textproto (which eml-to-jsonl used before) gave it. These fields carry
// no encoded-words.
func first(h mailparse.Header, name string) string {
	if f, ok := h.First(name); ok {
		return f.Collapsed()
	}
	return ""
}

func isASCIICompatible(charset string) bool {
	switch strings.ToLower(charset) {
	case "", "utf-8", "us-ascii":
		return true
	}
	return false
}
