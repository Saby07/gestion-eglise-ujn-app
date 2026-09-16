package seed_test

import (
	"os"
	"testing"

	"eglise_ujn/internal/database"
	"eglise_ujn/internal/models"
	"eglise_ujn/internal/seed"
	"eglise_ujn/internal/services"
)

func TestRunSeedsSuperAdminOnly(t *testing.T) {
	path := t.TempDir() + "/test.db"
	t.Setenv("DB_DRIVER", "sqlite")
	t.Setenv("DB_PATH", path)

	database.Init()
	db := database.GetDB()

	if err := seed.Run(db, seed.Config{
		AdminEmail:    "superadmin@test.local",
		AdminPassword: "ChangeMe123!",
		GinMode:       "debug",
	}); err != nil {
		t.Fatal(err)
	}

	var roleCount int64
	if err := db.Model(&models.Role{}).Count(&roleCount).Error; err != nil {
		t.Fatal(err)
	}
	if roleCount != int64(len(models.AllRoles)) {
		t.Fatalf("roles count = %d, want %d", roleCount, len(models.AllRoles))
	}

	var userCount int64
	if err := db.Model(&models.User{}).Count(&userCount).Error; err != nil {
		t.Fatal(err)
	}
	if userCount != 1 {
		t.Fatalf("users count = %d, want 1 (super admin only)", userCount)
	}

	var catCount int64
	if err := db.Model(&models.ExpenseCategory{}).Count(&catCount).Error; err != nil {
		t.Fatal(err)
	}
	if catCount != 5 {
		t.Fatalf("categories count = %d, want 5", catCount)
	}

	settingsSvc := services.NewSettingsService(db)
	name, _ := settingsSvc.Get(t.Context(), services.SettingChurchName)
	if name != "Église UJN" {
		t.Fatalf("church_name = %q, want Église UJN", name)
	}

	// Idempotent: run again with existing users
	if err := seed.Run(db, seed.Config{
		AdminEmail:    "superadmin@test.local",
		AdminPassword: "ChangeMe123!",
		GinMode:       "debug",
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.User{}).Count(&userCount).Error; err != nil {
		t.Fatal(err)
	}
	if userCount != 1 {
		t.Fatalf("after second run users count = %d, want 1", userCount)
	}
}

func TestEnsureSuperAdminUserRoleWhenUserExists(t *testing.T) {
	path := t.TempDir() + "/test.db"
	t.Setenv("DB_DRIVER", "sqlite")
	t.Setenv("DB_PATH", path)

	database.Init()
	db := database.GetDB()

	if err := seed.Run(db, seed.Config{
		AdminEmail:    "superadmin@eglise-ujn.local",
		AdminPassword: "ChangeMe123!",
		GinMode:       "debug",
	}); err != nil {
		t.Fatal(err)
	}

	var super models.User
	if err := db.Where("email = ?", "superadmin@eglise-ujn.local").First(&super).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("user_id = ?", super.ID).Delete(&models.UserRole{}).Error; err != nil {
		t.Fatal(err)
	}

	var links int64
	if err := db.Model(&models.UserRole{}).Where("user_id = ?", super.ID).Count(&links).Error; err != nil {
		t.Fatal(err)
	}
	if links != 0 {
		t.Fatalf("superadmin user_roles count = %d, want 0", links)
	}

	if err := seed.Run(db, seed.Config{
		AdminEmail:    "superadmin@eglise-ujn.local",
		AdminPassword: "ChangeMe123!",
		GinMode:       "debug",
	}); err != nil {
		t.Fatal(err)
	}

	var role models.Role
	if err := db.Where(models.Role{Key: models.RoleSuperAdmin}).First(&role).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.UserRole{}).
		Where("user_id = ? AND role_id = ?", super.ID, role.ID).
		Count(&links).Error; err != nil {
		t.Fatal(err)
	}
	if links != 1 {
		t.Fatalf("superadmin user_role count = %d, want 1", links)
	}
}

func TestReleaseModeRejectsDefaultPassword(t *testing.T) {
	path := t.TempDir() + "/test.db"
	t.Setenv("DB_DRIVER", "sqlite")
	t.Setenv("DB_PATH", path)

	database.Init()
	db := database.GetDB()

	err := seed.Run(db, seed.Config{
		AdminEmail:    "superadmin@test.local",
		AdminPassword: "ChangeMe123!",
		GinMode:       "release",
	})
	if err == nil {
		t.Fatal("expected error for default password in release mode")
	}
}

func TestSeedRequiresAdminPassword(t *testing.T) {
	path := t.TempDir() + "/test.db"
	t.Setenv("DB_DRIVER", "sqlite")
	t.Setenv("DB_PATH", path)

	database.Init()
	db := database.GetDB()

	err := seed.Run(db, seed.Config{
		AdminEmail: "superadmin@test.local",
		GinMode:    "debug",
	})
	if err == nil {
		t.Fatal("expected error when ADMIN_PASSWORD is empty")
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
