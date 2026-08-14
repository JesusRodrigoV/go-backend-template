package ports

type EmailService interface {
	SendWelcomeEmail(to, name, activationLink string) error
	SendResetPasswordEmail(to, resetLink string) error
	SendSecurityAlert(to, deviceIP string) error
}
