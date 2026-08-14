package services

import (
	"errors"
	"fmt"
	"time"
	"unicode"

	"github.com/JesusRodrigoV/granBackend/config"
	"github.com/JesusRodrigoV/granBackend/internal/core/domain"
	"github.com/JesusRodrigoV/granBackend/internal/core/ports"
	"github.com/JesusRodrigoV/granBackend/pkg/utils"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
)

type AuthService struct {
	repo      ports.UserRepository
	jwtSecret string
	cfg       *config.Config
	emailer   ports.EmailService
}

func NewAuthService(repo ports.UserRepository, emailer ports.EmailService, cfg *config.Config) *AuthService {
	return &AuthService{
		repo:      repo,
		jwtSecret: cfg.JWTSecret,
		cfg:       cfg,
		emailer:   emailer,
	}
}

func (s *AuthService) Register(email, password string) (*domain.User, error) {
	if err := s.validatePassword(password); err != nil {
		return nil, err
	}

	_, err := s.repo.GetByEmail(email)
	if err == nil {
		return nil, errors.New("el usuario ya existe")
	}

	hashedPwd, err := utils.HashPassword(password)
	if err != nil {
		return nil, err
	}

	newUser := &domain.User{
		ID:       uuid.New(),
		Email:    email,
		Password: hashedPwd,
		Verified: false,
	}

	if err := s.repo.Save(newUser); err != nil {
		return nil, err
	}

	token, err := utils.GenerateResetToken(newUser.ID.String(), s.jwtSecret)
	if err == nil {
		link := fmt.Sprintf("http://localhost:1323/api/v1/auth/verify-email?token=%s", token)
		go s.emailer.SendWelcomeEmail(newUser.Email, "Usuario", link)
	}

	return newUser, nil
}

func (s *AuthService) Login(email, password, ip, userAgent string) (string, string, bool, error) {
	user, err := s.repo.GetByEmail(email)
	if err != nil {
		return "", "", false, errors.New("credenciales inválidas")
	}

	if !utils.CheckPassword(password, user.Password) {
		s.logEvent(user.ID, "LOGIN_FAILED", ip, userAgent, map[string]any{"reason": "bad_password"})
		return "", "", false, errors.New("credenciales inválidas")
	}

	if !user.Verified {
		return "", "", false, errors.New("debes verificar tu correo electrónico para iniciar sesión")
	}

	if user.Mfa {
		s.logEvent(user.ID, "LOGIN_MFA_CHALLENGE", ip, userAgent, nil)
		tempToken, err := utils.GenerateTempToken(user.ID.String(), s.jwtSecret)
		if err != nil {
			return "", "", false, err
		}
		return "", tempToken, true, nil
	}

	accessToken, err := utils.GenerateJWT(user.ID.String(), s.jwtSecret)
	if err != nil {
		return "", "", false, err
	}

	refreshTokenRaw, err := utils.GenerateRandomString(32)
	if err != nil {
		return "", "", false, err
	}

	refreshTokenHash := utils.HashToken(refreshTokenRaw)

	newSession := &domain.Session{
		ID:        uuid.New(),
		UserID:    user.ID,
		Hash:      refreshTokenHash,
		Device:    userAgent,
		IP:        ip,
		Exp:       time.Now().Add(time.Hour * 24 * 30),
		Revoked:   false,
		CreatedAt: time.Now(),
	}

	if err := s.repo.SaveSession(newSession); err != nil {
		return "", "", false, err
	}

	s.logEvent(user.ID, "LOGIN_SUCCESS", ip, userAgent, nil)

	return accessToken, refreshTokenRaw, false, nil
}

func (s *AuthService) validatePassword(password string) error {
	var (
		hasMinLen = false
		hasUpper  = false
		hasLower  = false
		hasNumber = false
		// hasSpecial = false
	)

	if len(password) >= 8 {
		hasMinLen = true
	}

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
			// case unicode.IsPunct(char) || unicode.IsSymbol(char):
			// 	hasSpecial = true
		}
	}

	if !hasMinLen {
		return errors.New("la contraseña debe tener al menos 8 caracteres")
	}
	if !hasUpper {
		return errors.New("la contraseña debe tener al menos una mayúscula")
	}
	if !hasLower {
		return errors.New("la contraseña debe tener al menos una minúscula")
	}
	if !hasNumber {
		return errors.New("la contraseña debe tener al menos un número")
	}

	return nil
}

func (s *AuthService) RefreshToken(oldToken, ip, userAgent string) (string, string, error) {
	oldHash := utils.HashToken(oldToken)

	session, err := s.repo.GetSessionByHash(oldHash)
	if err != nil {
		return "", "", errors.New("sesión inválida o expirada")
	}

	if session.Revoked {
		return "", "", errors.New("sesión revocada")
	}
	if session.Exp.Before(time.Now()) {
		return "", "", errors.New("sesión expirada")
	}

	newAccessToken, err := utils.GenerateJWT(session.UserID.String(), s.jwtSecret)
	if err != nil {
		return "", "", err
	}

	newRefreshTokenRaw, err := utils.GenerateRandomString(32)
	if err != nil {
		return "", "", err
	}

	session.Hash = utils.HashToken(newRefreshTokenRaw)
	session.IP = ip
	session.Device = userAgent
	session.Exp = time.Now().Add(time.Hour * 24 * 30)

	if err := s.repo.UpdateSession(session); err != nil {
		return "", "", err
	}

	return newAccessToken, newRefreshTokenRaw, nil
}

func (s *AuthService) Logout(refreshToken string) error {
	hash := utils.HashToken(refreshToken)

	if err := s.repo.RevokeSession(hash); err != nil {
		return err
	}

	return nil
}

func (s *AuthService) ListSessions(userID string) ([]domain.Session, error) {
	return s.repo.GetSessionsByUserID(userID)
}

func (s *AuthService) LogoutAll(userID string) error {
	return s.repo.RevokeAllSessions(userID)
}

func (s *AuthService) SetupMFA(userID string) (string, string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "GranBackend",
		AccountName: userID,
	})
	if err != nil {
		return "", "", err
	}

	otp := &domain.OTP{
		UserID: uuid.MustParse(userID),
		Secret: key.Secret(),
		Backup: []string{},
	}

	if err := s.repo.SaveOTP(otp); err != nil {
		return "", "", err
	}

	return key.Secret(), key.URL(), nil
}

func (s *AuthService) EnableMFA(userID, code string) ([]string, error) {
	otpData, err := s.repo.GetOTP(userID)
	if err != nil {
		return nil, errors.New("debe ejecutar setup primero")
	}

	if !totp.Validate(code, otpData.Secret) {
		return nil, errors.New("código inválido")
	}

	backupCodes, err := utils.GenerateBackupCodes(10)
	if err != nil {
		return nil, err
	}

	otpData.Backup = backupCodes
	if err := s.repo.SaveOTP(otpData); err != nil {
		return nil, err
	}

	if err := s.repo.SetMFAStatus(userID, true); err != nil {
		return nil, err
	}

	return backupCodes, nil
}

func (s *AuthService) DisableMFA(userID string) error {
	if err := s.repo.DeleteOTP(userID); err != nil {
		return err
	}
	return s.repo.SetMFAStatus(userID, false)
}

func (s *AuthService) VerifyMFA(userID, code string) (bool, error) {
	otpData, err := s.repo.GetOTP(userID)
	if err != nil {
		return false, errors.New("mfa no configurado")
	}

	if totp.Validate(code, otpData.Secret) {
		return true, nil
	}

	for i, backup := range otpData.Backup {
		if backup == code {
			newBackups := append(otpData.Backup[:i], otpData.Backup[i+1:]...)

			if err := s.repo.UpdateBackupCodes(userID, newBackups); err != nil {
				return false, err
			}
			return true, nil
		}
	}

	return false, nil
}

func (s *AuthService) VerifyLoginMFA(tempTokenString, code, ip, userAgent string) (string, string, error) {
	token, err := jwt.Parse(tempTokenString, func(token *jwt.Token) (any, error) {
		return []byte(s.jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return "", "", errors.New("token temporal inválido o expirado")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", errors.New("claims inválidos")
	}

	if pur, ok := claims["pur"].(string); !ok || pur != "2fa_pending" {
		return "", "", errors.New("token incorrecto para verificación 2fa")
	}

	userIDStr, ok := claims["sub"].(string)
	if !ok {
		return "", "", errors.New("user_id no encontrado")
	}

	valid, err := s.VerifyMFA(userIDStr, code)
	if err != nil {
		return "", "", err
	}
	uid := uuid.MustParse(userIDStr)
	if !valid {
		s.logEvent(uid, "MFA_FAILED", ip, userAgent, map[string]any{"code_attempt": "invalid"})
		return "", "", errors.New("código incorrecto")
	}

	accessToken, err := utils.GenerateJWT(userIDStr, s.jwtSecret)
	if err != nil {
		return "", "", err
	}

	refreshTokenRaw, _ := utils.GenerateRandomString(32)
	refreshTokenHash := utils.HashToken(refreshTokenRaw)

	newSession := &domain.Session{
		ID:        uuid.New(),
		UserID:    uuid.MustParse(userIDStr),
		Hash:      refreshTokenHash,
		Device:    userAgent,
		IP:        ip,
		Exp:       time.Now().Add(time.Hour * 24 * 30),
		CreatedAt: time.Now(),
	}

	if err := s.repo.SaveSession(newSession); err != nil {
		return "", "", err
	}

	s.logEvent(uid, "LOGIN_SUCCESS_MFA", ip, userAgent, nil)

	return accessToken, refreshTokenRaw, nil
}

func (s *AuthService) ForgotPassword(email string) error {
	user, err := s.repo.GetByEmail(email)
	if err != nil {
		return nil
	}

	token, err := utils.GenerateResetToken(user.ID.String(), s.jwtSecret)
	if err != nil {
		return err
	}

	link := fmt.Sprintf("http://localhost:3000/reset-password?token=%s", token)

	go s.emailer.SendResetPasswordEmail(user.Email, link)

	return nil
}

func (s *AuthService) ResetPassword(tokenString, newPassword string) error {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		return []byte(s.jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return errors.New("enlace inválido o expirado")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return errors.New("claims inválidos")
	}

	if pur, ok := claims["pur"].(string); !ok || pur != "reset_pass" {
		return errors.New("token inválido para esta operación")
	}

	userID, ok := claims["sub"].(string)
	if !ok {
		return errors.New("user_id faltante")
	}

	newHash, err := utils.HashPassword(newPassword)
	if err != nil {
		return err
	}

	if err := s.repo.UpdatePassword(userID, newHash); err != nil {
		return err
	}

	return s.repo.RevokeAllSessions(userID)
}

func (s *AuthService) logEvent(userID uuid.UUID, event, ip, device string, extra map[string]any) {
	meta := map[string]any{
		"ip":     ip,
		"device": device,
	}
	for k, v := range extra {
		meta[k] = v
	}

	s.repo.SaveLog(&domain.Log{
		UserID: userID,
		Event:  event,
		Meta:   meta,
	})
}

func (s *AuthService) GetActivityLogs(userID string) ([]domain.Log, error) {
	return s.repo.GetLogsByUserID(userID)
}

func (s *AuthService) VerifyEmail(tokenString string) error {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return errors.New("token inválido")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return errors.New("claims inválidos")
	}

	userID, ok := claims["sub"].(string)
	if !ok {
		return errors.New("token mal formado")
	}

	return s.repo.VerifyUser(userID)
}
