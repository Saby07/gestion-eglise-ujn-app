package config

import (
	"fmt"
	"os"
	"strings"
)

type DBConfig struct {
	Driver   string
	Path     string
	Host     string
	Port     string
	Name     string
	User     string
	Password string
}

func LoadDB() DBConfig {
	cfg := DBConfig{
		Driver:   strings.ToLower(strings.TrimSpace(os.Getenv("DB_DRIVER"))),
		Path:     strings.TrimSpace(os.Getenv("DB_PATH")),
		Host:     strings.TrimSpace(os.Getenv("DB_HOST")),
		Port:     strings.TrimSpace(os.Getenv("DB_PORT")),
		Name:     strings.TrimSpace(os.Getenv("DB_NAME")),
		User:     strings.TrimSpace(os.Getenv("DB_USER")),
		Password: os.Getenv("DB_PASSWORD"),
	}

	if cfg.Driver == "" {
		cfg.Driver = "sqlite"
	}
	if cfg.Path == "" {
		cfg.Path = "./data/eglise_ujn.db"
	}
	if cfg.Host == "" {
		cfg.Host = "127.0.0.1"
	}
	if cfg.Port == "" {
		cfg.Port = "3306"
	}
	if cfg.Name == "" {
		cfg.Name = "eglise_ujn"
	}

	return cfg
}

func (c DBConfig) MySQLDSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.Name,
	)
}

func (c DBConfig) Validate() error {
	switch c.Driver {
	case "sqlite":
		return nil
	case "mysql":
		if c.User == "" {
			return fmt.Errorf("DB_USER est requis pour MySQL")
		}
		if c.Name == "" {
			return fmt.Errorf("DB_NAME est requis pour MySQL")
		}
		return nil
	default:
		return fmt.Errorf("DB_DRIVER invalide: %q (attendu: sqlite ou mysql)", c.Driver)
	}
}
