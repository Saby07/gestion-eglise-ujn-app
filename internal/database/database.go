package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"eglise_ujn/internal/models"

	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var db *gorm.DB

func Init() {
	driver := strings.ToLower(strings.TrimSpace(os.Getenv("DB_DRIVER")))
	if driver == "" {
		driver = "sqlite"
	}

	gormLogger := logger.Default.LogMode(logger.Warn)
	if os.Getenv("GIN_MODE") != "release" {
		gormLogger = logger.Default.LogMode(logger.Info)
	}

	var dialector gorm.Dialector
	switch driver {
	case "sqlite":
		dialector = sqlite.Open(sqliteDSN())
	case "mysql":
		dialector = mysql.Open(mysqlDSN())
	default:
		log.Fatalf("DB_DRIVER invalide: %q (attendu: sqlite ou mysql)", driver)
	}

	var err error
	db, err = gorm.Open(dialector, &gorm.Config{Logger: gormLogger})
	if err != nil {
		log.Fatal("connexion DB échouée:", err)
	}

	if driver == "sqlite" {
		_ = db.Exec("PRAGMA foreign_keys = ON").Error
		_ = db.Exec("PRAGMA journal_mode = WAL").Error
		_ = db.Exec("PRAGMA busy_timeout = 5000").Error
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("sql.DB:", err)
	}
	if driver == "sqlite" {
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
		&models.Requisition{},
		&models.RequisitionItem{},
		&models.RequisitionValidation{},
		&models.Disbursement{},
		&models.Notification{},
	); err != nil {
		log.Fatal("migration:", err)
	}
	log.Printf("[DB] Connecté (%s) et migrations appliquées", driver)
}

func sqliteDSN() string {
	path := strings.TrimSpace(os.Getenv("DB_PATH"))
	if path == "" {
		path = "./data/eglise_ujn.db"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		log.Fatal("impossible de créer le dossier data:", err)
	}
	return path
}

func mysqlDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)
}

func GetDB() *gorm.DB {
	return db
}
