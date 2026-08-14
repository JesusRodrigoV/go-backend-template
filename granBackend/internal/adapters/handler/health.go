package handler

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	"gorm.io/gorm"
)

type HealthHandler struct {
	DB *gorm.DB
}

func NewHealthHandler(db *gorm.DB) *HealthHandler {
	return &HealthHandler{DB: db}
}

type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services"`
}

func (h *HealthHandler) Check(c *echo.Context) error {
	status := "UP"
	httpStatus := http.StatusOK
	dbStatus := "UP"

	sqlDB, err := h.DB.DB()
	if err != nil || sqlDB.Ping() != nil {
		status = "DOWN"
		dbStatus = "DOWN"
		httpStatus = http.StatusServiceUnavailable
	}

	response := HealthResponse{
		Status:    status,
		Timestamp: time.Now(),
		Services: map[string]string{
			"database": dbStatus,
		},
	}

	return c.JSON(httpStatus, response)
}
