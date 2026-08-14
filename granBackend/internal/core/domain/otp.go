package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type OTP struct {
	UserID    uuid.UUID      `gorm:"column:user_id;primaryKey"`
	Secret    string         `gorm:"column:secret;not null"`
	Backup    pq.StringArray `gorm:"column:backup;type:text[]"`
	UpdatedAt time.Time      `gorm:"column:updated_at"`
}

func (OTP) TableName() string {
	return "otp"
}
