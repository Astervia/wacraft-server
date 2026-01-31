package email_service

// EmailService defines the interface for sending emails
type EmailService interface {
	SendVerificationEmail(to, name, token, baseURL string) error
	SendWorkspaceInvitation(to, workspaceName, inviterName, token, baseURL string) error
	SendPasswordReset(to, name, token string) error
}

// EmailData contains common email template data
type EmailData struct {
	Name         string
	Token        string
	BaseURL      string
	WorkspaceName string
	InviterName  string
}
