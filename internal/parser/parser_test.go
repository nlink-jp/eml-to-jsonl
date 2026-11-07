package parser_test

import (
	"os"
	"strings"
	"testing"

	"github.com/nlink-jp/eml-to-jsonl/internal/parser"
)

func TestParse_SimpleText(t *testing.T) {
	f, err := os.Open("../../testdata/simple.eml")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	email, err := parser.Parse(f, "simple.eml")
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	if email.From != "Alice <alice@example.com>" {
		t.Errorf("From = %q, want %q", email.From, "Alice <alice@example.com>")
	}
	if len(email.To) != 1 || email.To[0] != "Bob <bob@example.com>" {
		t.Errorf("To = %v, want [Bob <bob@example.com>]", email.To)
	}
	if email.Subject != "Hello World" {
		t.Errorf("Subject = %q, want %q", email.Subject, "Hello World")
	}
	if email.MessageID != "<simple001@example.com>" {
		t.Errorf("MessageID = %q", email.MessageID)
	}
	if email.XMailer != "TestMailer 1.0" {
		t.Errorf("XMailer = %q", email.XMailer)
	}
	if len(email.Body) != 1 {
		t.Fatalf("len(Body) = %d, want 1", len(email.Body))
	}
	if email.Body[0].Type != "text/plain" {
		t.Errorf("Body[0].Type = %q", email.Body[0].Type)
	}
	if !strings.Contains(email.Body[0].Content, "simple plain text") {
		t.Errorf("Body[0].Content does not contain expected text: %q", email.Body[0].Content)
	}
	if len(email.Attachments) != 0 {
		t.Errorf("expected no attachments, got %d", len(email.Attachments))
	}
}

func TestParse_MultipartAlternative(t *testing.T) {
	f, err := os.Open("../../testdata/multipart.eml")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	email, err := parser.Parse(f, "multipart.eml")
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	if len(email.To) != 2 {
		t.Errorf("To = %v, want 2 recipients", email.To)
	}
	if len(email.CC) != 1 {
		t.Errorf("CC = %v, want 1 recipient", email.CC)
	}
	if email.InReplyTo != "<simple001@example.com>" {
		t.Errorf("InReplyTo = %q", email.InReplyTo)
	}

	if len(email.Body) < 2 {
		t.Fatalf("len(Body) = %d, want >= 2", len(email.Body))
	}
	// text/plain must come first
	if email.Body[0].Type != "text/plain" {
		t.Errorf("Body[0].Type = %q, want text/plain", email.Body[0].Type)
	}
	if email.Body[1].Type != "text/html" {
		t.Errorf("Body[1].Type = %q, want text/html", email.Body[1].Type)
	}
}

func TestParse_Attachments(t *testing.T) {
	f, err := os.Open("../../testdata/attachment.eml")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	email, err := parser.Parse(f, "attachment.eml")
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	if len(email.Body) != 1 {
		t.Errorf("len(Body) = %d, want 1", len(email.Body))
	}
	if len(email.Attachments) != 2 {
		t.Fatalf("len(Attachments) = %d, want 2", len(email.Attachments))
	}

	pdf := email.Attachments[0]
	if pdf.Filename != "report.pdf" {
		t.Errorf("Attachments[0].Filename = %q, want report.pdf", pdf.Filename)
	}
	if pdf.MIMEType != "application/pdf" {
		t.Errorf("Attachments[0].MIMEType = %q", pdf.MIMEType)
	}
	if pdf.Size == 0 {
		t.Errorf("Attachments[0].Size = 0, want > 0")
	}

	png := email.Attachments[1]
	if png.Filename != "logo.png" {
		t.Errorf("Attachments[1].Filename = %q, want logo.png", png.Filename)
	}
}

func TestParse_JapaneseISO2022JP(t *testing.T) {
	f, err := os.Open("../../testdata/japanese.eml")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	email, err := parser.Parse(f, "japanese.eml")
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	// Encoding field should record the original charset.
	if !strings.EqualFold(email.Encoding, "ISO-2022-JP") {
		t.Errorf("Encoding = %q, want ISO-2022-JP", email.Encoding)
	}
	// Subject should be decoded to UTF-8 (contains katakana テスト).
	if !strings.Contains(email.Subject, "テスト") {
		t.Errorf("Subject = %q, expected decoded Japanese text", email.Subject)
	}
	// From should contain decoded name.
	if !strings.Contains(email.From, "山田") && !strings.Contains(email.From, "yamada") {
		t.Errorf("From = %q, expected decoded Japanese name or address", email.From)
	}
	// Body should be decoded to UTF-8 Japanese.
	if len(email.Body) == 0 {
		t.Fatal("expected at least one body part")
	}
	body := email.Body[0].Content
	if !strings.Contains(body, "テストメール") {
		t.Errorf("Body not decoded to UTF-8 Japanese: %q", body)
	}
}

func TestParse_Base64Body(t *testing.T) {
	f, err := os.Open("../../testdata/base64_body.eml")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	email, err := parser.Parse(f, "base64_body.eml")
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	if len(email.Body) != 1 {
		t.Fatalf("len(Body) = %d, want 1", len(email.Body))
	}
	if email.Body[0].Type != "text/plain" {
		t.Errorf("Body[0].Type = %q", email.Body[0].Type)
	}
	body := email.Body[0].Content
	if !strings.Contains(body, "base64-encoded plain text body") {
		t.Errorf("Body content not decoded correctly: %q", body)
	}
	if !strings.Contains(body, "Second line of content") {
		t.Errorf("Body missing second line: %q", body)
	}
}

func TestParse_Base64JapaneseBody(t *testing.T) {
	f, err := os.Open("../../testdata/base64_japanese.eml")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	email, err := parser.Parse(f, "base64_japanese.eml")
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	if !strings.EqualFold(email.Encoding, "ISO-2022-JP") {
		t.Errorf("Encoding = %q, want ISO-2022-JP", email.Encoding)
	}
	if len(email.Body) != 1 {
		t.Fatalf("len(Body) = %d, want 1", len(email.Body))
	}
	body := email.Body[0].Content
	// Decoded content should contain Japanese characters (UTF-8).
	if !strings.ContainsAny(body, "これはテスト") {
		t.Errorf("Body not decoded to UTF-8 Japanese: %q", body)
	}
	// XMailer should be preserved.
	if email.XMailer != "Thunderbird" {
		t.Errorf("XMailer = %q", email.XMailer)
	}
}

func TestParse_StdinAlias(t *testing.T) {
	raw := "From: a@b.com\r\nTo: c@d.com\r\nSubject: Test\r\nDate: Thu, 27 Mar 2026 10:00:00 +0000\r\n\r\nBody text.\r\n"
	email, err := parser.Parse(strings.NewReader(raw), "stdin")
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if email.Source != "stdin" {
		t.Errorf("Source = %q, want stdin", email.Source)
	}
	if email.Subject != "Test" {
		t.Errorf("Subject = %q", email.Subject)
	}
}
