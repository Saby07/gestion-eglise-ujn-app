package seed

import (
	"context"
	"errors"
	"fmt"

	"eglise_ujn/internal/models"
	"eglise_ujn/internal/services"

	"gorm.io/gorm"
)

type Config struct {
	AdminEmail    string
	AdminPassword string
}

func Run(db *gorm.DB, cfg Config) error {
	if db == nil {
		return errors.New("db nil")
	}
	for _, rk := range models.AllRoles {
		var role models.Role
		err := db.Where("key = ?", rk).First(&role).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			role = models.Role{Key: rk, Name: rk.Label(), Description: rk.Label()}
			if err := db.Create(&role).Error; err != nil {
				return fmt.Errorf("role %s: %w", rk, err)
			}
		}
	}

	if cfg.AdminEmail == "" {
		cfg.AdminEmail = "superadmin@eglise-ujn.local"
	}
	if cfg.AdminPassword == "" {
		cfg.AdminPassword = "ChangeMe123!"
	}

	var count int64
	db.Model(&models.User{}).Count(&count)
	if count > 0 {
		return nil
	}

	userSvc := services.NewUserService(db)
	users := []struct {
		email, first, last, pass string
		roles                    []models.RoleKey
		canDisburse              bool
	}{
		{cfg.AdminEmail, "Super", "Admin", cfg.AdminPassword, []models.RoleKey{models.RoleSuperAdmin}, false},
		{"admin@eglise-ujn.local", "Jean", "Admin", "ChangeMe123!", []models.RoleKey{models.RoleAdmin}, false},
		{"comptable@eglise-ujn.local", "Marie", "Comptable", "ChangeMe123!", []models.RoleKey{models.RoleAccountant}, false},
		{"caissier@eglise-ujn.local", "Paul", "Caissier", "ChangeMe123!", []models.RoleKey{models.RoleCashier}, false},
		{"staff@eglise-ujn.local", "Alice", "Staff", "ChangeMe123!", []models.RoleKey{models.RoleStaff}, false},
	}
	for _, u := range users {
		_, err := userSvc.Create(context.Background(), services.CreateUserInput{
			FirstName: u.first,
			LastName:  u.last,
			Email:     u.email,
			Password:  u.pass,
			Roles:     u.roles,
		}, "seed")
		if err != nil {
			return err
		}
	}

	accSvc := services.NewAccountService(db)
	acc, err := accSvc.Create(context.Background(), "Compte principal", "Compte de fonctionnement", "USD", "seed")
	if err != nil {
		return err
	}
	var super models.User
	if err := db.Preload("Roles.Role").Where("email = ?", cfg.AdminEmail).First(&super).Error; err == nil {
		_ = accSvc.Fund(context.Background(), acc.ID, 0, "Approvisionnement initial", super.ID, "seed")
	}
	return nil
}
