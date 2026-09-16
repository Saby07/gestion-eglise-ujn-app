package services

import (
	"context"
	"errors"
	"time"

	"eglise_ujn/internal/models"

	"gorm.io/gorm"
)

type BudgetService struct {
	db *gorm.DB
}

func NewBudgetService(db *gorm.DB) *BudgetService {
	return &BudgetService{db: db}
}

type BudgetWithUsage struct {
	models.Budget
	Spent float64
}

func (s *BudgetService) List(ctx context.Context, year int) ([]models.Budget, error) {
	q := s.db.WithContext(ctx).Preload("Category").Order("year desc, category_id asc")
	if year > 0 {
		q = q.Where("year = ?", year)
	}
	var list []models.Budget
	return list, q.Find(&list).Error
}

func (s *BudgetService) Create(ctx context.Context, categoryID uint, year int, amount float64, by string) (*models.Budget, error) {
	if categoryID == 0 || year <= 0 {
		return nil, errors.New("catégorie et année requis")
	}
	if amount < 0 {
		return nil, errors.New("montant invalide")
	}
	b := models.Budget{
		CategoryID: categoryID,
		Year:       year,
		Amount:     amount,
	}
	b.CreatedBy = by
	if err := s.db.WithContext(ctx).Create(&b).Error; err != nil {
		return nil, err
	}
	return s.GetByID(ctx, b.ID)
}

func (s *BudgetService) Update(ctx context.Context, id uint, amount float64, by string) error {
	if amount < 0 {
		return errors.New("montant invalide")
	}
	return s.db.WithContext(ctx).Model(&models.Budget{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"amount":     amount,
			"updated_by": by,
		}).Error
}

func (s *BudgetService) Delete(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&models.Budget{}, id).Error
}

func (s *BudgetService) GetByID(ctx context.Context, id uint) (*models.Budget, error) {
	var b models.Budget
	if err := s.db.WithContext(ctx).Preload("Category").First(&b, id).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

func (s *BudgetService) GetSpentForCategory(ctx context.Context, categoryID uint, year int) (float64, error) {
	start := time.Date(year, 1, 1, 0, 0, 0, 0, time.Local)
	end := start.AddDate(1, 0, 0)
	var total float64
	err := s.db.WithContext(ctx).Model(&models.Requisition{}).
		Joins("JOIN disbursements ON disbursements.requisition_id = requisitions.id AND disbursements.deleted_at IS NULL").
		Where("requisitions.category_id = ? AND requisitions.status = ?", categoryID, models.ReqStatusCompleted).
		Where("disbursements.disbursed_at >= ? AND disbursements.disbursed_at < ?", start, end).
		Select("COALESCE(SUM(disbursements.amount),0)").Scan(&total).Error
	return total, err
}

func (s *BudgetService) ListWithUsage(ctx context.Context, year int) ([]BudgetWithUsage, error) {
	budgets, err := s.List(ctx, year)
	if err != nil {
		return nil, err
	}
	out := make([]BudgetWithUsage, 0, len(budgets))
	for _, b := range budgets {
		spent, err := s.GetSpentForCategory(ctx, b.CategoryID, b.Year)
		if err != nil {
			return nil, err
		}
		out = append(out, BudgetWithUsage{Budget: b, Spent: spent})
	}
	return out, nil
}
