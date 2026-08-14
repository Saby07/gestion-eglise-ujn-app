package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"eglise_ujn/internal/config"
	"eglise_ujn/internal/database"
	"eglise_ujn/internal/routes"
	"eglise_ujn/internal/seed"

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

func main() {
	loadEnv()
	app := config.LoadApp()
	gin.SetMode(app.GinMode)

	database.Init()
	db := database.GetDB()

	if app.RunSeed {
		if err := seed.Run(db, seed.Config{
			AdminEmail:    app.AdminEmail,
			AdminPassword: app.AdminPassword,
		}); err != nil {
			log.Fatal("seed:", err)
		}
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
	r.Static("/uploads", "./uploads")
	routes.Setup(r, db)

	addr := app.Server + ":" + app.Port
	log.Printf("[HTTP] http://%s (health: /healthz)", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
