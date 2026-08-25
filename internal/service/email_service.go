package service

import (
	"fmt"
	"net/smtp"
	"strings"
)

type EmailService struct {
	host     string
	port     int
	user     string
	password string
	from     string
}

func NewEmailService(host string, port int, user, password, from string) *EmailService {
	return &EmailService{host: host, port: port, user: user, password: password, from: from}
}

func (s *EmailService) Send(to, subject, htmlBody string) error {
	addr := fmt.Sprintf("%s:%d", s.host, s.port)

	headers := map[string]string{
		"From":         s.from,
		"To":           to,
		"Subject":      subject,
		"MIME-Version": "1.0",
		"Content-Type": "text/html; charset=\"UTF-8\"",
	}

	var msg strings.Builder
	for k, v := range headers {
		fmt.Fprintf(&msg, "%s: %s\r\n", k, v)
	}
	msg.WriteString("\r\n")
	msg.WriteString(htmlBody)

	var auth smtp.Auth
	if s.user != "" {
		auth = smtp.PlainAuth("", s.user, s.password, s.host)
	}

	return smtp.SendMail(addr, auth, s.from, []string{to}, []byte(msg.String()))
}
