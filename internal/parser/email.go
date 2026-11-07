// Package parser maps a message read by github.com/nlink-jp/mailparse onto
// the JSONL record eml-to-jsonl and msg-to-jsonl print.
//
// This file is identical in eml-to-jsonl and msg-to-jsonl: the two tools
// print the same schema and differ only in the input format they accept.
package parser

import (
	"strings"
	"time"

	"github.com/nlink-jp/mailparse"
)

// Email is one JSONL record. A message attached to another appears, in the
// same shape, as its attachment's Message.
type Email struct {
	Source      string              `json:"source,omitempty"` // file path or "stdin"; omitted for an attached message
	MessageID   string              `json:"message_id,omitempty"`
	InReplyTo   string              `json:"in_reply_to,omitempty"`
	References  string              `json:"references,omitempty"`
	From        string              `json:"from,omitempty"` // every From mailbox, RFC 5322 form, joined by ", "
	Sender      string              `json:"sender,omitempty"`
	ReplyTo     []string            `json:"reply_to,omitempty"`
	ReturnPath  string              `json:"return_path,omitempty"` // "<>" for a null reverse path (a bounce)
	To          []string            `json:"to,omitempty"`
	CC          []string            `json:"cc,omitempty"`
	BCC         []string            `json:"bcc,omitempty"`
	Subject     string              `json:"subject,omitempty"`
	Date        string              `json:"date,omitempty"`
	XMailer     string              `json:"x_mailer,omitempty"`
	Received    []string            `json:"received,omitempty"`
	Encoding    string              `json:"encoding,omitempty"`
	Body        []BodyPart          `json:"body"`
	Attachments []Attachment        `json:"attachments"`
	Warnings    []mailparse.Warning `json:"warnings"` // what the parser met and handled in this message
}

// BodyPart represents a single decoded body section.
type BodyPart struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

// Attachment holds metadata for an attached file.
type Attachment struct {
	Filename string `json:"filename"`
	MIMEType string `json:"mime_type"`
	Size     int    `json:"size"`
	Message  *Email `json:"message,omitempty"` // the attached message, when the attachment is one
}

// FromMessage maps a parsed message onto an Email.
func FromMessage(m *mailparse.Message, source string) *Email {
	e := &Email{
		Source:      source,
		MessageID:   m.MessageID,
		InReplyTo:   m.InReplyTo,
		References:  m.References,
		From:        joinMailboxes(m.From),
		Sender:      joinMailboxes(m.Sender),
		ReplyTo:     mailboxStrings(m.ReplyTo),
		ReturnPath:  returnPath(m.ReturnPath),
		To:          mailboxStrings(m.To),
		CC:          mailboxStrings(m.Cc),
		BCC:         mailboxStrings(m.Bcc),
		Subject:     m.Subject,
		Date:        formatDate(m),
		Received:    collapsedAll(m.Header, "Received"),
		Body:        []BodyPart{},
		Attachments: []Attachment{},
		Warnings:    m.Warnings,
	}
	if e.Warnings == nil {
		e.Warnings = []mailparse.Warning{}
	}
	if f, ok := m.Header.First("X-Mailer"); ok {
		e.XMailer = f.Collapsed()
	}
	bs := bodies(m)
	for _, b := range bs {
		e.Body = append(e.Body, BodyPart{Type: b.MediaType, Content: b.Content})
	}
	e.Encoding = encoding(bs)
	for _, a := range m.Attachments {
		att := Attachment{Filename: a.Filename, MIMEType: a.MediaType, Size: a.Size}
		if a.Message != nil {
			att.Message = FromMessage(a.Message, "")
		}
		e.Attachments = append(e.Attachments, att)
	}
	return e
}

// bodies returns the bodies to print, leaving out the bodies recovered from
// compressed RTF when the message has another body: in a .msg the RTF is
// normally a copy of PidTagBody or PidTagHtml.
func bodies(m *mailparse.Message) []mailparse.Body {
	var other, rtf []mailparse.Body
	for _, b := range m.Bodies {
		if b.Origin == "rtf" {
			rtf = append(rtf, b)
		} else {
			other = append(other, b)
		}
	}
	if len(other) > 0 {
		return other
	}
	return rtf
}

// encoding returns the first charset other than UTF-8 and US-ASCII, upper-cased.
// A .msg Unicode property (UTF-16LE) is the container's storage, not a charset
// the sender chose, and is skipped as before.
func encoding(bs []mailparse.Body) string {
	for _, b := range bs {
		if b.CharsetSource == "unicode" {
			continue
		}
		if b.Charset != "" && !strings.EqualFold(b.Charset, "utf-8") && !strings.EqualFold(b.Charset, "us-ascii") {
			return strings.ToUpper(b.Charset)
		}
	}
	return ""
}

func formatDate(m *mailparse.Message) string {
	if !m.Date.IsZero() {
		return m.Date.Format(time.RFC3339)
	}
	return strings.TrimSpace(m.DateRaw)
}

func mailboxStrings(lists []mailparse.AddressList) []string {
	var out []string
	for _, l := range lists {
		for _, mb := range l.Mailboxes {
			out = append(out, mb.String())
		}
	}
	return out
}

func joinMailboxes(lists []mailparse.AddressList) string {
	return strings.Join(mailboxStrings(lists), ", ")
}

// returnPath writes the Return-Path mailboxes; a null reverse path is "<>".
func returnPath(lists []mailparse.AddressList) string {
	if s := joinMailboxes(lists); s != "" {
		return s
	}
	for _, l := range lists {
		if l.Parsed {
			return "<>"
		}
	}
	return ""
}

func collapsedAll(h mailparse.Header, name string) []string {
	var out []string
	for _, f := range h.All(name) {
		out = append(out, f.Collapsed())
	}
	return out
}
