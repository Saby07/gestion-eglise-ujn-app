package services

import (
	"context"
	"time"

	"eglise_ujn/internal/models"

	"gorm.io/gorm"
)

type DashboardReqRow struct {
	ID        uint
	Title     string
	Author    string
	Amount    float64
	Date      string
	Step      string
	StepLabel string
	Badge     string
}

type DashboardMetrics struct {
	Role models.RoleKey

	AccountBalance         float64
	PendingDisbursementAmt float64
	MyRequisitionsCount    int64
	MyOpenCount            int64
	MyCompletedCount       int64
	MyCancelledCount       int64
	TotalRequisitionsCount int64
	PendingRequisitions    int64
	PendingAccountant      int64
	PendingAdmin           int64
	PendingSuperAdmin      int64
	PendingDisbursement    int64
	CompletedCount         int64
	UsersCount             int64
	ActiveUsersCount       int64
	AccountsCount          int64
	DisbursedToday         float64
	DisbursedMonth         float64
	UnreadNotifications    int64
	CanDisburse            bool

	// Chart: décaissements 7 jours
	ChartLabels []string
	ChartValues []float64

	// Chart: répartition par statut / étape
	StatusLabels []string
	StatusValues []int64

	// Chart: dépenses par catégorie (année en cours)
	CategoryLabels []string
	CategoryValues []float64

	ActionQueue []DashboardReqRow
	RecentList  []DashboardReqRow
}

type DashboardService struct {
	db         *gorm.DB
	accountSvc *AccountService
	reqSvc     *RequisitionService
}

func NewDashboardService(db *gorm.DB, accountSvc *AccountService, reqSvc *RequisitionService) *DashboardService {
	return &DashboardService{db: db, accountSvc: accountSvc, reqSvc: reqSvc}
}

func (s *DashboardService) Metrics(ctx context.Context, user *models.User) (*DashboardMetrics, error) {
	m := &DashboardMetrics{}
	if user != nil {
		m.Role = user.PrimaryRole()
		m.CanDisburse = user.HasRole(models.RoleCashier) || (user.HasRole(models.RoleAccountant) && user.CanDisburse)
	}

	s.loadGlobalCounts(ctx, m, user)
	s.loadCharts(ctx, m)
	s.loadRoleLists(ctx, m, user)
	return m, nil
}

func (s *DashboardService) loadGlobalCounts(ctx context.Context, m *DashboardMetrics, user *models.User) {
	db := s.db.WithContext(ctx)
	var bal float64
	db.Model(&models.ChurchAccount{}).Where("is_active = ?", true).
		Select("COALESCE(SUM(balance),0)").Scan(&bal)
	m.AccountBalance = bal
	db.Model(&models.ChurchAccount{}).Where("is_active = ?", true).Count(&m.AccountsCount)

	userID := uint(0)
	if user != nil {
		userID = user.ID
	}
	db.Model(&models.Requisition{}).Where("user_id = ?", userID).Count(&m.MyRequisitionsCount)
	db.Model(&models.Requisition{}).Where("user_id = ? AND status = ?", userID, models.ReqStatusOpen).Count(&m.MyOpenCount)
	db.Model(&models.Requisition{}).Where("user_id = ? AND status = ?", userID, models.ReqStatusCompleted).Count(&m.MyCompletedCount)
	db.Model(&models.Requisition{}).Where("user_id = ? AND status = ?", userID, models.ReqStatusCancelled).Count(&m.MyCancelledCount)
	db.Model(&models.Requisition{}).Count(&m.TotalRequisitionsCount)
	db.Model(&models.Requisition{}).Where("status = ?", models.ReqStatusCompleted).Count(&m.CompletedCount)

	db.Model(&models.Requisition{}).Where("status = ? AND current_step = ?", models.ReqStatusOpen, models.StepPendingAccountant).Count(&m.PendingAccountant)
	db.Model(&models.Requisition{}).Where("status = ? AND current_step = ?", models.ReqStatusOpen, models.StepPendingAdmin).Count(&m.PendingAdmin)
	db.Model(&models.Requisition{}).Where("status = ? AND current_step = ?", models.ReqStatusOpen, models.StepPendingSuperAdmin).Count(&m.PendingSuperAdmin)
	db.Model(&models.Requisition{}).Where("status = ? AND current_step = ?", models.ReqStatusOpen, models.StepPendingDisbursement).Count(&m.PendingDisbursement)
	db.Model(&models.Requisition{}).Where("status = ? AND current_step = ?", models.ReqStatusOpen, models.StepPendingDisbursement).
		Select("COALESCE(SUM(total_amount),0)").Scan(&m.PendingDisbursementAmt)

	switch {
	case user != nil && user.HasRole(models.RoleAccountant) && !user.HasRole(models.RoleAdmin) && !user.HasRole(models.RoleSuperAdmin):
		m.PendingRequisitions = m.PendingAccountant
	case user != nil && user.HasRole(models.RoleAdmin) && !user.HasRole(models.RoleSuperAdmin):
		m.PendingRequisitions = m.PendingAdmin
	case user != nil && user.HasRole(models.RoleSuperAdmin):
		m.PendingRequisitions = m.PendingAccountant + m.PendingAdmin + m.PendingSuperAdmin
	case user != nil && user.HasRole(models.RoleStaff):
		m.PendingRequisitions = m.MyOpenCount
	default:
		m.PendingRequisitions = m.PendingAccountant + m.PendingAdmin + m.PendingSuperAdmin
	}

	db.Model(&models.User{}).Where("deleted_at IS NULL").Count(&m.UsersCount)
	db.Model(&models.User{}).Where("is_active = ? AND deleted_at IS NULL", true).Count(&m.ActiveUsersCount)
	if user != nil {
		db.Model(&models.Notification{}).Where("user_id = ? AND read_at IS NULL", user.ID).Count(&m.UnreadNotifications)
	}

	now := time.Now()
	startDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	startMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	m.DisbursedToday, _ = s.reqSvc.TotalDisbursed(ctx, startDay, startDay.AddDate(0, 0, 1))
	m.DisbursedMonth, _ = s.reqSvc.TotalDisbursed(ctx, startMonth, startMonth.AddDate(0, 1, 0))
}

func (s *DashboardService) loadCharts(ctx context.Context, m *DashboardMetrics) {
	now := time.Now()
	startDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	m.ChartLabels = make([]string, 7)
	m.ChartValues = make([]float64, 7)
	for i := 6; i >= 0; i-- {
		d := startDay.AddDate(0, 0, -i)
		end := d.AddDate(0, 0, 1)
		m.ChartLabels[6-i] = d.Format("02/01")
		m.ChartValues[6-i], _ = s.reqSvc.TotalDisbursed(ctx, d, end)
	}

	m.StatusLabels = []string{"Comptable", "Admin", "Super Admin", "Décaissement", "Terminées", "Annulées"}
	var cancelled int64
	s.db.WithContext(ctx).Model(&models.Requisition{}).Where("status = ?", models.ReqStatusCancelled).Count(&cancelled)
	m.StatusValues = []int64{
		m.PendingAccountant,
		m.PendingAdmin,
		m.PendingSuperAdmin,
		m.PendingDisbursement,
		m.CompletedCount,
		cancelled,
	}

	type catRow struct {
		Name  string
		Total float64
	}
	var cats []catRow
	yearStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
	s.db.WithContext(ctx).Raw(`
		SELECT COALESCE(c.name, 'Sans catégorie') AS name, COALESCE(SUM(r.total_amount), 0) AS total
		FROM requisitions r
		LEFT JOIN expense_categories c ON c.id = r.category_id AND c.deleted_at IS NULL
		WHERE r.deleted_at IS NULL AND r.status = ? AND r.created_at >= ?
		GROUP BY COALESCE(c.name, 'Sans catégorie')
		ORDER BY total DESC
		LIMIT 6
	`, models.ReqStatusCompleted, yearStart).Scan(&cats)
	for _, c := range cats {
		m.CategoryLabels = append(m.CategoryLabels, c.Name)
		m.CategoryValues = append(m.CategoryValues, c.Total)
	}
}

func (s *DashboardService) loadRoleLists(ctx context.Context, m *DashboardMetrics, user *models.User) {
	if user == nil {
		return
	}
	switch user.PrimaryRole() {
	case models.RoleSuperAdmin:
		m.ActionQueue = s.fetchReqs(ctx, 8, "status = ? AND current_step IN ?", models.ReqStatusOpen,
			[]models.RequisitionStep{models.StepPendingAccountant, models.StepPendingAdmin, models.StepPendingSuperAdmin, models.StepPendingDisbursement})
		m.RecentList = s.fetchReqs(ctx, 8, "", nil)
	case models.RoleAdmin:
		m.ActionQueue = s.fetchReqs(ctx, 8, "status = ? AND current_step = ?", models.ReqStatusOpen, models.StepPendingAdmin)
		m.RecentList = s.fetchReqs(ctx, 8, "", nil)
	case models.RoleAccountant:
		m.ActionQueue = s.fetchReqs(ctx, 8, "status = ? AND current_step = ?", models.ReqStatusOpen, models.StepPendingAccountant)
		if m.CanDisburse {
			m.RecentList = s.fetchReqs(ctx, 8, "status = ? AND current_step = ?", models.ReqStatusOpen, models.StepPendingDisbursement)
		} else {
			m.RecentList = s.fetchReqs(ctx, 8, "", nil)
		}
	case models.RoleCashier:
		m.ActionQueue = s.fetchReqs(ctx, 10, "status = ? AND current_step = ?", models.ReqStatusOpen, models.StepPendingDisbursement)
		m.RecentList = s.fetchDisbursed(ctx, 8)
	case models.RoleStaff:
		m.ActionQueue = s.fetchReqs(ctx, 8, "user_id = ? AND status = ?", user.ID, models.ReqStatusOpen)
		m.RecentList = s.fetchReqs(ctx, 8, "user_id = ?", user.ID)
	}
}

func (s *DashboardService) fetchReqs(ctx context.Context, limit int, where string, args ...interface{}) []DashboardReqRow {
	q := s.db.WithContext(ctx).Preload("User").Order("created_at desc").Limit(limit)
	if where != "" {
		q = q.Where(where, args...)
	}
	var list []models.Requisition
	if err := q.Find(&list).Error; err != nil {
		return nil
	}
	return mapDashboardReqs(list)
}

func (s *DashboardService) fetchDisbursed(ctx context.Context, limit int) []DashboardReqRow {
	var list []models.Requisition
	err := s.db.WithContext(ctx).Preload("User").Preload("Disbursement").
		Where("status = ?", models.ReqStatusCompleted).
		Order("updated_at desc").Limit(limit).Find(&list).Error
	if err != nil {
		return nil
	}
	return mapDashboardReqs(list)
}

func mapDashboardReqs(list []models.Requisition) []DashboardReqRow {
	out := make([]DashboardReqRow, 0, len(list))
	for _, r := range list {
		out = append(out, DashboardReqRow{
			ID:        r.ID,
			Title:     r.Title,
			Author:    r.User.FullName(),
			Amount:    r.TotalAmount,
			Date:      r.RequisitionDate.Format("02/01/2006"),
			Step:      string(r.CurrentStep),
			StepLabel: stepLabel(r.CurrentStep),
			Badge:     statusBadge(r.Status, r.CurrentStep),
		})
	}
	return out
}

func stepLabel(step models.RequisitionStep) string {
	switch step {
	case models.StepPendingAccountant:
		return "Comptable"
	case models.StepPendingAdmin:
		return "Admin"
	case models.StepPendingSuperAdmin:
		return "Super Admin"
	case models.StepPendingDisbursement:
		return "À décaisser"
	case models.StepDisbursed:
		return "Décaissée"
	case models.StepCancelled:
		return "Annulée"
	default:
		return string(step)
	}
}

func statusBadge(status models.RequisitionStatus, step models.RequisitionStep) string {
	if status == models.ReqStatusCancelled {
		return "bg-danger"
	}
	if status == models.ReqStatusCompleted {
		return "bg-success"
	}
	if step == models.StepPendingDisbursement {
		return "bg-info"
	}
	return "bg-warning"
}
