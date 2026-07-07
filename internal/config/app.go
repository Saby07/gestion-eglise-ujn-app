package config

import (
	"os"
	"strings"
)

type App struct {
	GinMode       string
	Port          string
	Server        string
	RunSeed       bool
	AdminEmail    string
	AdminPassword string
}

func LoadApp() App {
	a := App{
		GinMode: strings.TrimSpace(os.Getenv("GIN_MODE")),
		Port:    strings.TrimSpace(os.Getenv("PORT")),
		Server:  strings.TrimSpace(os.Getenv("SERVER")),
	}
	if a.GinMode == "" {
		a.GinMode = "debug"
	}
	if a.Port == "" {
		a.Port = "8080"
	}
	if a.Server == "" {
		a.Server = "0.0.0.0"
	}
	a.RunSeed = strings.EqualFold(strings.TrimSpace(os.Getenv("RUN_SEED")), "true")
	a.AdminEmail = strings.TrimSpace(os.Getenv("ADMIN_EMAIL"))
	a.AdminPassword = os.Getenv("ADMIN_PASSWORD")
	return a
}
