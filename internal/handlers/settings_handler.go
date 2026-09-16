package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"eglise_ujn/internal/httputil"
	"eglise_ujn/internal/middlewares"
	"eglise_ujn/internal/services"
	"eglise_ujn/templates/pages"
	"eglise_ujn/templates/viewmodels"

	"github.com/gin-gonic/gin"
)

type SettingsHandler struct {
	settings *services.SettingsService
	backup   *services.BackupService
	audit    *services.AuditService
}

func NewSettingsHandler(settings *services.SettingsService, backup *services.BackupService, audit *services.AuditService) *SettingsHandler {
	return &SettingsHandler{settings: settings, backup: backup, audit: audit}
}

func (h *SettingsHandler) Index(c *gin.Context) {
	vm, err := h.loadSettingsVM(c)
	if err != nil {
		httputil.RedirectFlash(c, "/dashboard", "Impossible de charger les paramètres")
		return
	}
	httputil.Render(c, http.StatusOK, pages.Settings(layoutFromCtx(c, "Paramètres", "settings"), vm))
}

func (h *SettingsHandler) Save(c *gin.Context) {
	ctx := c.Request.Context()
	_ = h.settings.Set(ctx, services.SettingChurchName, strings.TrimSpace(c.PostForm("church_name")))
	if c.PostForm("dark_mode") == "1" {
		_ = h.settings.Set(ctx, services.SettingTheme, "dark")
	} else {
		_ = h.settings.Set(ctx, services.SettingTheme, "light")
	}
	_ = h.settings.Set(ctx, services.SettingReminderDays, strings.TrimSpace(c.PostForm("reminder_days")))
	_ = h.settings.Set(ctx, services.SettingSMTPHost, strings.TrimSpace(c.PostForm("smtp_host")))
	_ = h.settings.Set(ctx, services.SettingSMTPPort, strings.TrimSpace(c.PostForm("smtp_port")))
	_ = h.settings.Set(ctx, services.SettingSMTPUser, strings.TrimSpace(c.PostForm("smtp_user")))
	_ = h.settings.Set(ctx, services.SettingSMTPFrom, strings.TrimSpace(c.PostForm("smtp_from")))
	if pwd := c.PostForm("smtp_password"); strings.TrimSpace(pwd) != "" {
		_ = h.settings.Set(ctx, services.SettingSMTPPassword, pwd)
	}

	cur := middlewares.CurrentUser(c)
	if cur != nil && h.audit != nil {
		_ = h.audit.Log(ctx, cur.ID, "update", "settings", 0, "Mise à jour des paramètres", clientIP(c))
	}
	httputil.RedirectFlash(c, "/settings", "Paramètres enregistrés")
}

func (h *SettingsHandler) Backup(c *gin.Context) {
	path, err := h.backup.RunBackup(c.Request.Context())
	if err != nil {
		httputil.RedirectFlash(c, "/settings", "Erreur lors de la sauvegarde : "+err.Error())
		return
	}
	cur := middlewares.CurrentUser(c)
	if cur != nil && h.audit != nil {
		_ = h.audit.Log(c.Request.Context(), cur.ID, "backup", "settings", 0, "Sauvegarde : "+path, clientIP(c))
	}
	httputil.RedirectFlash(c, "/settings", "Sauvegarde créée : "+path)
}

func (h *SettingsHandler) loadSettingsVM(c *gin.Context) (viewmodels.SettingsVM, error) {
	ctx := c.Request.Context()
	church, _ := h.settings.Get(ctx, services.SettingChurchName)
	theme, _ := h.settings.Get(ctx, services.SettingTheme)
	reminderStr, _ := h.settings.Get(ctx, services.SettingReminderDays)
	smtpHost, _ := h.settings.Get(ctx, services.SettingSMTPHost)
	smtpPort, _ := h.settings.Get(ctx, services.SettingSMTPPort)
	smtpUser, _ := h.settings.Get(ctx, services.SettingSMTPUser)
	smtpFrom, _ := h.settings.Get(ctx, services.SettingSMTPFrom)
	reminderDays, _ := strconv.Atoi(reminderStr)
	if reminderDays <= 0 {
		reminderDays = 3
	}
	return viewmodels.SettingsVM{
		ChurchName:   church,
		DarkMode:     theme == "dark",
		SMTPHost:     smtpHost,
		SMTPPort:     smtpPort,
		SMTPUser:     smtpUser,
		SMTPFrom:     smtpFrom,
		ReminderDays: reminderDays,
	}, nil
}
