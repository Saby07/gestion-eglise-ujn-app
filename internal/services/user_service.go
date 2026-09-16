package services

import (
	"context"
	"fmt"
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
	if err := ValidatePassword(in.Password); err != nil {
		return nil, err
	}
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
		if err := s.db.WithContext(ctx).Where(models.Role{Key: rk}).First(&role).Error; err != nil {
			return nil, fmt.Errorf("role %s introuvable: %w", rk, err)
		}
		ur := models.UserRole{UserID: u.ID, RoleID: role.ID}
		if err := s.db.WithContext(ctx).Create(&ur).Error; err != nil {
			return nil, err
		}
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
			if err := tx.Where(models.Role{Key: rk}).First(&role).Error; err != nil {
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
	if err := ValidatePassword(password); err != nil {
		return err
	}
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
	if err := s.db.Where(models.Role{Key: roleKey}).First(&role).Error; err != nil {
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
