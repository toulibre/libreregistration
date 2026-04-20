package mail

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/smtp"

	"github.com/toulibre/libreregistration/internal/config"
	"github.com/toulibre/libreregistration/internal/i18n"
)

func SendConfirmation(cfg *config.Config, ctx context.Context, to, eventTitle, cancelURL string) {
	subject := i18n.Tf(ctx, "mail.confirmation_subject_fmt", eventTitle)
	body := i18n.Tf(ctx, "mail.confirmation_body_fmt", eventTitle, cancelURL, cfg.SMTPFrom)

	if err := send(cfg, to, subject, body); err != nil {
		log.Printf("Failed to send email to %s: %v", to, err)
	}
}

func SendOrganizerNotification(cfg *config.Config, ctx context.Context, to, attendeeName, attendeeEmail, eventTitle string) {
	subject := i18n.Tf(ctx, "mail.organizer_notification_subject_fmt", eventTitle)
	body := i18n.Tf(ctx, "mail.organizer_notification_body_fmt", eventTitle, attendeeName, attendeeEmail, cfg.SMTPFrom)

	if err := send(cfg, to, subject, body); err != nil {
		log.Printf("Failed to send organizer notification to %s: %v", to, err)
	}
}

func SendPasswordReset(cfg *config.Config, ctx context.Context, to, resetURL string) {
	subject := i18n.T(ctx, "mail.reset_subject")
	body := i18n.Tf(ctx, "mail.reset_body_fmt", resetURL, cfg.SMTPFrom)

	if err := send(cfg, to, subject, body); err != nil {
		log.Printf("Failed to send password reset email to %s: %v", to, err)
	}
}

func SendEmailVerification(cfg *config.Config, ctx context.Context, to, verifyURL string) {
	subject := i18n.T(ctx, "mail.verify_subject")
	body := i18n.Tf(ctx, "mail.verify_body_fmt", verifyURL, cfg.SMTPFrom)

	if err := send(cfg, to, subject, body); err != nil {
		log.Printf("Failed to send verification email to %s: %v", to, err)
	}
}

func send(cfg *config.Config, to, subject, body string) error {
	addr := fmt.Sprintf("%s:%s", cfg.SMTPHost, cfg.SMTPPort)
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		cfg.SMTPFrom, to, subject, body)

	if !cfg.SMTPInsecure {
		var auth smtp.Auth
		if cfg.SMTPUser != "" {
			auth = smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPHost)
		}
		return smtp.SendMail(addr, auth, cfg.SMTPFrom, []string{to}, []byte(msg))
	}

	// Insecure mode: skip TLS certificate verification
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	c, err := smtp.NewClient(conn, cfg.SMTPHost)
	if err != nil {
		conn.Close()
		return fmt.Errorf("smtp client: %w", err)
	}
	defer c.Close()

	tlsConfig := &tls.Config{ServerName: cfg.SMTPHost, InsecureSkipVerify: true}
	if err := c.StartTLS(tlsConfig); err != nil {
		// STARTTLS not supported, continue without
	}

	if cfg.SMTPUser != "" {
		auth := smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPHost)
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("auth: %w", err)
		}
	}

	if err := c.Mail(cfg.SMTPFrom); err != nil {
		return fmt.Errorf("mail from: %w", err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("rcpt to: %w", err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close data: %w", err)
	}
	return c.Quit()
}
