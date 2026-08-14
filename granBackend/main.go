package main

import (
	"log/slog"
	"os"

	"github.com/JesusRodrigoV/granBackend/config"
	_ "github.com/JesusRodrigoV/granBackend/docs"
	"github.com/JesusRodrigoV/granBackend/internal/adapters/api"
	"github.com/JesusRodrigoV/granBackend/internal/adapters/handler"
	"github.com/JesusRodrigoV/granBackend/internal/adapters/mail"
	"github.com/JesusRodrigoV/granBackend/internal/adapters/repository"
	"github.com/JesusRodrigoV/granBackend/internal/core/services"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

func main() {
	// Todo esto es para iniciar la API
	cfg := config.LoadConfig()
	db := config.InitDB(cfg)
	redisClient := config.InitRedis()

	sqlDB, err := db.DB()
	if err == nil {
		if err := sqlDB.Ping(); err != nil {
			slog.Error("La base de datos no responde", "error", err)
			os.Exit(1)
		}
		slog.Info("BD conectada, celebrar")
	}

	//Las dependencias
	healthH := handler.NewHealthHandler(db)
	userRepo := repository.NewUserRepo(db)
	emailService := mail.NewSMTPEmailService(cfg)
	authService := services.NewAuthService(userRepo, emailService, cfg)
	authHandler := handler.NewAuthHandler(authService)

	//Configuracion de echo y mi poderoso middleware
	e := echo.New()

	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())

	// Llamamos al router para configurar las rutas
	api.SetupRoutes(e, cfg, redisClient, authHandler, healthH)

	// Inicio del gran servidor
	addr := ":" + cfg.Port
	slog.Info("Iniciando servidor", "address", addr)

	if err := e.Start(addr); err != nil {
		slog.Error("failed to start server", "error", err)
	}
}
