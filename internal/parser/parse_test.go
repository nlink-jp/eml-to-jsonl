package parser_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/nlink-jp/eml-to-jsonl/internal/parser"
	"github.com/nlink-jp/mailparse"
	"github.com/nlink-jp/mailparse/msgtest"
)

// An Outlook .msg is refused with a pointer to msg-to-jsonl.
func TestParse_RefusesMSG(t *testing.T) {
	b := msgtest.Message([]msgtest.Prop{msgtest.P(msgtest.PidTagSubject, "s")})
	if _, err := parser.Parse(bytes.NewReader(b), "x.eml"); !errors.Is(err, parser.ErrNotEML) {
		t.Fatalf("err = %v", err)
	}
}

// A winmail.dat (TNEF) is refused the same way.
func TestParse_RefusesTNEF(t *testing.T) {
	b := msgtest.TNEF(msgtest.TNEFAttr(msgtest.LevelMessage, msgtest.AttSubject, []byte("s\x00")))
	if _, err := parser.Parse(bytes.NewReader(b), "winmail.eml"); !errors.Is(err, parser.ErrNotEML) {
		t.Fatalf("err = %v", err)
	}
}

// Minimal mail that format detection does not recognize is read, with a warning.
func TestParse_MinimalMail(t *testing.T) {
	email, err := parser.Parse(strings.NewReader("Subject: cron\r\n\r\ndone\r\n"), "stdin")
	if err != nil {
		t.Fatal(err)
	}
	if email.Subject != "cron" || len(email.Warnings) == 0 || email.Warnings[0].Code != mailparse.WarnNotDetected {
		t.Fatalf("%+v", email)
	}
}

func TestParse_TooLarge(t *testing.T) {
	big := bytes.Repeat([]byte("a"), mailparse.DefaultMaxSize+1)
	if _, err := parser.Parse(bytes.NewReader(big), "big.eml"); !errors.Is(err, mailparse.ErrTooLarge) {
		t.Fatalf("err = %v", err)
	}
}
