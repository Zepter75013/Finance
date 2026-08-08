package mailer

import (
	"bytes"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net/smtp"

	"finance/backend/internal/config"
)

type Mailer struct {
	host     string
	port     string
	username string
	password string
	from     string
}

func New(cfg config.Config) *Mailer {
	return &Mailer{
		host:     cfg.SMTPHost,
		port:     cfg.SMTPPort,
		username: cfg.SMTPUsername,
		password: cfg.SMTPPassword,
		from:     cfg.SMTPFrom,
	}
}

func (m *Mailer) Configured() bool {
	return m.host != "" && m.username != "" && m.password != "" && m.from != ""
}

// Send encodes the subject (RFC 2047) and body (quoted-printable) so the
// message stays plain 7-bit ASCII on the wire — some receiving mail servers
// (e.g. free.fr) reject messages that rely on the SMTPUTF8/8BITMIME
// extensions for raw UTF-8 headers or bodies.
func (m *Mailer) Send(to, subject, body string) error {
	if !m.Configured() {
		return fmt.Errorf("SMTP not configured")
	}

	addr := fmt.Sprintf("%s:%s", m.host, m.port)
	auth := smtp.PlainAuth("", m.username, m.password, m.host)

	encodedSubject := mime.BEncoding.Encode("UTF-8", subject)

	var encodedBody bytes.Buffer
	qpWriter := quotedprintable.NewWriter(&encodedBody)
	if _, err := qpWriter.Write([]byte(body)); err != nil {
		return err
	}
	if err := qpWriter.Close(); err != nil {
		return err
	}

	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\n%s\r\n",
		m.from, to, encodedSubject, encodedBody.String(),
	)

	return smtp.SendMail(addr, auth, m.from, []string{to}, []byte(msg))
}
