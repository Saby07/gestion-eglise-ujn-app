package seed

import (
	"context"
	"errors"
	"fmt"
	"log"

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

	if err := seedRoles(db); err != nil {
		return err
	}

	if cfg.AdminEmail == "" {
		cfg.AdminEmail = "superadmin@eglise-ujn.local"
	}
	if cfg.AdminPassword == "" {
		cfg.AdminPassword = "ChangeMe123!"
	}

	var count int64
	if err := db.Model(&models.User{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ensureUserRole(db, cfg.AdminEmail, models.RoleSuperAdmin)
	}

	userSvc := services.NewUserService(db)
	users := []struct {
		email, first, last, pass string
		roles                    []models.RoleKey
	}{
		{cfg.AdminEmail, "Super", "Admin", cfg.AdminPassword, []models.RoleKey{models.RoleSuperAdmin}},
		{"admin@eglise-ujn.local", "Jean", "Admin", "ChangeMe123!", []models.RoleKey{models.RoleAdmin}},
		{"comptable@eglise-ujn.local", "Marie", "Comptable", "ChangeMe123!", []models.RoleKey{models.RoleAccountant}},
		{"caissier@eglise-ujn.local", "Paul", "Caissier", "ChangeMe123!", []models.RoleKey{models.RoleCashier}},
		{"staff@eglise-ujn.local", "Alice", "Staff", "ChangeMe123!", []models.RoleKey{models.RoleStaff}},
	}
	for _, u := range users {
		created, err := userSvc.Create(context.Background(), services.CreateUserInput{
			FirstName: u.first,
			LastName:  u.last,
			Email:     u.email,
			Password:  u.pass,
			Roles:     u.roles,
		}, "seed")
		if err != nil {
			return err
		}
		if len(created.Roles) != len(u.roles) {
			return fmt.Errorf("utilisateur %s: rôles attendus %d, obtenus %d", u.email, len(u.roles), len(created.Roles))
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

func seedRoles(db *gorm.DB) error {
	for _, rk := range models.AllRoles {
		var role models.Role
		result := db.Where(models.Role{Key: rk}).Attrs(models.Role{
			Name:        rk.Label(),
			Description: rk.Label(),
		}).FirstOrCreate(&role)
		if result.Error != nil {
			return fmt.Errorf("role %s: %w", rk, result.Error)
		}
	}

	var count int64
	if err := db.Model(&models.Role{}).Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(models.AllRoles)) {
		return fmt.Errorf("roles incomplets: %d/%d enregistrés", count, len(models.AllRoles))
	}

	log.Printf("[seed] %d rôles disponibles", count)
	return nil
}

func ensureUserRole(db *gorm.DB, email string, roleKey models.RoleKey) error {
	var user models.User
	if err := db.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	var role models.Role
	if err := db.Where(models.Role{Key: roleKey}).First(&role).Error; err != nil {
		return fmt.Errorf("role %s introuvable: %w", roleKey, err)
	}

	var link models.UserRole
	result := db.Where(models.UserRole{UserID: user.ID, RoleID: role.ID}).FirstOrCreate(&link)
	if result.Error != nil {
		return fmt.Errorf("user_role %s/%s: %w", email, roleKey, result.Error)
	}
	if result.RowsAffected > 0 {
		log.Printf("[seed] rôle %s attribué à %s", roleKey, email)
	}
	return nil
}
