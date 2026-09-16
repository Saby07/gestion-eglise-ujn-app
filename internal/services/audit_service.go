package services

import (
	"context"

	"eglise_ujn/internal/models"

	"gorm.io/gorm"
)

type AuditService struct {
	db *gorm.DB
}

func NewAuditService(db *gorm.DB) *AuditService {
	return &AuditService{db: db}
}

func (s *AuditService) Log(ctx context.Context, userID uint, action, entity string, entityID uint, summary, ip string) error {
	entry := models.AuditLog{
		UserID:   userID,
		Action:   action,
		Entity:   entity,
		EntityID: entityID,
		Summary:  summary,
		IP:       ip,
	}
	return s.db.WithContext(ctx).Create(&entry).Error
}

func (s *AuditService) List(ctx context.Context, limit int) ([]models.AuditLog, error) {
	if limit <= 0 {
		limit = 50
	}
	var list []models.AuditLog
	err := s.db.WithContext(ctx).
		Preload("User").
		Order("created_at desc").
		Limit(limit).
		Find(&list).Error
	return list, err
}
