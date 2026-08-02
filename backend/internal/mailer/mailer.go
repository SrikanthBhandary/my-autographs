// Package mailer sends transactional email over plain SMTP, so it works
// with whatever provider you point it at (Gmail app password, SendGrid,
// Mailgun, Postmark, AWS SES, Resend's SMTP relay, a local Mailhog/Maildev
// instance for dev, etc) — just credentials in env vars, no provider SDK.
package mailer

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

type Config struct {
	Host      string
	Port      string
	Username  string
	Password  string
	FromEmail string
	FromName  string
}

type Mailer struct {
	cfg Config
}

func New(cfg Config) *Mailer {
	return &Mailer{cfg: cfg}
}

// Configured reports whether SMTP has actually been set up. Callers should
// skip sending (and just log) when this is false — e.g. local dev without a
// mail provider configured yet.
func (m *Mailer) Configured() bool {
	return m.cfg.Host != "" && m.cfg.FromEmail != ""
}

// Send delivers an HTML email to a single recipient.
func (m *Mailer) Send(to, subject, htmlBody string) error {
	if !m.Configured() {
		return fmt.Errorf("mailer not configured (SMTP_HOST / SMTP_FROM_EMAIL missing)")
	}

	var b strings.Builder
	fmt.Fprintf(&b, "From: %s <%s>\r\n", m.cfg.FromName, m.cfg.FromEmail)
	fmt.Fprintf(&b, "To: %s\r\n", to)
	fmt.Fprintf(&b, "Subject: %s\r\n", subject)
	fmt.Fprintf(&b, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&b, "Content-Type: text/html; charset=UTF-8\r\n\r\n")
	b.WriteString(htmlBody)

	addr := fmt.Sprintf("%s:%s", m.cfg.Host, m.cfg.Port)
	var auth smtp.Auth
	if m.cfg.Username != "" {
		auth = smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	}

	if m.cfg.Port == "465" {
		// Port 465 expects TLS from the very first byte — smtp.SendMail
		// only knows how to negotiate STARTTLS, so this needs a manual dial.
		return sendImplicitTLS(addr, m.cfg.Host, auth, m.cfg.FromEmail, []string{to}, []byte(b.String()))
	}
	return smtp.SendMail(addr, auth, m.cfg.FromEmail, []string{to}, []byte(b.String()))
}

func sendImplicitTLS(addr, host string, auth smtp.Auth, from string, to []string, msg []byte) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	for _, addr := range to {
		if err := client.Rcpt(addr); err != nil {
			return err
		}
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	return w.Close()
}

// NewEntrySubmittedEmail builds the HTML body for the "someone just signed
// your book" notification.
func NewEntrySubmittedEmail(ownerName, guestName, categoryName, reviewURL string) string {
	return fmt.Sprintf(`<div style="font-family:Georgia,serif;max-width:480px;margin:0 auto;padding:32px;color:#1c1e2a;">
  <p style="font-size:12px;letter-spacing:0.08em;text-transform:uppercase;color:#2f5d50;font-weight:600;margin:0 0 8px;">New autograph</p>
  <h2 style="margin:0 0 12px;font-family:Georgia,serif;">%s just signed your book</h2>
  <p style="color:#565a6e;line-height:1.5;">Hi %s — they left an entry in <strong>%s</strong>. Approve it to add it to your keepsake book.</p>
  <a href="%s" style="display:inline-block;margin-top:16px;background:#2f5d50;color:#ffffff;padding:10px 22px;border-radius:8px;text-decoration:none;font-family:sans-serif;">Review submission</a>
</div>`, guestName, ownerName, categoryName, reviewURL)
}
