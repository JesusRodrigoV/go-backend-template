package repository

import (
	"github.com/JesusRodrigoV/granBackend/internal/core/domain"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type UserRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Save(user *domain.User) error {
	return r.db.Create(user).Error
}

func (r *UserRepo) GetByEmail(email string) (*domain.User, error) {
	var user domain.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepo) SaveSession(session *domain.Session) error {
	return r.db.Create(session).Error
}

func (r *UserRepo) GetSessionByHash(hash string) (*domain.Session, error) {
	var session domain.Session
	err := r.db.Where("hash = ?", hash).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *UserRepo) UpdateSession(session *domain.Session) error {
	return r.db.Save(session).Error
}

func (r *UserRepo) RevokeSession(hash string) error {
	return r.db.Model(&domain.Session{}).
		Where("hash = ?", hash).
		Update("revoked", true).Error
}

func (r *UserRepo) GetSessionsByUserID(userID string) ([]domain.Session, error) {
	var sessions []domain.Session
	err := r.db.Where("user_id = ? AND revoked = ? AND exp > NOW()", userID, false).
		Find(&sessions).Error
	return sessions, err
}

func (r *UserRepo) RevokeAllSessions(userID string) error {
	return r.db.Model(&domain.Session{}).
		Where("user_id = ?", userID).
		Update("revoked", true).Error
}

func (r *UserRepo) SaveOTP(otp *domain.OTP) error {
	return r.db.Save(otp).Error
}

func (r *UserRepo) GetOTP(userID string) (*domain.OTP, error) {
	var otp domain.OTP
	err := r.db.Where("user_id = ?", userID).First(&otp).Error
	return &otp, err
}

func (r *UserRepo) DeleteOTP(userID string) error {
	return r.db.Delete(&domain.OTP{}, "user_id = ?", userID).Error
}

func (r *UserRepo) SetMFAStatus(userID string, status bool) error {
	return r.db.Model(&domain.User{}).Where("id = ?", userID).Update("mfa", status).Error
}

func (r *UserRepo) UpdateBackupCodes(userID string, codes []string) error {
	return r.db.Model(&domain.OTP{}).Where("user_id = ?", userID).
		Update("backup", pq.StringArray(codes)).Error
}

func (r *UserRepo) UpdatePassword(userID string, hash string) error {
	return r.db.Model(&domain.User{}).
		Where("id = ?", userID).
		Update("pwd", hash).Error
}

func (r *UserRepo) SaveLog(log *domain.Log) error {
	return r.db.Create(log).Error
}

func (r *UserRepo) GetLogsByUserID(userID string) ([]domain.Log, error) {
	var logs []domain.Log
	err := r.db.Where("user_id = ?", userID).
		Order("created_at desc").
		Limit(20).
		Find(&logs).Error
	return logs, err
}

func (r *UserRepo) FindSocial(provider, provID string) (*domain.Social, error) {
	var social domain.Social
	err := r.db.Where("provider = ? AND prov_id = ?", provider, provID).First(&social).Error
	return &social, err
}

func (r *UserRepo) SaveSocial(social *domain.Social) error {
	return r.db.Create(social).Error
}

func (r *UserRepo) VerifyUser(userID string) error {
	return r.db.Model(&domain.User{}).Where("id = ?", userID).Update("verified", true).Error
}
