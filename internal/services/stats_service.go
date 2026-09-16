package services

import (
	"context"
	"time"

	"eglise_ujn/internal/models"

	"gorm.io/gorm"
)

type StatsService struct {
	db *gorm.DB
}

func NewStatsService(db *gorm.DB) *StatsService {
	return &StatsService{db: db}
}

type CategoryStat struct {
	CategoryID   uint
	CategoryName string
	Total        float64
	Count        int64
}

type MonthlyTrend struct {
	Month string
	Total float64
}

type SupplierStat struct {
	SupplierName string
	Total        float64
	Count        int64
}

type AdvancedStats struct {
	ByCategory         []CategoryStat
	AvgValidationDays  float64
	MonthlyTrend       []MonthlyTrend
	TopSuppliers       []SupplierStat
}

func (s *StatsService) AdvancedStats(ctx context.Context, year int) (*AdvancedStats, error) {
	if year <= 0 {
		year = time.Now().Year()
	}
	start := time.Date(year, 1, 1, 0, 0, 0, 0, time.Local)
	end := start.AddDate(1, 0, 0)

	stats := &AdvancedStats{}

	type catRow struct {
		CategoryID   uint
		CategoryName string
		Total        float64
		Count        int64
	}
	var catRows []catRow
	err := s.db.WithContext(ctx).Model(&models.Disbursement{}).
		Select("requisitions.category_id as category_id, COALESCE(expense_categories.name, 'Sans catégorie') as category_name, COALESCE(SUM(disbursements.amount),0) as total, COUNT(*) as count").
		Joins("JOIN requisitions ON requisitions.id = disbursements.requisition_id AND requisitions.deleted_at IS NULL").
		Joins("LEFT JOIN expense_categories ON expense_categories.id = requisitions.category_id AND expense_categories.deleted_at IS NULL").
		Where("disbursements.disbursed_at >= ? AND disbursements.disbursed_at < ?", start, end).
		Group("requisitions.category_id, expense_categories.name").
		Scan(&catRows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range catRows {
		stats.ByCategory = append(stats.ByCategory, CategoryStat{
			CategoryID:   r.CategoryID,
			CategoryName: r.CategoryName,
			Total:        r.Total,
			Count:        r.Count,
		})
	}

	type valRow struct {
		CreatedAt   time.Time
		ValidatedAt time.Time
	}
	var valRows []valRow
	err = s.db.WithContext(ctx).Model(&models.Requisition{}).
		Select("requisitions.created_at, MIN(requisition_validations.validated_at) as validated_at").
		Joins("JOIN requisition_validations ON requisition_validations.requisition_id = requisitions.id AND requisition_validations.deleted_at IS NULL").
		Where("requisitions.status = ? AND requisitions.created_at >= ? AND requisitions.created_at < ?",
			models.ReqStatusCompleted, start, end).
		Group("requisitions.id, requisitions.created_at").
		Scan(&valRows).Error
	if err != nil {
		return nil, err
	}
	if len(valRows) > 0 {
		var sum float64
		for _, r := range valRows {
			sum += r.ValidatedAt.Sub(r.CreatedAt).Hours() / 24
		}
		stats.AvgValidationDays = sum / float64(len(valRows))
	}

	for m := 1; m <= 12; m++ {
		mStart := time.Date(year, time.Month(m), 1, 0, 0, 0, 0, time.Local)
		mEnd := mStart.AddDate(0, 1, 0)
		var total float64
		if err := s.db.WithContext(ctx).Model(&models.Disbursement{}).
			Where("disbursed_at >= ? AND disbursed_at < ?", mStart, mEnd).
			Select("COALESCE(SUM(amount),0)").Scan(&total).Error; err != nil {
			return nil, err
		}
		stats.MonthlyTrend = append(stats.MonthlyTrend, MonthlyTrend{
			Month: mStart.Format("01/2006"),
			Total: total,
		})
	}

	type supRow struct {
		SupplierName string
		Total        float64
		Count        int64
	}
	var supRows []supRow
	err = s.db.WithContext(ctx).Model(&models.Disbursement{}).
		Select(`COALESCE(NULLIF(requisitions.supplier_name, ''), suppliers.name, 'Inconnu') as supplier_name,
			COALESCE(SUM(disbursements.amount),0) as total, COUNT(*) as count`).
		Joins("JOIN requisitions ON requisitions.id = disbursements.requisition_id AND requisitions.deleted_at IS NULL").
		Joins("LEFT JOIN suppliers ON suppliers.id = requisitions.supplier_id AND suppliers.deleted_at IS NULL").
		Where("disbursements.disbursed_at >= ? AND disbursements.disbursed_at < ?", start, end).
		Group("supplier_name").
		Order("total desc").
		Limit(10).
		Scan(&supRows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range supRows {
		stats.TopSuppliers = append(stats.TopSuppliers, SupplierStat{
			SupplierName: r.SupplierName,
			Total:        r.Total,
			Count:        r.Count,
		})
	}

	return stats, nil
}
