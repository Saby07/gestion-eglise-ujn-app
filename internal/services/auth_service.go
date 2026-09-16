package services

import (
	"context"
	"errors"

	"eglise_ujn/internal/config"
	"eglise_ujn/internal/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	db *gorm.DB
}

func NewAuthService(db *gorm.DB) *AuthService {
	return &AuthService{db: db}
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*models.User, string, error) {
	var user models.User
	if err := s.db.WithContext(ctx).
		Preload("Roles.Role").
		Where("email = ? AND is_active = ?", email, true).
		First(&user).Error; err != nil {
		return nil, "", errors.New("identifiants invalides")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, "", errors.New("identifiants invalides")
	}
	cfg, err := config.LoadJWTConfig()
	if err != nil {
		return nil, "", err
	}
	token, err := config.NewToken(cfg, user.ID, 0)
	if err != nil {
		return nil, "", err
	}
	return &user, token, nil
}

func (s *AuthService) GetUserByID(ctx context.Context, id uint) (*models.User, error) {
	var user models.User
	if err := s.db.WithContext(ctx).Preload("Roles.Role").First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *AuthService) VerifyPassword(ctx context.Context, userID uint, password string) error {
	var user models.User
	if err := s.db.WithContext(ctx).First(&user, userID).Error; err != nil {
		return errors.New("utilisateur introuvable")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return errors.New("mot de passe incorrect")
	}
	return nil
}

func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}

func ValidatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("mot de passe trop court (minimum 8 caractères)")
	}
	return nil
}
