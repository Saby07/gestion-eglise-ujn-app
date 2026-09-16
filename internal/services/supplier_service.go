package services

import (
	"context"
	"errors"

	"eglise_ujn/internal/models"

	"gorm.io/gorm"
)

type SupplierService struct {
	db *gorm.DB
}

func NewSupplierService(db *gorm.DB) *SupplierService {
	return &SupplierService{db: db}
}

func (s *SupplierService) List(ctx context.Context, activeOnly bool) ([]models.Supplier, error) {
	q := s.db.WithContext(ctx).Order("name asc")
	if activeOnly {
		q = q.Where("is_active = ?", true)
	}
	var list []models.Supplier
	return list, q.Find(&list).Error
}

func (s *SupplierService) GetByID(ctx context.Context, id uint) (*models.Supplier, error) {
	var sup models.Supplier
	if err := s.db.WithContext(ctx).First(&sup, id).Error; err != nil {
		return nil, err
	}
	return &sup, nil
}

func (s *SupplierService) Create(ctx context.Context, name, phone, address, email, by string) (*models.Supplier, error) {
	if name == "" {
		return nil, errors.New("nom du fournisseur requis")
	}
	sup := models.Supplier{
		Name:     name,
		Phone:    phone,
		Address:  address,
		Email:    email,
		IsActive: true,
	}
	sup.CreatedBy = by
	if err := s.db.WithContext(ctx).Create(&sup).Error; err != nil {
		return nil, err
	}
	return &sup, nil
}

func (s *SupplierService) Update(ctx context.Context, id uint, name, phone, address, email, by string) error {
	if name == "" {
		return errors.New("nom du fournisseur requis")
	}
	return s.db.WithContext(ctx).Model(&models.Supplier{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"name":       name,
			"phone":      phone,
			"address":    address,
			"email":      email,
			"updated_by": by,
		}).Error
}

func (s *SupplierService) Delete(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&models.Supplier{}, id).Error
}

func (s *SupplierService) Toggle(ctx context.Context, id uint, by string) error {
	var sup models.Supplier
	if err := s.db.WithContext(ctx).First(&sup, id).Error; err != nil {
		return err
	}
	return s.db.WithContext(ctx).Model(&sup).Updates(map[string]interface{}{
		"is_active":  !sup.IsActive,
		"updated_by": by,
	}).Error
}
