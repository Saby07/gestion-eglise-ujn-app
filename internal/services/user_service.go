package services

import (
	"context"
	"time"

	"eglise_ujn/internal/models"

	"gorm.io/gorm"
)

type UserService struct {
	db *gorm.DB
}

func NewUserService(db *gorm.DB) *UserService {
	return &UserService{db: db}
}

func (s *UserService) List(ctx context.Context) ([]models.User, error) {
	var users []models.User
	err := s.db.WithContext(ctx).Preload("Roles.Role").Order("last_name, first_name").Find(&users).Error
	return users, err
}

func (s *UserService) GetByID(ctx context.Context, id uint) (*models.User, error) {
	var u models.User
	if err := s.db.WithContext(ctx).Preload("Roles.Role").First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

type CreateUserInput struct {
	FirstName, LastName, Email, Phone, Password string
	Roles                                       []models.RoleKey
}

func (s *UserService) Create(ctx context.Context, in CreateUserInput, by string) (*models.User, error) {
	hash, err := HashPassword(in.Password)
	if err != nil {
		return nil, err
	}
	u := models.User{
		FirstName:    in.FirstName,
		LastName:     in.LastName,
		Email:        in.Email,
		Phone:        in.Phone,
		PasswordHash: hash,
		IsActive:     true,
	}
	u.CreatedBy = by
	if err := s.db.WithContext(ctx).Create(&u).Error; err != nil {
		return nil, err
	}
	for _, rk := range in.Roles {
		var role models.Role
		if err := s.db.Where("key = ?", rk).First(&role).Error; err != nil {
			continue
		}
		ur := models.UserRole{UserID: u.ID, RoleID: role.ID}
		_ = s.db.Create(&ur).Error
	}
	return s.GetByID(ctx, u.ID)
}

type UpdateUserInput struct {
	FirstName, LastName, Email, Phone string
	Roles                             []models.RoleKey
}

func (s *UserService) Update(ctx context.Context, userID uint, in UpdateUserInput, by string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var u models.User
		if err := tx.First(&u, userID).Error; err != nil {
			return err
		}
		u.FirstName = in.FirstName
		u.LastName = in.LastName
		u.Email = in.Email
		u.Phone = in.Phone
		u.UpdatedBy = by
		if err := tx.Save(&u).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&models.UserRole{}).Error; err != nil {
			return err
		}
		for _, rk := range in.Roles {
			var role models.Role
			if err := tx.Where("key = ?", rk).First(&role).Error; err != nil {
				continue
			}
			if err := tx.Create(&models.UserRole{UserID: userID, RoleID: role.ID}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *UserService) UpdatePassword(ctx context.Context, userID uint, password, by string) error {
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).
		Updates(map[string]interface{}{"password_hash": hash, "updated_by": by}).Error
}

func (s *UserService) Delete(ctx context.Context, userID uint, by string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var u models.User
		if err := tx.First(&u, userID).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&models.UserRole{}).Error; err != nil {
			return err
		}
		if err := tx.Model(&u).Update("deleted_by", by).Error; err != nil {
			return err
		}
		return tx.Delete(&u).Error
	})
}

func (s *UserService) ToggleRole(ctx context.Context, userID uint, roleKey models.RoleKey) error {
	var role models.Role
	if err := s.db.Where("key = ?", roleKey).First(&role).Error; err != nil {
		return err
	}
	var existing models.UserRole
	err := s.db.Where("user_id = ? AND role_id = ?", userID, role.ID).First(&existing).Error
	if err == nil {
		return s.db.Delete(&existing).Error
	}
	return s.db.Create(&models.UserRole{UserID: userID, RoleID: role.ID}).Error
}

func (s *UserService) ToggleActive(ctx context.Context, userID uint) error {
	var u models.User
	if err := s.db.First(&u, userID).Error; err != nil {
		return err
	}
	return s.db.Model(&u).Update("is_active", !u.IsActive).Error
}

type ReportService struct {
	db *gorm.DB
}

func NewReportService(db *gorm.DB) *ReportService {
	return &ReportService{db: db}
}

type DisbursementReportRow struct {
	ID            uint
	RequisitionID uint
	Title         string
	Amount        float64
	PaymentMode   models.PaymentMode
	DisbursedAt   time.Time
	DisbursedBy   string
}

func (s *ReportService) Disbursements(ctx context.Context, from, to time.Time) ([]DisbursementReportRow, float64, error) {
	var list []models.Disbursement
	err := s.db.WithContext(ctx).
		Preload("Requisition").
		Preload("DisbursedBy").
		Where("disbursed_at >= ? AND disbursed_at < ?", from, to).
		Order("disbursed_at desc").
		Find(&list).Error
	if err != nil {
		return nil, 0, err
	}
	rows := make([]DisbursementReportRow, 0, len(list))
	var total float64
	for _, d := range list {
		total += d.Amount
		rows = append(rows, DisbursementReportRow{
			ID:            d.ID,
			RequisitionID: d.RequisitionID,
			Title:         d.Requisition.Title,
			Amount:        d.Amount,
			PaymentMode:   d.PaymentMode,
			DisbursedAt:   d.DisbursedAt,
			DisbursedBy:   d.DisbursedBy.FullName(),
		})
	}
	return rows, total, nil
}

type DashboardMetrics struct {
	AccountBalance         float64
	MyRequisitionsCount    int64
	TotalRequisitionsCount int64
	PendingRequisitions    int64
	PendingDisbursement    int64
	DisbursedToday         float64
	DisbursedMonth         float64
	ChartLabels            []string
	ChartValues            []float64
}

type DashboardService struct {
	db         *gorm.DB
	accountSvc *AccountService
	reqSvc     *RequisitionService
}

func NewDashboardService(db *gorm.DB, accountSvc *AccountService, reqSvc *RequisitionService) *DashboardService {
	return &DashboardService{db: db, accountSvc: accountSvc, reqSvc: reqSvc}
}

func (s *DashboardService) Metrics(ctx context.Context, userID uint) (*DashboardMetrics, error) {
	m := &DashboardMetrics{}
	var bal float64
	s.db.Model(&models.ChurchAccount{}).Where("is_active = ?", true).
		Select("COALESCE(SUM(balance),0)").Scan(&bal)
	m.AccountBalance = bal

	s.db.Model(&models.Requisition{}).Where("user_id = ?", userID).Count(&m.MyRequisitionsCount)
	s.db.Model(&models.Requisition{}).Count(&m.TotalRequisitionsCount)

	s.db.Model(&models.Requisition{}).Where("status = ? AND current_step != ?",
		models.ReqStatusOpen, models.StepPendingDisbursement).Count(&m.PendingRequisitions)
	s.db.Model(&models.Requisition{}).Where("current_step = ?", models.StepPendingDisbursement).Count(&m.PendingDisbursement)

	now := time.Now()
	startDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	startMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	m.DisbursedToday, _ = s.reqSvc.TotalDisbursed(ctx, startDay, startDay.AddDate(0, 0, 1))
	m.DisbursedMonth, _ = s.reqSvc.TotalDisbursed(ctx, startMonth, startMonth.AddDate(0, 1, 0))

	// 7 derniers jours pour graphique
	m.ChartLabels = make([]string, 7)
	m.ChartValues = make([]float64, 7)
	for i := 6; i >= 0; i-- {
		d := startDay.AddDate(0, 0, -i)
		end := d.AddDate(0, 0, 1)
		m.ChartLabels[6-i] = d.Format("02/01")
		m.ChartValues[6-i], _ = s.reqSvc.TotalDisbursed(ctx, d, end)
	}
	return m, nil
}
