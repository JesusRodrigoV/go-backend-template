package domain

import (
	"time"

	"github.com/google/uuid"
)

type Log struct {
	ID        int64          `gorm:"column:id;primaryKey;autoIncrement"`
	UserID    uuid.UUID      `gorm:"column:user_id"`
	Event     string         `gorm:"column:event;not null"`
	Meta      map[string]any `gorm:"column:meta;serializer:json"`
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime"`
}

func (Log) TableName() string {
	return "logs"
}
