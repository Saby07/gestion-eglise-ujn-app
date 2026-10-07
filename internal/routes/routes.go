package routes

import (
	"context"
	"net/http"

	"eglise_ujn/internal/handlers"
	"eglise_ujn/internal/middlewares"
	"eglise_ujn/internal/models"
	"eglise_ujn/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Setup(r *gin.Engine, db *gorm.DB) {
	settingsSvc := services.NewSettingsService(db)
	emailSvc := services.NewEmailService(settingsSvc)
	auditSvc := services.NewAuditService(db)
	authSvc := services.NewAuthService(db)
	accountSvc := services.NewAccountService(db)
	workflowSvc := services.NewWorkflowService(db, settingsSvc)
	_ = workflowSvc.EnsureDefault(context.Background())
	notifSvc := services.NewNotificationService(db, emailSvc, workflowSvc)
	reqSvc := services.NewRequisitionService(db, notifSvc, auditSvc, emailSvc, workflowSvc)
	userSvc := services.NewUserService(db)
	reportSvc := services.NewReportService(db)
	dashSvc := services.NewDashboardService(db, accountSvc, reqSvc, workflowSvc)
	categorySvc := services.NewCategoryService(db)
	budgetSvc := services.NewBudgetService(db)
	supplierSvc := services.NewSupplierService(db)
	statsSvc := services.NewStatsService(db)
	exportSvc := services.NewExportService(db)
	backupSvc := services.NewBackupService(db)

	authH := handlers.NewAuthHandler(authSvc, userSvc)
	dashH := handlers.NewDashboardHandler(dashSvc)
	usersH := handlers.NewUsersHandler(userSvc)
	accountsH := handlers.NewAccountsHandler(accountSvc, workflowSvc)
	reqH := handlers.NewRequisitionsHandler(reqSvc, accountSvc, categorySvc, supplierSvc, auditSvc, workflowSvc)
	reportsH := handlers.NewReportsHandler(reportSvc)
	notifH := handlers.NewNotificationHandler(notifSvc)
	uploadH := handlers.NewUploadHandler()
	auditH := handlers.NewAuditHandler(auditSvc)
	categoryH := handlers.NewCategoryHandler(categorySvc)
	budgetH := handlers.NewBudgetHandler(budgetSvc, categorySvc)
	supplierH := handlers.NewSupplierHandler(supplierSvc)
	settingsH := handlers.NewSettingsHandler(settingsSvc, backupSvc, auditSvc, workflowSvc)
	statsH := handlers.NewStatsHandler(statsSvc)
	exportH := handlers.NewExportHandler(exportSvc, reportSvc, reqSvc)
	apiH := handlers.NewAPIHandler(reqSvc, accountSvc, statsSvc)

	r.Use(middlewares.SecurityHeaders())

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/", func(c *gin.Context) { c.Redirect(http.StatusFound, "/dashboard") })

	login := r.Group("/")
	login.Use(middlewares.LoginRateLimitMiddleware())
	login.Use(middlewares.LoginCSRFMiddleware())
	login.GET("/login", authH.LoginPage)
	login.POST("/login", authH.LoginPost)

	auth := r.Group("/")
	auth.Use(middlewares.AuthMiddleware(authSvc))
	auth.Use(handlers.SetNotificationService(notifSvc))
	auth.Use(handlers.SetSettingsService(settingsSvc))
	auth.Use(middlewares.CSRFMiddleware())

	auth.POST("/logout", authH.Logout)
	auth.GET("/profile/password", authH.ProfilePasswordPage)
	auth.POST("/profile/password", authH.ProfileUpdatePassword)
	auth.GET("/dashboard", dashH.Index)
	auth.GET("/notifications", notifH.ListPage)
	auth.POST("/notifications/read-all", notifH.MarkAllRead)
	auth.GET("/notifications/:id/read", notifH.Read)
	auth.GET("/uploads/*filepath", uploadH.Serve)

	// Utilisateurs — Super Admin uniquement
	users := auth.Group("/users")
	users.Use(middlewares.RequireRoles(models.RoleSuperAdmin))
	users.GET("", usersH.List)
	users.GET("/new", usersH.NewPage)
	users.GET("/:id/edit", usersH.EditPage)
	users.GET("/:id/password", usersH.PasswordPage)
	users.POST("", usersH.Create)
	users.POST("/:id/edit", usersH.Update)
	users.POST("/:id/password", usersH.UpdatePassword)
	users.POST("/:id/delete", usersH.Delete)
	users.POST("/:id/roles", usersH.ToggleRole)
	users.POST("/:id/toggle", usersH.ToggleActive)

	// Comptes — Admin + Accountant
	accounts := auth.Group("/accounts")
	accounts.Use(middlewares.RequireRoles(models.RoleAdmin, models.RoleAccountant, models.RoleSuperAdmin))
	accounts.GET("", accountsH.List)
	accounts.GET("/new", accountsH.NewPage)
	accounts.POST("", accountsH.Create)
	accounts.GET("/:id", accountsH.Detail)
	accounts.POST("/:id/fund", accountsH.Fund)

	// Catégories — Admin + Super Admin
	categories := auth.Group("/categories")
	categories.Use(middlewares.RequireRoles(models.RoleAdmin, models.RoleSuperAdmin))
	categories.GET("", categoryH.List)
	categories.POST("", categoryH.Create)
	categories.POST("/:id/toggle", categoryH.Toggle)
	categories.POST("/:id/delete", categoryH.Delete)

	// Budgets — Admin + Accountant + Super Admin
	budgets := auth.Group("/budgets")
	budgets.Use(middlewares.RequireRoles(models.RoleAdmin, models.RoleAccountant, models.RoleSuperAdmin))
	budgets.GET("", budgetH.List)
	budgets.GET("/new", budgetH.NewPage)
	budgets.POST("", budgetH.Create)
	budgets.POST("/:id/edit", budgetH.Update)
	budgets.POST("/:id/delete", budgetH.Delete)

	// Fournisseurs — Admin + Accountant + Super Admin
	suppliers := auth.Group("/suppliers")
	suppliers.Use(middlewares.RequireRoles(models.RoleAdmin, models.RoleAccountant, models.RoleSuperAdmin))
	suppliers.GET("", supplierH.List)
	suppliers.POST("", supplierH.Create)
	suppliers.GET("/:id/edit", supplierH.EditPage)
	suppliers.POST("/:id/edit", supplierH.Update)
	suppliers.POST("/:id/toggle", supplierH.Toggle)
	suppliers.POST("/:id/delete", supplierH.Delete)

	// Réquisitions
	req := auth.Group("/requisitions")
	req.GET("", reqH.List)
	req.GET("/new", reqH.NewPage)
	req.POST("", reqH.Create)
	req.GET("/:id/export", exportH.RequisitionExport)
	req.GET("/:id/edit", reqH.EditPage)
	req.POST("/:id/edit", reqH.Update)
	req.GET("/:id", reqH.Detail)
	req.POST("/:id/validate", reqH.Validate)
	req.POST("/:id/return", reqH.Return)
	req.POST("/:id/cancel", reqH.Cancel)
	req.POST("/:id/delete", middlewares.RequireRoles(models.RoleAdmin, models.RoleSuperAdmin), reqH.Delete)
	req.POST("/:id/disburse", reqH.Disburse)

	// Rapports & exports
	reports := auth.Group("/reports")
	reports.Use(middlewares.RequireRoles(models.RoleAdmin, models.RoleAccountant, models.RoleSuperAdmin, models.RoleCashier))
	reports.GET("", reportsH.Index)
	reports.GET("/export", exportH.ReportsExport)
	reports.GET("/export.csv", func(c *gin.Context) {
		q := c.Request.URL.Query()
		if q.Get("format") == "" {
			q.Set("format", "csv")
		}
		c.Request.URL.RawQuery = q.Encode()
		exportH.ReportsExport(c)
	})

	// Statistiques
	stats := auth.Group("/stats")
	stats.Use(middlewares.RequireRoles(models.RoleAdmin, models.RoleAccountant, models.RoleSuperAdmin, models.RoleCashier))
	stats.GET("", statsH.Index)

	// Journal d'audit — Super Admin
	audit := auth.Group("/audit")
	audit.Use(middlewares.RequireRoles(models.RoleSuperAdmin))
	audit.GET("", auditH.List)

	// Paramètres — Super Admin
	settings := auth.Group("/settings")
	settings.Use(middlewares.RequireRoles(models.RoleSuperAdmin))
	settings.GET("", settingsH.Index)
	settings.POST("", settingsH.Save)
	settings.POST("/workflow", settingsH.SaveWorkflow)
	settings.POST("/backup", settingsH.Backup)

	// API REST
	api := r.Group("/api/v1")
	api.Use(middlewares.AuthMiddleware(authSvc))
	api.GET("/requisitions", apiH.ListRequisitions)
	api.GET("/requisitions/:id", apiH.GetRequisition)
	api.GET("/accounts", apiH.ListAccounts)
	api.GET("/stats", middlewares.RequireRoles(models.RoleAdmin, models.RoleAccountant, models.RoleSuperAdmin, models.RoleCashier), apiH.Stats)
}
