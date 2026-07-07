package seed_test

import (
	"os"
	"testing"

	"eglise_ujn/internal/database"
	"eglise_ujn/internal/models"
	"eglise_ujn/internal/seed"
)

func TestRunSeedsAllRoles(t *testing.T) {
	path := t.TempDir() + "/test.db"
	t.Setenv("DB_DRIVER", "sqlite")
	t.Setenv("DB_PATH", path)
	t.Setenv("GIN_MODE", "release")

	database.Init()
	db := database.GetDB()

	if err := seed.Run(db, seed.Config{
		AdminEmail:    "superadmin@test.local",
		AdminPassword: "ChangeMe123!",
	}); err != nil {
		t.Fatal(err)
	}

	var count int64
	if err := db.Model(&models.Role{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != int64(len(models.AllRoles)) {
		t.Fatalf("roles count = %d, want %d", count, len(models.AllRoles))
	}

	// Idempotent: run again with existing users
	if err := seed.Run(db, seed.Config{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.Role{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != int64(len(models.AllRoles)) {
		t.Fatalf("after second run roles count = %d, want %d", count, len(models.AllRoles))
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
