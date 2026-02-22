package mail

import (
	"context"
	"fmt"
	"log"
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

func send(cfg *config.Config, to, subject, body string) error {
	addr := fmt.Sprintf("%s:%s", cfg.SMTPHost, cfg.SMTPPort)
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		cfg.SMTPFrom, to, subject, body)

	var auth smtp.Auth
	if cfg.SMTPUser != "" {
		auth = smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPHost)
	}

	return smtp.SendMail(addr, auth, cfg.SMTPFrom, []string{to}, []byte(msg))
}
