package main

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/gomail.v2"
)

func SendEPUB(cfg SMTPConfig, to string, epubPath string) error {
	subject := fmt.Sprintf("Your Briefme – %s", time.Now().Format("2006-01-02"))
	m := gomail.NewMessage()
	m.SetHeader("From", cfg.From)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", "Your daily briefing is attached.")
	m.Attach(epubPath)

	d := gomail.NewDialer(cfg.Host, cfg.Port, cfg.Username, cfg.Password)
	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("send email: %w", err)
	}
	return nil
}

// buildRawMessage constructs a raw MIME message with the EPUB attached.
// Used by tests and by SendEPUB internally for verification.
func buildRawMessage(from, to, subject, epubPath string) ([]byte, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	header := make(textproto.MIMEHeader)
	header.Set("From", from)
	header.Set("To", to)
	header.Set("Subject", subject)
	header.Set("MIME-Version", "1.0")
	header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", w.Boundary()))
	for k, vs := range header {
		for _, v := range vs {
			buf.WriteString(k + ": " + v + "\r\n")
		}
	}
	buf.WriteString("\r\n")

	// text part
	th := make(textproto.MIMEHeader)
	th.Set("Content-Type", "text/plain; charset=utf-8")
	tw, err := w.CreatePart(th)
	if err != nil {
		return nil, err
	}
	tw.Write([]byte("Your daily briefing is attached."))

	// attachment
	f, err := os.Open(epubPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	ah := make(textproto.MIMEHeader)
	ah.Set("Content-Type", "application/epub+zip")
	ah.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filepath.Base(epubPath)))
	ah.Set("Content-Transfer-Encoding", "base64")
	aw, err := w.CreatePart(ah)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(aw, f); err != nil {
		return nil, err
	}

	w.Close()
	return buf.Bytes(), nil
}
