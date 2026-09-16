package seed

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"eglise_ujn/internal/models"
	"eglise_ujn/internal/services"

	"gorm.io/gorm"
)

type Config struct {
	AdminEmail    string
	AdminPassword string
	GinMode       string
}

func Run(db *gorm.DB, cfg Config) error {
	if db == nil {
		return errors.New("db nil")
	}

	if err := seedRoles(db); err != nil {
		return err
	}

	if err := validateSeedConfig(cfg); err != nil {
		return err
	}

	if cfg.AdminEmail == "" {
		cfg.AdminEmail = "superadmin@eglise-ujn.local"
	}

	var count int64
	if err := db.Model(&models.User{}).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		if err := seedSuperAdmin(db, cfg); err != nil {
			return err
		}
	} else if err := ensureUserRole(db, cfg.AdminEmail, models.RoleSuperAdmin); err != nil {
		return err
	}

	return seedDefaults(context.Background(), db)
}

func seedSuperAdmin(db *gorm.DB, cfg Config) error {
	userSvc := services.NewUserService(db)
	created, err := userSvc.Create(context.Background(), services.CreateUserInput{
		FirstName: "Super",
		LastName:  "Admin",
		Email:     cfg.AdminEmail,
		Password:  cfg.AdminPassword,
		Roles:     []models.RoleKey{models.RoleSuperAdmin},
	}, "seed")
	if err != nil {
		return err
	}
	if len(created.Roles) != 1 {
		return fmt.Errorf("utilisateur %s: rôles attendus 1, obtenus %d", cfg.AdminEmail, len(created.Roles))
	}
	log.Printf("[seed] super admin créé: %s", cfg.AdminEmail)
	return nil
}

func seedDefaults(ctx context.Context, db *gorm.DB) error {
	settingsSvc := services.NewSettingsService(db)
	defaults := map[string]string{
		services.SettingChurchName:   "Église UJN",
		services.SettingTheme:        "light",
		services.SettingReminderDays: "3",
	}
	for k, v := range defaults {
		cur, err := settingsSvc.Get(ctx, k)
		if err != nil {
			return err
		}
		if cur == "" {
			if err := settingsSvc.Set(ctx, k, v); err != nil {
				return err
			}
		}
	}

	catSvc := services.NewCategoryService(db)
	for _, name := range []string{"Culte", "Entretien", "Évangélisation", "Social", "Administration"} {
		var count int64
		if err := db.Model(&models.ExpenseCategory{}).Where("name = ?", name).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if _, err := catSvc.Create(ctx, name, name, "seed"); err != nil {
				return err
			}
		}
	}
	log.Printf("[seed] paramètres et catégories par défaut prêts")
	return nil
}

func validateSeedConfig(cfg Config) error {
	if strings.TrimSpace(cfg.AdminPassword) == "" {
		return errors.New("ADMIN_PASSWORD est requis dans .env pour le seed")
	}
	if strings.EqualFold(cfg.GinMode, "release") {
		if cfg.AdminPassword == "ChangeMe123!" {
			return errors.New("ADMIN_PASSWORD doit être changé avant le seed en production")
		}
		if err := services.ValidatePassword(cfg.AdminPassword); err != nil {
			return fmt.Errorf("ADMIN_PASSWORD invalide: %w", err)
		}
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
