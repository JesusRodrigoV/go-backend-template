package mail

import (
	"fmt"
	"net/smtp"

	"github.com/JesusRodrigoV/granBackend/config"
	"github.com/JesusRodrigoV/granBackend/internal/core/ports"
)

type SMTPEmailService struct {
	cfg  *config.Config
	auth smtp.Auth
}

func NewSMTPEmailService(cfg *config.Config) ports.EmailService {
	return &SMTPEmailService{
		cfg:  cfg,
		auth: smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPHost),
	}
}

func (s *SMTPEmailService) send(to, subject, body string) error {
	msg := []byte(fmt.Sprintf("To: %s\r\n"+
		"From: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/html; charset=\"UTF-8\"\r\n"+
		"\r\n"+
		"%s\r\n", to, s.cfg.SMTPFrom, subject, body))

	addr := fmt.Sprintf("%s:%s", s.cfg.SMTPHost, s.cfg.SMTPPort)
	return smtp.SendMail(addr, s.auth, s.cfg.SMTPFrom, []string{to}, msg)
}

func (s *SMTPEmailService) SendWelcomeEmail(to, name, link string) error {
	subject := "Bienvenido a GranBackend - Activa tu cuenta"
	body := fmt.Sprintf(`
		<h1>Hola %s, bienvenido!</h1>
		<p>Gracias por registrarte. Por favor confirma tu correo haciendo click aquí:</p>
		<a href="%s">ACTIVAR CUENTA</a>
		<p>Si no fuiste tú, ignora este mensaje.</p>
	`, name, link)
	return s.send(to, subject, body)
}

func (s *SMTPEmailService) SendResetPasswordEmail(to, link string) error {
	subject := "Recuperación de Contraseña"
	body := fmt.Sprintf(`
		<h1>Recuperar acceso</h1>
		<p>Hemos recibido una solicitud para cambiar tu contraseña.</p>
		<a href="%s">RESET PASSWORD</a>
		<p>Este enlace expira en 15 minutos.</p>
	`, link)
	return s.send(to, subject, body)
}

func (s *SMTPEmailService) SendSecurityAlert(to, ip string) error {
	subject := "ALERTA DE SEGURIDAD: Cambio de contraseña"
	body := fmt.Sprintf(`
		<h1>Tu contraseña ha sido cambiada</h1>
		<p>Se ha realizado un cambio de contraseña desde la IP: <strong>%s</strong>.</p>
		<p>Todas las sesiones activas han sido cerradas por seguridad.</p>
	`, ip)
	return s.send(to, subject, body)
}
