package services

import (
	"context"
	"errors"

	"eglise_ujn/internal/models"

	"gorm.io/gorm"
)

type CategoryService struct {
	db *gorm.DB
}

func NewCategoryService(db *gorm.DB) *CategoryService {
	return &CategoryService{db: db}
}

func (s *CategoryService) List(ctx context.Context, activeOnly bool) ([]models.ExpenseCategory, error) {
	q := s.db.WithContext(ctx).Order("name asc")
	if activeOnly {
		q = q.Where("is_active = ?", true)
	}
	var list []models.ExpenseCategory
	return list, q.Find(&list).Error
}

func (s *CategoryService) Create(ctx context.Context, name, description, by string) (*models.ExpenseCategory, error) {
	if name == "" {
		return nil, errors.New("nom de catégorie requis")
	}
	c := models.ExpenseCategory{
		Name:        name,
		Description: description,
		IsActive:    true,
	}
	c.CreatedBy = by
	if err := s.db.WithContext(ctx).Create(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *CategoryService) Update(ctx context.Context, id uint, name, description, by string) error {
	if name == "" {
		return errors.New("nom de catégorie requis")
	}
	return s.db.WithContext(ctx).Model(&models.ExpenseCategory{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"name":        name,
			"description": description,
			"updated_by":  by,
		}).Error
}

func (s *CategoryService) Delete(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&models.ExpenseCategory{}, id).Error
}

func (s *CategoryService) Toggle(ctx context.Context, id uint, by string) error {
	var c models.ExpenseCategory
	if err := s.db.WithContext(ctx).First(&c, id).Error; err != nil {
		return err
	}
	return s.db.WithContext(ctx).Model(&c).Updates(map[string]interface{}{
		"is_active":  !c.IsActive,
		"updated_by": by,
	}).Error
}
