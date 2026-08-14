package ports

import "github.com/JesusRodrigoV/granBackend/internal/core/domain"

type AuthService interface {
	Register(email, password string) (*domain.User, error)
	Login(email, password, ip, userAgent string) (string, string, bool, error)
	RefreshToken(oldRefreshToken, ip, userAgent string) (string, string, error)
	Logout(refreshToken string) error
	ListSessions(userID string) ([]domain.Session, error)
	LogoutAll(userID string) error
	SetupMFA(userID string) (string, string, error)
	EnableMFA(userID, code string) ([]string, error)
	DisableMFA(userID string) error
	VerifyMFA(userID, code string) (bool, error)
	VerifyLoginMFA(tempTokenString, code, ip, userAgent string) (string, string, error)
	ForgotPassword(email string) error
	ResetPassword(token, newPassword string) error
	GetActivityLogs(userID string) ([]domain.Log, error)
	GetGoogleAuthURL() string
	LoginWithGoogle(code, ip, userAgent string) (string, string, bool, error)
	VerifyEmail(token string) error
}

type UserRepository interface {
	Save(user *domain.User) error
	GetByEmail(email string) (*domain.User, error)
	SaveSession(session *domain.Session) error
	GetSessionByHash(hash string) (*domain.Session, error)
	UpdateSession(session *domain.Session) error
	RevokeSession(hash string) error
	GetSessionsByUserID(userID string) ([]domain.Session, error)
	RevokeAllSessions(userID string) error
	SaveOTP(otp *domain.OTP) error
	GetOTP(userID string) (*domain.OTP, error)
	DeleteOTP(userID string) error
	SetMFAStatus(userID string, status bool) error
	UpdateBackupCodes(userID string, codes []string) error
	UpdatePassword(userID string, hash string) error
	SaveLog(log *domain.Log) error
	GetLogsByUserID(userID string) ([]domain.Log, error)
	FindSocial(provider, provID string) (*domain.Social, error)
	SaveSocial(social *domain.Social) error
	VerifyUser(userID string) error
}
