package domain

import (
	"time"

	"github.com/google/uuid"
)

type Social struct {
	ID        int            `gorm:"column:id;primaryKey;autoIncrement"`
	UserID    uuid.UUID      `gorm:"column:user_id"`
	Provider  string         `gorm:"column:provider"`
	ProvID    string         `gorm:"column:prov_id"`
	Raw       map[string]any `gorm:"column:raw;serializer:json"`
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime"`
}

func (Social) TableName() string {
	return "socials"
}
