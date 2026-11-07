package parser_test

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/nlink-jp/eml-to-jsonl/internal/parser"
	"github.com/nlink-jp/mailparse"
	"github.com/nlink-jp/mailparse/msgtest"
)

// parseFile parses a fixture under testdata/.
func parseFile(t *testing.T, name string) *parser.Email {
	t.Helper()
	f, err := os.Open("../../testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	email, err := parser.Parse(f, name)
	if err != nil {
		t.Fatalf("Parse(%s) error: %v", name, err)
	}
	return email
}

func parseString(t *testing.T, raw string) *parser.Email {
	t.Helper()
	email, err := parser.Parse(strings.NewReader(raw), "stdin")
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	return email
}

func warningCodes(e *parser.Email) []string {
	var codes []string
	for _, w := range e.Warnings {
		codes = append(codes, w.Code)
	}
	return codes
}

func hasWarning(e *parser.Email, code string) bool {
	for _, w := range e.Warnings {
		if w.Code == code {
			return true
		}
	}
	return false
}

// The defect fixtures come from github.com/nlink-jp/mailparse testdata/eml/
// (its Phase 0 record measured the old parser of this tool on them).

func TestDefect_BrokenEncodedWordRepaired(t *testing.T) {
	e := parseFile(t, "defects/02-broken-b64-word.eml")
	if e.Subject != "重要なお知らせ" {
		t.Errorf("Subject = %q, want the repaired Japanese subject", e.Subject)
	}
	if !hasWarning(e, "repaired_encoded_word") {
		t.Errorf("warnings = %v, want repaired_encoded_word", warningCodes(e))
	}
}

func TestDefect_ForwardedRFC822Opened(t *testing.T) {
	e := parseFile(t, "defects/04-forwarded-rfc822.eml")
	if len(e.Attachments) != 1 {
		t.Fatalf("len(Attachments) = %d, want 1", len(e.Attachments))
	}
	a := e.Attachments[0]
	if a.MIMEType != "message/rfc822" || a.Filename != "original.eml" {
		t.Errorf("attachment = %q %q", a.MIMEType, a.Filename)
	}
	if a.Message == nil {
		t.Fatal("forwarded message was not opened")
	}
	if a.Message.From != "Support <support@phish.example>" {
		t.Errorf("nested From = %q", a.Message.From)
	}
	if a.Message.Source != "" {
		t.Errorf("nested Source = %q, want empty", a.Message.Source)
	}
	if len(a.Message.Attachments) != 1 || a.Message.Attachments[0].Filename != "invoice.zip" {
		t.Errorf("nested attachments = %+v, want invoice.zip", a.Message.Attachments)
	}
}

func TestDefect_ForwardedOctetStreamOpened(t *testing.T) {
	e := parseFile(t, "defects/05-forwarded-octet-stream.eml")
	if len(e.Attachments) != 1 || e.Attachments[0].Message == nil {
		t.Fatalf("message sent as application/octet-stream was not opened: %+v", e.Attachments)
	}
	if got := e.Attachments[0].MIMEType; got != "application/octet-stream" {
		t.Errorf("mime_type = %q, want the declared type kept", got)
	}
}

func TestDefect_UnclosedBoundaryKeepsParts(t *testing.T) {
	e := parseFile(t, "defects/06-unclosed-boundary.eml")
	if len(e.Body) != 2 {
		t.Fatalf("len(Body) = %d, want 2 (the second part used to be dropped)", len(e.Body))
	}
	if !strings.Contains(e.Body[1].Content, "second part") {
		t.Errorf("Body[1] = %q", e.Body[1].Content)
	}
	if !hasWarning(e, "unclosed_boundary") {
		t.Errorf("warnings = %v, want unclosed_boundary", warningCodes(e))
	}
}

func TestDefect_DuplicateFromAndSender(t *testing.T) {
	e := parseFile(t, "defects/07-duplicate-headers.eml")
	if e.From != "ceo@example.org" {
		t.Errorf("From = %q, want the first From field", e.From)
	}
	want := []string{"ceo@example.org", "CEO <ceo@example.org>", "attacker@phish.example"}
	if strings.Join(e.FromAll, "|") != strings.Join(want, "|") {
		t.Errorf("FromAll = %q, want %q", e.FromAll, want)
	}
	if len(e.Sender) != 1 || e.Sender[0] != "attacker@phish.example" {
		t.Errorf("Sender = %q", e.Sender)
	}
	if len(e.Received) != 2 {
		t.Errorf("len(Received) = %d, want 2", len(e.Received))
	}
	if !hasWarning(e, "duplicate_field") {
		t.Errorf("warnings = %v, want duplicate_field", warningCodes(e))
	}
}

func TestDefect_EncodedDisplayNameCannotForgeAddress(t *testing.T) {
	e := parseFile(t, "defects/08-rfc2047-address-spoof.eml")
	want := `"Bank Security <security@bank.example>" <attacker@phish.example>`
	if e.From != want {
		t.Errorf("From = %q, want %q", e.From, want)
	}
}

func TestDefect_MetaCharset(t *testing.T) {
	e := parseFile(t, "defects/09-meta-charset.eml")
	if len(e.Body) != 1 || !strings.Contains(e.Body[0].Content, "パスワード") {
		t.Fatalf("HTML body not decoded by its <meta charset>: %+v", e.Body)
	}
	if strings.ContainsRune(e.Body[0].Content, '�') {
		t.Error("body still holds replacement characters")
	}
	if e.Encoding != "SHIFT_JIS" {
		t.Errorf("Encoding = %q, want SHIFT_JIS", e.Encoding)
	}
}

func TestDefect_MalformedHeaderLineDoesNotFail(t *testing.T) {
	e := parseFile(t, "defects/10-malformed-header-line.eml")
	if e.Subject != "malformed" {
		t.Errorf("Subject = %q", e.Subject)
	}
	if len(e.Body) != 1 || !strings.Contains(e.Body[0].Content, "body survives") {
		t.Errorf("Body = %+v", e.Body)
	}
	if !hasWarning(e, "malformed_header_line") {
		t.Errorf("warnings = %v, want malformed_header_line", warningCodes(e))
	}
}

func TestDefect_BadBase64KeepsWhatDecodes(t *testing.T) {
	e := parseFile(t, "defects/11-bad-base64-attachment.eml")
	if len(e.Attachments) != 1 || e.Attachments[0].Size == 0 {
		t.Fatalf("attachment = %+v, want the bytes that decode (was size 0)", e.Attachments)
	}
	if !hasWarning(e, "invalid_base64") {
		t.Errorf("warnings = %v, want invalid_base64", warningCodes(e))
	}
}

func TestParse_CleanMessageHasNoWarnings(t *testing.T) {
	for _, name := range []string{"simple.eml", "multipart.eml", "attachment.eml", "japanese.eml"} {
		if e := parseFile(t, name); len(e.Warnings) != 0 {
			t.Errorf("%s: warnings = %v, want none", name, warningCodes(e))
		}
	}
}

func TestParse_DuplicateToFieldsAreAllKept(t *testing.T) {
	e := parseString(t, "From: a@example.org\r\nTo: b@example.org\r\nTo: c@example.org, d@example.org\r\nSubject: x\r\n\r\nbody\r\n")
	if strings.Join(e.To, ",") != "b@example.org,c@example.org,d@example.org" {
		t.Errorf("To = %q", e.To)
	}
}

func TestParse_ReplyToAndReturnPath(t *testing.T) {
	e := parseString(t, "Return-Path: <>\r\nFrom: a@example.org\r\nReply-To: Help Desk <help@example.net>\r\nSubject: x\r\n\r\nbody\r\n")
	if len(e.ReplyTo) != 1 || e.ReplyTo[0] != "Help Desk <help@example.net>" {
		t.Errorf("ReplyTo = %q", e.ReplyTo)
	}
	if len(e.ReturnPath) != 1 || e.ReturnPath[0] != "<>" {
		t.Errorf("ReturnPath = %q, want the null reverse path", e.ReturnPath)
	}
}

func TestParse_EmptyGroupHasNoRecipients(t *testing.T) {
	e := parseString(t, "From: a@example.org\r\nTo: undisclosed-recipients:;\r\nSubject: x\r\n\r\nbody\r\n")
	if len(e.To) != 0 {
		t.Errorf("To = %q, want none", e.To)
	}
}

func TestParse_DisplayNameQuoting(t *testing.T) {
	e := parseString(t, "From: \"Doe, John\" <john@example.org>\r\nTo: \"Q \\\"x\\\" Z\" <q@example.org>\r\nCc: John Q. Public <jqp@example.org>\r\nSubject: x\r\n\r\nbody\r\n")
	if e.From != `"Doe, John" <john@example.org>` {
		t.Errorf("From = %q", e.From)
	}
	if len(e.To) != 1 || e.To[0] != `"Q \"x\" Z" <q@example.org>` {
		t.Errorf("To = %q", e.To)
	}
	// A "." is no atom character: mailparse.Mailbox.String quotes the name
	// (net/textproto-era output left it bare).
	if len(e.CC) != 1 || e.CC[0] != `"John Q. Public" <jqp@example.org>` {
		t.Errorf("CC = %q", e.CC)
	}
}

func TestParse_PlainNamesStayUnquoted(t *testing.T) {
	e := parseString(t, "From: Taro Yamada <taro@example.org>\r\nTo: =?UTF-8?B?5bGx55SwIOWkqumDjg==?= <yamada@example.org>\r\nSubject: x\r\n\r\nbody\r\n")
	if e.From != "Taro Yamada <taro@example.org>" {
		t.Errorf("From = %q", e.From)
	}
	if len(e.To) != 1 || e.To[0] != "山田 太郎 <yamada@example.org>" {
		t.Errorf("To = %q", e.To)
	}
}

func TestParse_UnparsedAddressField(t *testing.T) {
	e := parseString(t, "From: =?UTF-8?B?5bGx55Sw?= <a@example.org>, ,,broken <<x\r\nTo: b@example.org\r\nSubject: x\r\n\r\nbody\r\n")
	if e.From != `山田 <a@example.org>, "broken <<x"` {
		t.Errorf("From = %q", e.From)
	}
	if !hasWarning(e, "invalid_address") {
		t.Errorf("warnings = %v, want invalid_address", warningCodes(e))
	}
}

func TestParse_OtherTextTypesAreAttachments(t *testing.T) {
	raw := "From: a@example.org\r\nTo: b@example.org\r\nSubject: x\r\nMIME-Version: 1.0\r\nContent-Type: multipart/mixed; boundary=b\r\n\r\n" +
		"--b\r\nContent-Type: text/plain\r\n\r\nhi\r\n" +
		"--b\r\nContent-Type: text/calendar; charset=utf-8\r\n\r\nBEGIN:VCALENDAR\r\n" +
		"--b\r\nContent-Type: text/plain; name=\"n.txt\"\r\n\r\nnamed\r\n--b--\r\n"
	e := parseString(t, raw)
	if len(e.Body) != 1 || e.Body[0].Type != "text/plain" {
		t.Errorf("Body = %+v, want only the unnamed text/plain", e.Body)
	}
	if len(e.Attachments) != 2 || e.Attachments[0].MIMEType != "text/calendar" || e.Attachments[1].Filename != "n.txt" {
		t.Errorf("Attachments = %+v", e.Attachments)
	}
}

func TestParse_FoldedFieldsJoinedWithOneSpace(t *testing.T) {
	e := parseString(t, "Received: from a.example.org (a.example.org [192.0.2.1])\r\n\tby b.example.org;\r\n\tFri, 6 Nov 2026 10:00:00 +0900\r\nFrom: a@example.org\r\nX-Mailer: Mailer\r\n 2.0\r\nSubject: x\r\n\r\nbody\r\n")
	want := "from a.example.org (a.example.org [192.0.2.1]) by b.example.org; Fri, 6 Nov 2026 10:00:00 +0900"
	if len(e.Received) != 1 || e.Received[0] != want {
		t.Errorf("Received = %q, want %q", e.Received, want)
	}
	if e.XMailer != "Mailer 2.0" {
		t.Errorf("XMailer = %q", e.XMailer)
	}
}

func TestParse_InvalidDateKeepsRaw(t *testing.T) {
	e := parseString(t, "From: a@example.org\r\nDate: sometime yesterday\r\nSubject: x\r\n\r\nbody\r\n")
	if e.Date != "sometime yesterday" {
		t.Errorf("Date = %q", e.Date)
	}
	if !hasWarning(e, "invalid_date") {
		t.Errorf("warnings = %v, want invalid_date", warningCodes(e))
	}
}

func TestParse_UnnamedAttachmentKeepsPlaceholderName(t *testing.T) {
	raw := "From: a@example.org\r\nSubject: x\r\nMIME-Version: 1.0\r\nContent-Type: multipart/mixed; boundary=b\r\n\r\n" +
		"--b\r\nContent-Type: text/plain\r\n\r\nhi\r\n" +
		"--b\r\nContent-Type: application/octet-stream\r\n\r\nxyz\r\n--b--\r\n"
	e := parseString(t, raw)
	if len(e.Attachments) != 1 || e.Attachments[0].Filename != "attachment" {
		t.Errorf("Attachments = %+v, want filename \"attachment\"", e.Attachments)
	}
}

func TestParse_MinimalMailReadWithWarning(t *testing.T) {
	for _, raw := range []string{
		"From: root\nSubject: cron\n\nbody\n",
		"To: x@example.org\nSubject: s\n\nbody\n",
		"Subject: only\n\nbody\n",
	} {
		e := parseString(t, raw)
		if len(e.Body) != 1 || e.Body[0].Content != "body\n" {
			t.Errorf("%q: Body = %+v", raw, e.Body)
		}
		if len(e.Warnings) == 0 || e.Warnings[0].Code != mailparse.WarnNotDetected {
			t.Errorf("%q: warnings = %v, want %s first", raw, warningCodes(e), mailparse.WarnNotDetected)
		}
	}
}

func TestParse_DetectedMailHasNoDetectionWarning(t *testing.T) {
	e := parseString(t, "From: a@example.org\r\nTo: c@example.org\r\nSubject: s\r\n\r\nbody\r\n")
	if hasWarning(e, mailparse.WarnNotDetected) {
		t.Errorf("warnings = %v", warningCodes(e))
	}
}

func TestParse_RefusesInputWithoutHeader(t *testing.T) {
	for _, raw := range []string{"", "hello world\n", "\n\nbody only\n", "\x00\x01\x02binary"} {
		if _, err := parser.Parse(strings.NewReader(raw), "x"); !errors.Is(err, parser.ErrNotEML) {
			t.Errorf("%q: err = %v, want ErrNotEML", raw, err)
		}
	}
}

func TestParse_LeadingBOMSkipped(t *testing.T) {
	e := parseString(t, "\xEF\xBB\xBFFrom: a@example.org\r\nTo: b@example.org\r\nSubject: bom\r\n\r\nbody\r\n")
	if e.From != "a@example.org" || e.Subject != "bom" {
		t.Errorf("From = %q, Subject = %q, want the first field kept", e.From, e.Subject)
	}
	if !hasWarning(e, mailparse.WarnLeadingBOM) {
		t.Errorf("warnings = %v, want %s", warningCodes(e), mailparse.WarnLeadingBOM)
	}
}

func TestParse_References(t *testing.T) {
	e := parseString(t, "From: a@example.org\r\nTo: b@example.org\r\nSubject: re\r\nIn-Reply-To: <2@example.org>\r\nReferences: <1@example.org>\r\n\t<2@example.org>\r\n\r\nbody\r\n")
	if e.References != "<1@example.org> <2@example.org>" {
		t.Errorf("References = %q", e.References)
	}
	if e.InReplyTo != "<2@example.org>" {
		t.Errorf("InReplyTo = %q", e.InReplyTo)
	}
}

func TestParse_RefusesRealOutlookMSG(t *testing.T) {
	msg := msgtest.Message([]msgtest.Prop{msgtest.P(msgtest.PidTagSubject, "fictional")})
	_, err := parser.Parse(bytes.NewReader(msg), "x.msg")
	if !errors.Is(err, parser.ErrNotEML) || !strings.Contains(err.Error(), "msg-to-jsonl") {
		t.Errorf("err = %v, want ErrNotEML pointing at msg-to-jsonl", err)
	}
}

func TestParse_RefusesOutlookMSG(t *testing.T) {
	// A Compound File Binary signature is enough for detection to say "msg"
	// or "unknown"; either way it must not be read as RFC 5322.
	cfb := append([]byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}, bytes.Repeat([]byte{0}, 504)...)
	if _, err := parser.Parse(bytes.NewReader(cfb), "x.msg"); !errors.Is(err, parser.ErrNotEML) {
		t.Errorf("err = %v, want ErrNotEML", err)
	}
}

func TestParse_RefusesOversizedInput(t *testing.T) {
	r := strings.NewReader("From: a@example.org\r\n\r\n" + strings.Repeat("x", mailparse.DefaultMaxSize))
	if _, err := parser.Parse(r, "big.eml"); err == nil {
		t.Error("err = nil, want a size error")
	}
}
