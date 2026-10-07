package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"eglise_ujn/internal/config"
	"eglise_ujn/internal/database"
	"eglise_ujn/internal/routes"
	"eglise_ujn/internal/seed"
	"eglise_ujn/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func loadEnv() {
	if err := godotenv.Load(); err == nil {
		return
	}
	if exe, err := os.Executable(); err == nil {
		_ = godotenv.Load(filepath.Join(filepath.Dir(exe), ".env"))
	}
}

func envBool(key string) bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv(key)), "true")
}

func main() {
	loadEnv()
	app := config.LoadApp()
	gin.SetMode(app.GinMode)

	if _, err := config.LoadJWTConfig(); err != nil {
		log.Fatal("config JWT:", err)
	}

	database.Init()
	db := database.GetDB()

	if app.RunSeed {
		if err := seed.Run(db, seed.Config{
			AdminEmail:    app.AdminEmail,
			AdminPassword: app.AdminPassword,
			GinMode:       app.GinMode,
		}); err != nil {
			log.Fatal("seed:", err)
		}
	}

	if envBool("BACKUP_ON_START") {
		backupSvc := services.NewBackupService(db)
		if path, err := backupSvc.RunBackup(context.Background()); err != nil {
			log.Printf("[BACKUP] échec au démarrage: %v", err)
		} else {
			log.Printf("[BACKUP] sauvegarde créée: %s", path)
		}
	}

	if envBool("REMINDER_ENABLED") {
		settingsSvc := services.NewSettingsService(db)
		emailSvc := services.NewEmailService(settingsSvc)
		workflowSvc := services.NewWorkflowService(db, settingsSvc)
		notifSvc := services.NewNotificationService(db, emailSvc, workflowSvc)
		reminderSvc := services.NewReminderService(db, settingsSvc, notifSvc, emailSvc)
		go func() {
			ticker := time.NewTicker(24 * time.Hour)
			defer ticker.Stop()
			runReminders(reminderSvc)
			for range ticker.C {
				runReminders(reminderSvc)
			}
		}()
		log.Println("[REMINDER] goroutine de rappels activée")
	}

	r := gin.Default()
	if proxies := strings.TrimSpace(os.Getenv("TRUSTED_PROXIES")); proxies != "" {
		parts := strings.Split(proxies, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		if err := r.SetTrustedProxies(parts); err != nil {
			log.Printf("[HTTP] TRUSTED_PROXIES ignoré: %v", err)
		}
	}
	r.Static("/assets", "./assets")
	routes.Setup(r, db)

	addr := app.Server + ":" + app.Port
	log.Printf("[HTTP] http://%s (health: /healthz)", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}

func runReminders(svc *services.ReminderService) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	n, err := svc.ProcessReminders(ctx)
	if err != nil {
		log.Printf("[REMINDER] erreur: %v", err)
		return
	}
	if n > 0 {
		log.Printf("[REMINDER] %d rappel(s) traité(s)", n)
	}
}
