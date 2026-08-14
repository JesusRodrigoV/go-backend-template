package handler

import (
	"net/http"
	"time"

	"github.com/JesusRodrigoV/granBackend/internal/core/ports"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type userResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Verified  bool      `json:"verified"`
	CreatedAt time.Time `json:"created_at"`
}

type AuthHandler struct {
	service ports.AuthService
}

func NewAuthHandler(service ports.AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

type registerReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Register(c *echo.Context) error {
	var req registerReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
	}

	user, err := h.service.Register(req.Email, req.Password)
	if err != nil {
		return c.JSON(http.StatusConflict, map[string]string{"error": err.Error()})
	}

	response := userResponse{
		ID:        user.ID,
		Email:     user.Email,
		Verified:  user.Verified,
		CreatedAt: user.CreatedAt,
	}

	return c.JSON(http.StatusCreated, response)
}

func (h *AuthHandler) Login(c *echo.Context) error {
	var req loginReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
	}

	ip := c.RealIP()
	userAgent := c.Request().UserAgent()

	token1, token2, require2FA, err := h.service.Login(req.Email, req.Password, ip, userAgent)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	if require2FA {
		return c.JSON(http.StatusAccepted, map[string]any{
			"message":    "2FA requerido",
			"temp_token": token2,
			"2fa":        true,
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"access_token":  token1,
		"refresh_token": token2,
		"type":          "Bearer",
		"2fa":           false,
	})
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *AuthHandler) Refresh(c *echo.Context) error {
	var req refreshReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
	}

	ip := c.RealIP()
	userAgent := c.Request().UserAgent()

	accessToken, refreshToken, err := h.service.RefreshToken(req.RefreshToken, ip, userAgent)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"type":          "Bearer",
	})
}

type logoutReq struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *AuthHandler) Logout(c *echo.Context) error {
	var req logoutReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
	}

	if err := h.service.Logout(req.RefreshToken); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error al cerrar sesión"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Sesión cerrada exitosamente"})
}

type sessionResponse struct {
	Device    string    `json:"device"`
	IP        string    `json:"ip"`
	CreatedAt time.Time `json:"created_at"`
	IsCurrent bool      `json:"is_current"`
}

func (h *AuthHandler) GetSessions(c *echo.Context) error {
	userID := c.Get("user_id").(string)

	sessions, err := h.service.ListSessions(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error obteniendo sesiones"})
	}

	var response []sessionResponse
	for _, s := range sessions {
		response = append(response, sessionResponse{
			Device:    s.Device,
			IP:        s.IP,
			CreatedAt: s.CreatedAt,
			IsCurrent: false,
		})
	}

	return c.JSON(http.StatusOK, response)
}

func (h *AuthHandler) LogoutAll(c *echo.Context) error {
	userID := c.Get("user_id").(string)

	if err := h.service.LogoutAll(userID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Error al cerrar sesiones globales",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Se han cerrado todas las sesiones en todos los dispositivos.",
	})
}

type mfaCodeReq struct {
	Code string `json:"code"`
}

func (h *AuthHandler) SetupMFA(c *echo.Context) error {
	userID := c.Get("user_id").(string)

	secret, qrURL, err := h.service.SetupMFA(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"secret": secret,
		"qr_url": qrURL,
	})
}

func (h *AuthHandler) EnableMFA(c *echo.Context) error {
	userID := c.Get("user_id").(string)

	var req mfaCodeReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
	}

	backupCodes, err := h.service.EnableMFA(userID, req.Code)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"message":      "MFA Activado correctamente",
		"backup_codes": backupCodes,
	})
}

func (h *AuthHandler) DisableMFA(c *echo.Context) error {
	userID := c.Get("user_id").(string)

	if err := h.service.DisableMFA(userID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "MFA Desactivado"})
}

type verifyLoginReq struct {
	TempToken string `json:"temp_token"`
	Code      string `json:"code"`
}

func (h *AuthHandler) VerifyLoginMFA(c *echo.Context) error {
	var req verifyLoginReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
	}

	ip := c.RealIP()
	userAgent := c.Request().UserAgent()

	accessToken, refreshToken, err := h.service.VerifyLoginMFA(req.TempToken, req.Code, ip, userAgent)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"type":          "Bearer",
	})
}

type forgotReq struct {
	Email string `json:"email"`
}

type resetReq struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

func (h *AuthHandler) ForgotPassword(c *echo.Context) error {
	var req forgotReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
	}
	err := h.service.ForgotPassword(req.Email)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error enviando correo o usuario no existe"})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Si el correo existe, se ha enviado un enlace de recuperación.",
	})
}

func (h *AuthHandler) ResetPassword(c *echo.Context) error {
	var req resetReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
	}

	if err := h.service.ResetPassword(req.Token, req.NewPassword); err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Contraseña actualizada exitosamente. Todas las sesiones han sido cerradas.",
	})
}

type logResponse struct {
	Event     string    `json:"event"`
	IP        string    `json:"ip"`
	Device    string    `json:"device"`
	CreatedAt time.Time `json:"created_at"`
	Status    string    `json:"status"`
}

func (h *AuthHandler) GetActivityLogs(c *echo.Context) error {
	userID := c.Get("user_id").(string)

	logs, err := h.service.GetActivityLogs(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error obteniendo logs"})
	}

	var response []logResponse
	for _, l := range logs {
		ip, _ := l.Meta["ip"].(string)
		device, _ := l.Meta["device"].(string)

		status := "Exitoso"
		switch l.Event {
		case "LOGIN_FAILED", "MFA_FAILED":
			status = "Fallido"
		case "LOGIN_MFA_CHALLENGE":
			status = "Pendiente MFA"
		}

		response = append(response, logResponse{
			Event:     l.Event,
			IP:        ip,
			Device:    device,
			CreatedAt: l.CreatedAt,
			Status:    status,
		})
	}

	return c.JSON(http.StatusOK, response)
}

func (h *AuthHandler) GoogleLogin(c *echo.Context) error {
	url := h.service.GetGoogleAuthURL()
	return c.Redirect(http.StatusTemporaryRedirect, url)
}

func (h *AuthHandler) GoogleCallback(c *echo.Context) error {
	code := c.QueryParam("code")
	if code == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "No se recibió código de Google"})
	}

	ip := c.RealIP()
	userAgent := c.Request().UserAgent()

	accessToken, refreshToken, require2FA, err := h.service.LoginWithGoogle(code, ip, userAgent)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	if require2FA {
		// En un flujo real de navegador, aquí redirigirías a una página de frontend:
		// return c.Redirect(302, "http://tu-frontend.com/verify-2fa?temp_token=" + refreshToken)

		return c.JSON(http.StatusAccepted, map[string]any{
			"message":    "2FA requerido. Usa el temp_token en /login-verify",
			"temp_token": refreshToken,
			"2fa":        true,
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"type":          "Bearer",
		"message":       "Bienvenido vía Google",
	})
}

func (h *AuthHandler) VerifyEmail(c *echo.Context) error {
	token := c.QueryParam("token")
	if token == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Token faltante"})
	}

	if err := h.service.VerifyEmail(token); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "¡Cuenta activada! Ya puedes entrar."})
}
