package mailer

import (
	"fmt"
	"net/smtp"
	"strings"

	"github.com/codetheuri/todolist/config"
	appErrors "github.com/codetheuri/todolist/pkg/errors"
	"github.com/codetheuri/todolist/pkg/logger"
)

type MailerService interface {
	SendEmail(to []string, subject string, body string) error
	SendWWelcomeEmail(recipientEmail string) error
}

type smtpMailer struct {
	cfg *config.Config
	log logger.Logger
}

// new instance of mailer service
func NewMailerService(cfg *config.Config, log logger.Logger) MailerService {
	return &smtpMailer{
		cfg: cfg,
		log: log,
	}
}
func (s *smtpMailer) SendEmail(to []string, subject string, body string) error {
	if s.cfg.MailerHost == "" || s.cfg.MailerPort == 0 || s.cfg.MailerUsername == "" || s.cfg.MailerPassword == "" {
		s.log.Error("Mailer config is incomplete. Skipping email send.", nil)
		return appErrors.InternalServerError("email service not fully configured", nil)
	}

	from := s.cfg.MailerSender
	auth := smtp.PlainAuth("", s.cfg.MailerUsername, s.cfg.MailerPassword, s.cfg.MailerHost)
	addr := fmt.Sprintf("%s:%d", s.cfg.MailerHost, s.cfg.MailerPort)

	msg := []byte(
		"From: " + from + "\r\n" +
			"To: " + strings.Join(to, ",") + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"\r\n" +
			body,
	)

	s.log.Info("Attempting to send email", "to", to, "subject", subject)

	err := smtp.SendMail(addr, auth, from, to, []byte(msg))
	if err != nil {
		s.log.Error("Failed to send email via SMTP", err, "to", to, "subject", subject)
		return appErrors.InternalServerError("failed to send email", err)
	}

	s.log.Info("Email sent successfully", "to", to, "subject", subject)
	return nil
}

func (s *smtpMailer) SendWWelcomeEmail(recipientEmail string) error {
	subject := "Welcome to Tusk"
	body := "Thank you for signing up for Tusk!"
	return s.SendEmail([]string{recipientEmail}, subject, body)
}
