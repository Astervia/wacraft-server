package email_service

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"

	"github.com/Astervia/wacraft-server/src/config/env"
)

// SMTPService implements EmailService using SMTP
type SMTPService struct {
	host     string
	port     string
	user     string
	password string
	from     string
	baseURL  string
}

// NewSMTPService creates a new SMTP email service
func NewSMTPService() *SMTPService {
	return &SMTPService{
		host:     env.SMTPHost,
		port:     env.SMTPPort,
		user:     env.SMTPUser,
		password: env.SMTPPassword,
		from:     env.SMTPFrom,
		baseURL:  env.AppBaseURL,
	}
}

func (s *SMTPService) sendEmail(to, subject, body string) error {
	if s.host == "" {
		// Log-only mode if SMTP not configured
		fmt.Printf("[EMAIL] To: %s, Subject: %s\nBody:\n%s\n", to, subject, body)
		return nil
	}

	auth := smtp.PlainAuth("", s.user, s.password, s.host)

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s",
		s.from, to, subject, body)

	addr := fmt.Sprintf("%s:%s", s.host, s.port)
	return smtp.SendMail(addr, auth, s.from, []string{to}, []byte(msg))
}

func (s *SMTPService) SendVerificationEmail(to, name, token, baseURL string) error {
	subject := "Verify your email address"

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2>Welcome, {{.Name}}!</h2>
        <p>Thank you for registering. Please verify your email address by clicking the button below:</p>
        <p style="text-align: center; margin: 30px 0;">
            <a href="{{.BaseURL}}/auth/verify-email?token={{.Token}}"
               style="background-color: #4CAF50; color: white; padding: 14px 28px; text-decoration: none; border-radius: 4px; display: inline-block;">
                Verify Email
            </a>
        </p>
        <p>Or copy and paste this link into your browser:</p>
        <p style="word-break: break-all; color: #666;">{{.BaseURL}}/auth/verify-email?token={{.Token}}</p>
        <p>This link will expire in 24 hours.</p>
        <hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;">
        <p style="color: #999; font-size: 12px;">If you didn't create an account, you can safely ignore this email.</p>
    </div>
</body>
</html>
`
	t, err := template.New("verification").Parse(tmpl)
	if err != nil {
		return err
	}

	if baseURL == "" {
		baseURL = s.baseURL
	}

	var body bytes.Buffer
	err = t.Execute(&body, EmailData{
		Name:    name,
		Token:   token,
		BaseURL: baseURL,
	})
	if err != nil {
		return err
	}

	return s.sendEmail(to, subject, body.String())
}

func (s *SMTPService) SendWorkspaceInvitation(to, workspaceName, inviterName, token, baseURL string) error {
	subject := fmt.Sprintf("You've been invited to join %s", workspaceName)

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2>You've been invited!</h2>
        <p><strong>{{.InviterName}}</strong> has invited you to join the workspace <strong>{{.WorkspaceName}}</strong>.</p>
        <p style="text-align: center; margin: 30px 0;">
            <a href="{{.BaseURL}}/auth/accept-invitation?token={{.Token}}"
               style="background-color: #2196F3; color: white; padding: 14px 28px; text-decoration: none; border-radius: 4px; display: inline-block;">
                Accept Invitation
            </a>
        </p>
        <p>Or copy and paste this link into your browser:</p>
        <p style="word-break: break-all; color: #666;">{{.BaseURL}}/auth/accept-invitation?token={{.Token}}</p>
        <p>This invitation will expire in 7 days.</p>
        <hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;">
        <p style="color: #999; font-size: 12px;">If you don't want to join this workspace, you can safely ignore this email.</p>
    </div>
</body>
</html>
`
	t, err := template.New("invitation").Parse(tmpl)
	if err != nil {
		return err
	}

	if baseURL == "" {
		baseURL = s.baseURL
	}

	var body bytes.Buffer
	err = t.Execute(&body, EmailData{
		Token:         token,
		BaseURL:       baseURL,
		WorkspaceName: workspaceName,
		InviterName:   inviterName,
	})
	if err != nil {
		return err
	}

	return s.sendEmail(to, subject, body.String())
}

func (s *SMTPService) SendPasswordReset(to, name, token string) error {
	subject := "Reset your password"

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2>Password Reset Request</h2>
        <p>Hi {{.Name}},</p>
        <p>We received a request to reset your password. Click the button below to create a new password:</p>
        <p style="text-align: center; margin: 30px 0;">
            <a href="{{.BaseURL}}/auth/reset-password?token={{.Token}}"
               style="background-color: #FF9800; color: white; padding: 14px 28px; text-decoration: none; border-radius: 4px; display: inline-block;">
                Reset Password
            </a>
        </p>
        <p>Or copy and paste this link into your browser:</p>
        <p style="word-break: break-all; color: #666;">{{.BaseURL}}/auth/reset-password?token={{.Token}}</p>
        <p>This link will expire in 1 hour.</p>
        <hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;">
        <p style="color: #999; font-size: 12px;">If you didn't request a password reset, you can safely ignore this email. Your password will not be changed.</p>
    </div>
</body>
</html>
`
	t, err := template.New("reset").Parse(tmpl)
	if err != nil {
		return err
	}

	var body bytes.Buffer
	err = t.Execute(&body, EmailData{
		Name:    name,
		Token:   token,
		BaseURL: s.baseURL,
	})
	if err != nil {
		return err
	}

	return s.sendEmail(to, subject, body.String())
}

// Default email service instance
var DefaultEmailService EmailService = NewSMTPService()
