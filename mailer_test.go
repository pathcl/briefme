package main

import (
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"os"
	"strings"
	"testing"
)

// capturedMessage holds the raw bytes of what was "sent".
var capturedMessage []byte

// testSender replaces the real gomail send in tests.
type testSender struct {
	sent []byte
}

func (ts *testSender) send(cfg SMTPConfig, to, subject string, epubPath string) error {
	raw, err := buildRawMessage(cfg.From, to, subject, epubPath)
	if err != nil {
		return err
	}
	ts.sent = raw
	return nil
}

func TestSendEPUB_AttachmentPresent(t *testing.T) {
	epubPath := writeTempEPUB(t)

	cfg := SMTPConfig{
		Host: "smtp.example.com", Port: 587,
		Username: "user", Password: "pass", From: "user@example.com",
	}

	ts := &testSender{}
	if err := ts.send(cfg, "kobo@example.com", "Briefme", epubPath); err != nil {
		t.Fatalf("send error: %v", err)
	}

	msg, err := mail.ReadMessage(strings.NewReader(string(ts.sent)))
	if err != nil {
		t.Fatalf("parse message: %v", err)
	}

	ct := msg.Header.Get("Content-Type")
	mediaType, params, err := mime.ParseMediaType(ct)
	if err != nil {
		t.Fatalf("parse content-type: %v", err)
	}
	if !strings.HasPrefix(mediaType, "multipart/") {
		t.Fatalf("expected multipart, got %s", mediaType)
	}

	mr := multipart.NewReader(msg.Body, params["boundary"])
	foundEPUB := false
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("read part: %v", err)
		}
		cd := p.Header.Get("Content-Disposition")
		if strings.Contains(cd, ".epub") {
			foundEPUB = true
		}
		p.Close()
	}
	if !foundEPUB {
		t.Error("no .epub attachment found in message")
	}
}

func writeTempEPUB(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp("", "briefme-test-*.epub")
	if err != nil {
		t.Fatal(err)
	}
	// Write minimal content (doesn't need to be a real EPUB for attachment test)
	f.WriteString("fake epub content")
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })
	return f.Name()
}
