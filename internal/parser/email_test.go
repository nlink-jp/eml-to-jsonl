package parser

// This file is identical in eml-to-jsonl and msg-to-jsonl: it holds the
// schema both tools print.

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/nlink-jp/mailparse"
	"github.com/nlink-jp/mailparse/msgtest"
)

func fromEML(t *testing.T, raw string) *Email {
	t.Helper()
	m, err := mailparse.ParseBytes([]byte(strings.ReplaceAll(raw, "\n", "\r\n")), mailparse.Options{AssumeEML: true})
	if err != nil {
		t.Fatal(err)
	}
	return FromMessage(m, "t.eml")
}

func hasCode(ws []mailparse.Warning, code string) bool {
	for _, w := range ws {
		if w.Code == code {
			return true
		}
	}
	return false
}

func TestAddressFields(t *testing.T) {
	// The From display name is "Bank <security@bank.example>" in RFC 2047:
	// it stays a quoted name, never a second address.
	e := fromEML(t, `Return-Path: <>
From: =?utf-8?b?QmFuayA8c2VjdXJpdHlAYmFuay5leGFtcGxlPg==?= <attacker@evil.example>
From: second@example.org
Sender: list@example.net
Reply-To: a@example.com, "Doe, Jane" <jane@example.com>
To: b@example.com
Subject: s
Message-ID: <1@example.com>
References: <0@example.com>

body
`)
	if e.From != `"Bank <security@bank.example>" <attacker@evil.example>, second@example.org` {
		t.Errorf("From = %q", e.From)
	}
	if e.Sender != "list@example.net" || e.ReturnPath != "<>" || e.References != "<0@example.com>" {
		t.Errorf("%+v", e)
	}
	if len(e.ReplyTo) != 2 || e.ReplyTo[1] != `"Doe, Jane" <jane@example.com>` {
		t.Errorf("ReplyTo = %q", e.ReplyTo)
	}
	if !hasCode(e.Warnings, mailparse.WarnDuplicateField) {
		t.Errorf("warnings = %+v", e.Warnings)
	}
}

// An attached message appears as its attachment's message, in the same
// shape, without a source and with its own warnings.
func TestNestedMessage(t *testing.T) {
	e := fromEML(t, `From: a@example.com
To: b@example.com
Subject: fwd
Message-ID: <1@example.com>
MIME-Version: 1.0
Content-Type: multipart/mixed; boundary="b1"

--b1
Content-Type: text/plain

see attached
--b1
Content-Type: text/calendar; charset=utf-8; name="invite.ics"
Content-Disposition: attachment; filename="invite.ics"

BEGIN:VCALENDAR
--b1
Content-Type: message/rfc822

From: x@evil.example
From: y@evil.example
To: b@example.com
Subject: inner
Message-ID: <2@evil.example>

inner body
--b1--
`)
	if len(e.Body) != 1 || len(e.Attachments) != 2 {
		t.Fatalf("body %d attachments %+v", len(e.Body), e.Attachments)
	}
	if e.Attachments[0].Filename != "invite.ics" || e.Attachments[0].MIMEType != "text/calendar" || e.Attachments[0].Message != nil {
		t.Errorf("calendar = %+v", e.Attachments[0])
	}
	if e.Attachments[1].Filename != "" {
		t.Errorf("an attachment without a name has filename %q", e.Attachments[1].Filename)
	}
	inner := e.Attachments[1].Message
	if inner == nil || inner.Subject != "inner" || inner.Source != "" || len(inner.Body) != 1 {
		t.Fatalf("inner = %+v", inner)
	}
	if !hasCode(inner.Warnings, mailparse.WarnDuplicateField) || hasCode(e.Warnings, mailparse.WarnDuplicateField) {
		t.Errorf("outer %+v inner %+v", e.Warnings, inner.Warnings)
	}
	var buf strings.Builder
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false) // as main prints it
	if err := enc.Encode(e); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.HasPrefix(s, `{"source":"t.eml",`) || !strings.Contains(s, `"message":{"message_id":"<2@evil.example>"`) || !strings.Contains(s, `"warnings":[]`) {
		t.Errorf("json = %s", s)
	}
}

// The RTF copy of a .msg body is left out when another body exists, and is
// the body when it is the only one.
func TestMSGBodies(t *testing.T) {
	rtf := msgtest.CompressRTF([]byte(`{\rtf1\ansi\fromhtml1 {\*\htmltag <a href="https://rtf.example.com/">}x{\*\htmltag </a>}}`))
	sent := time.Date(2026, 11, 7, 1, 2, 3, 0, time.UTC)
	for _, tc := range []struct {
		name  string
		props []msgtest.Prop
		want  string
	}{
		{"plain and rtf", []msgtest.Prop{msgtest.P(msgtest.PidTagBody, msgtest.String8("plain")), msgtest.P(msgtest.PidTagRtfCompressed, rtf)}, "text/plain"},
		{"rtf only", []msgtest.Prop{msgtest.P(msgtest.PidTagRtfCompressed, rtf)}, "text/html"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			props := append([]msgtest.Prop{msgtest.P(msgtest.PidTagSubject, "s"), msgtest.P(msgtest.PidTagClientSubmitTime, sent)}, tc.props...)
			m, err := mailparse.ParseBytes(msgtest.Message(props), mailparse.Options{})
			if err != nil {
				t.Fatal(err)
			}
			e := FromMessage(m, "x.msg")
			if len(e.Body) != 1 || e.Body[0].Type != tc.want || e.Date != "2026-11-07T01:02:03Z" {
				t.Fatalf("%+v", e)
			}
		})
	}
}

// A Unicode .msg body is stored as UTF-16LE; that is no charset to report.
func TestEncodingSkipsMSGUnicode(t *testing.T) {
	m, err := mailparse.ParseBytes(msgtest.Message([]msgtest.Prop{msgtest.P(msgtest.PidTagBody, "unicode body")}), mailparse.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Bodies) != 1 || m.Bodies[0].CharsetSource != "unicode" {
		t.Fatalf("bodies = %+v", m.Bodies)
	}
	if e := FromMessage(m, "x.msg"); e.Encoding != "" {
		t.Fatalf("Encoding = %q", e.Encoding)
	}
}

func TestEncoding(t *testing.T) {
	e := fromEML(t, "From: a@example.com\nTo: b@example.com\nSubject: s\nMessage-ID: <1@example.com>\nContent-Type: text/plain; charset=iso-2022-jp\n\nhello\n")
	if e.Encoding != "ISO-2022-JP" {
		t.Fatalf("Encoding = %q", e.Encoding)
	}
}
