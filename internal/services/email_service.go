package services

import (
	"context"
	"fmt"
	"net/smtp"
	"strconv"
	"strings"
)

type EmailService struct {
	settings *SettingsService
}

func NewEmailService(settings *SettingsService) *EmailService {
	return &EmailService{settings: settings}
}

func (s *EmailService) IsEnabled(ctx context.Context) bool {
	host, _ := s.settings.Get(ctx, SettingSMTPHost)
	user, _ := s.settings.Get(ctx, SettingSMTPUser)
	from, _ := s.settings.Get(ctx, SettingSMTPFrom)
	return strings.TrimSpace(host) != "" && strings.TrimSpace(user) != "" && strings.TrimSpace(from) != ""
}

func (s *EmailService) Send(ctx context.Context, to, subject, body string) error {
	if !s.IsEnabled(ctx) {
		return fmt.Errorf("envoi d'email désactivé (configuration SMTP incomplète)")
	}
	host, _ := s.settings.Get(ctx, SettingSMTPHost)
	portStr, _ := s.settings.Get(ctx, SettingSMTPPort)
	user, _ := s.settings.Get(ctx, SettingSMTPUser)
	password, _ := s.settings.Get(ctx, SettingSMTPPassword)
	from, _ := s.settings.Get(ctx, SettingSMTPFrom)

	port, err := strconv.Atoi(strings.TrimSpace(portStr))
	if err != nil || port <= 0 {
		port = 587
	}
	addr := fmt.Sprintf("%s:%d", strings.TrimSpace(host), port)

	msg := []byte(fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		from, to, subject, body))

	auth := smtp.PlainAuth("", user, password, strings.TrimSpace(host))
	return smtp.SendMail(addr, auth, from, []string{to}, msg)
}
