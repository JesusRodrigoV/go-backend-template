package api

import (
	"net/http"
	"time"

	"github.com/JesusRodrigoV/granBackend/config"
	"github.com/JesusRodrigoV/granBackend/internal/adapters/handler"
	guards "github.com/JesusRodrigoV/granBackend/internal/adapters/middleware"
	"github.com/labstack/echo/v5"
	"github.com/redis/go-redis/v9"
	httpSwagger "github.com/swaggo/http-swagger"
)

func SetupRoutes(e *echo.Echo, cfg *config.Config, redisClient *redis.Client, authHandler *handler.AuthHandler, healthH *handler.HealthHandler) {
	registerSystemRoutes(e, healthH)
	apiV1 := e.Group("/api/v1")
	registerAuthRoutes(apiV1, authHandler, redisClient)
	registerProtectedRoutes(apiV1, authHandler, cfg)
}

func registerSystemRoutes(e *echo.Echo, h *handler.HealthHandler) {
	e.GET("/", hello)
	e.GET("/health", h.Check)
	e.GET("/docs/*", echo.WrapHandler(httpSwagger.Handler(
		httpSwagger.URL("http://localhost:1323/docs/doc.json"),
	)))
}

func registerAuthRoutes(g *echo.Group, h *handler.AuthHandler, redisClient *redis.Client) {
	auth := g.Group("/auth")

	limiterConfig := guards.RateLimiterConfig{
		Limit:  5,
		Burst:  5,
		Window: 1 * time.Minute,
	}
	auth.Use(guards.RedisRateLimiter(redisClient, limiterConfig))

	// Endpoints
	auth.POST("/register", h.Register)
	auth.POST("/login", h.Login)
	auth.POST("/refresh", h.Refresh)
	auth.POST("/login-verify", h.VerifyLoginMFA)
	auth.POST("/forgot-password", h.ForgotPassword)
	auth.POST("/reset-password", h.ResetPassword)
	auth.GET("/google/login", h.GoogleLogin)
	auth.GET("/google/callback", h.GoogleCallback)
	auth.GET("/verify-email", h.VerifyEmail)
}

func registerProtectedRoutes(g *echo.Group, h *handler.AuthHandler, cfg *config.Config) {
	protected := g.Group("")
	protected.Use(guards.JWTGuard(cfg.JWTSecret))

	protected.POST("/auth/logout", h.Logout)
	protected.POST("/auth/logout-all", h.LogoutAll)
	protected.GET("/sessions", h.GetSessions)
	protected.GET("/activity-logs", h.GetActivityLogs)
	protected.POST("/auth/mfa/setup", h.SetupMFA)
	protected.POST("/auth/mfa/enable", h.EnableMFA)
	protected.POST("/auth/mfa/disable", h.DisableMFA)
	protected.GET("/users/me", func(c *echo.Context) error {
		userID := c.Get("user_id").(string)
		return c.JSON(http.StatusOK, map[string]string{
			"mensaje": "Zona Protegida",
			"usuario": userID,
		})
	})
}

func hello(c *echo.Context) error {
	return c.String(http.StatusOK, "Primer Saludo desde backend Go, somos grandes")
}
