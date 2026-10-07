package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"eglise_ujn/internal/httputil"
	"eglise_ujn/internal/middlewares"
	"eglise_ujn/internal/models"
	"eglise_ujn/internal/services"
	"eglise_ujn/templates/pages"
	"eglise_ujn/templates/viewmodels"

	"github.com/gin-gonic/gin"
)

type SettingsHandler struct {
	settings    *services.SettingsService
	backup      *services.BackupService
	audit       *services.AuditService
	workflowSvc *services.WorkflowService
}

func NewSettingsHandler(settings *services.SettingsService, backup *services.BackupService, audit *services.AuditService, workflowSvc *services.WorkflowService) *SettingsHandler {
	return &SettingsHandler{settings: settings, backup: backup, audit: audit, workflowSvc: workflowSvc}
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

func (h *SettingsHandler) SaveWorkflow(c *gin.Context) {
	schema := models.DefaultWorkflowSchema()
	schema.Capabilities = map[models.RoleKey]models.RoleCapabilities{}
	for _, role := range models.AllRoles {
		schema.Capabilities[role] = models.RoleCapabilities{
			Create:     c.PostForm("cap_create_"+string(role)) == "1",
			AttachDocs: c.PostForm("cap_attach_"+string(role)) == "1",
			Disburse:   c.PostForm("cap_disburse_"+string(role)) == "1",
			Fund:       c.PostForm("cap_fund_"+string(role)) == "1",
		}
	}
	chainRaw := c.PostFormArray("validation_chain")
	chain := make([]models.RoleKey, 0, len(chainRaw))
	for _, k := range chainRaw {
		rk := models.RoleKey(strings.TrimSpace(k))
		if rk.Valid() && rk != models.RoleSuperAdmin {
			chain = append(chain, rk)
		}
	}
	schema.ValidationChain = chain

	if err := h.workflowSvc.Save(c.Request.Context(), schema); err != nil {
		httputil.RedirectFlash(c, "/settings", "Impossible d'enregistrer le schéma : "+err.Error())
		return
	}
	cur := middlewares.CurrentUser(c)
	if cur != nil && h.audit != nil {
		_ = h.audit.Log(c.Request.Context(), cur.ID, "update", "workflow", 0, "Mise à jour du schéma de validation", clientIP(c))
	}
	httputil.RedirectFlash(c, "/settings", "Schéma de validation enregistré")
}

func (h *SettingsHandler) Backup(c *gin.Context) {
	path, err := h.backup.RunBackup(c.Request.Context())
	if err != nil {
		httputil.RedirectFlash(c, "/settings", "Erreur lors de la sauvegarde")
		return
	}
	cur := middlewares.CurrentUser(c)
	if cur != nil && h.audit != nil {
		_ = h.audit.Log(c.Request.Context(), cur.ID, "backup", "settings", 0, "Sauvegarde : "+path, clientIP(c))
	}
	httputil.RedirectFlash(c, "/settings", "Sauvegarde créée avec succès")
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
	schema, err := h.workflowSvc.Load(ctx)
	if err != nil {
		schema = models.DefaultWorkflowSchema()
	}
	roleCaps := make([]viewmodels.WorkflowRoleCapVM, 0, len(models.AllRoles))
	for _, role := range models.AllRoles {
		caps := schema.Capabilities[role]
		roleCaps = append(roleCaps, viewmodels.WorkflowRoleCapVM{
			Key:        string(role),
			Label:      role.Label(),
			Create:     caps.Create,
			AttachDocs: caps.AttachDocs,
			Disburse:   caps.Disburse,
			Fund:       caps.Fund,
			IsSuper:    role == models.RoleSuperAdmin,
		})
	}
	chain := make([]viewmodels.WorkflowChainItemVM, 0, len(models.IntermediateRoles))
	inChain := map[models.RoleKey]int{}
	for i, r := range schema.ValidationChain {
		inChain[r] = i
	}
	for _, role := range models.IntermediateRoles {
		order, ok := inChain[role]
		chain = append(chain, viewmodels.WorkflowChainItemVM{
			Key:      string(role),
			Label:    role.Label(),
			Selected: ok,
			Order:    order,
		})
	}
	return viewmodels.SettingsVM{
		ChurchName:      church,
		DarkMode:        theme == "dark",
		SMTPHost:        smtpHost,
		SMTPPort:        smtpPort,
		SMTPUser:        smtpUser,
		SMTPFrom:        smtpFrom,
		ReminderDays:    reminderDays,
		WorkflowRoles:   roleCaps,
		WorkflowChain:   chain,
		WorkflowPreview: services.ChainPreviewLabels(schema),
	}, nil
}
