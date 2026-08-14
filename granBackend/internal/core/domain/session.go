package domain

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID        uuid.UUID `gorm:"column:id;primaryKey"`
	UserID    uuid.UUID `gorm:"column:user_id;not null"`
	Hash      string    `gorm:"column:hash;not null"`
	Device    string    `gorm:"column:device"`
	IP        string    `gorm:"column:ip;type:inet"`
	Exp       time.Time `gorm:"column:exp;not null"`
	Revoked   bool      `gorm:"column:revoked;default:false"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (Session) TableName() string {
	return "sessions"
}
