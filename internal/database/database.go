package database

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"eglise_ujn/internal/config"
	"eglise_ujn/internal/models"

	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var db *gorm.DB

func Init() {
	cfg := config.LoadDB()
	if err := cfg.Validate(); err != nil {
		log.Fatal(err)
	}

	gormLogger := logger.Default.LogMode(logger.Warn)
	if os.Getenv("GIN_MODE") != "release" {
		gormLogger = logger.Default.LogMode(logger.Info)
	}

	var dialector gorm.Dialector
	switch cfg.Driver {
	case "sqlite":
		dialector = sqlite.Open(sqliteDSN(cfg.Path))
	case "mysql":
		dialector = mysql.Open(cfg.MySQLDSN())
	default:
		log.Fatalf("DB_DRIVER invalide: %q (attendu: sqlite ou mysql)", cfg.Driver)
	}

	var err error
	db, err = gorm.Open(dialector, &gorm.Config{Logger: gormLogger})
	if err != nil {
		log.Fatal("connexion DB échouée:", err)
	}

	if cfg.Driver == "sqlite" {
		// OFF pendant AutoMigrate : SQLite/GORM peut DROP+recréer une table
		// (ex. nouvelles FK) et échoue sinon avec "FOREIGN KEY constraint failed".
		_ = db.Exec("PRAGMA foreign_keys = OFF").Error
		_ = db.Exec("PRAGMA journal_mode = WAL").Error
		_ = db.Exec("PRAGMA busy_timeout = 5000").Error
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("sql.DB:", err)
	}
	if cfg.Driver == "sqlite" {
		sqlDB.SetMaxOpenConns(1)
	} else {
		sqlDB.SetMaxOpenConns(25)
		sqlDB.SetMaxIdleConns(5)
	}
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err := db.AutoMigrate(
		&models.Role{},
		&models.User{},
		&models.UserRole{},
		&models.ChurchAccount{},
		&models.AccountTransaction{},
		&models.ExpenseCategory{},
		&models.Supplier{},
		&models.Budget{},
		&models.AppSetting{},
		&models.Requisition{},
		&models.RequisitionItem{},
		&models.RequisitionValidation{},
		&models.Disbursement{},
		&models.Notification{},
		&models.AuditLog{},
	); err != nil {
		log.Fatal("migration:", err)
	}

	if cfg.Driver == "sqlite" {
		_ = db.Exec("PRAGMA foreign_keys = ON").Error
	}

	if cfg.Driver == "mysql" {
		log.Printf("[DB] Connecté à MySQL %s:%s/%s", cfg.Host, cfg.Port, cfg.Name)
	} else {
		log.Printf("[DB] Connecté à SQLite (%s)", cfg.Path)
	}
	log.Printf("[DB] Migrations appliquées")
}

func sqliteDSN(path string) string {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		log.Fatal("impossible de créer le dossier data:", err)
	}
	return path
}

func GetDB() *gorm.DB {
	return db
}
