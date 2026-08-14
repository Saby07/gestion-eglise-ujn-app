package routes

import (
	"net/http"

	"eglise_ujn/internal/handlers"
	"eglise_ujn/internal/middlewares"
	"eglise_ujn/internal/models"
	"eglise_ujn/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Setup(r *gin.Engine, db *gorm.DB) {
	authSvc := services.NewAuthService(db)
	accountSvc := services.NewAccountService(db)
	notifSvc := services.NewNotificationService(db)
	reqSvc := services.NewRequisitionService(db, notifSvc)
	userSvc := services.NewUserService(db)
	reportSvc := services.NewReportService(db)
	dashSvc := services.NewDashboardService(db, accountSvc, reqSvc)

	authH := handlers.NewAuthHandler(authSvc, userSvc)
	dashH := handlers.NewDashboardHandler(dashSvc)
	usersH := handlers.NewUsersHandler(userSvc, reqSvc)
	accountsH := handlers.NewAccountsHandler(accountSvc)
	reqH := handlers.NewRequisitionsHandler(reqSvc, accountSvc)
	reportsH := handlers.NewReportsHandler(reportSvc)
	notifH := handlers.NewNotificationHandler(notifSvc)

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/", func(c *gin.Context) { c.Redirect(http.StatusFound, "/dashboard") })
	r.GET("/login", authH.LoginPage)
	r.POST("/login", authH.LoginPost)

	auth := r.Group("/")
	auth.Use(middlewares.AuthMiddleware(authSvc))
	auth.Use(handlers.SetNotificationService(notifSvc))
	auth.Use(middlewares.CSRFMiddleware())

	auth.GET("/logout", authH.Logout)
	auth.GET("/profile/password", authH.ProfilePasswordPage)
	auth.POST("/profile/password", authH.ProfileUpdatePassword)
	auth.GET("/dashboard", dashH.Index)
	auth.GET("/notifications/:id/read", notifH.Read)

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
	users.POST("/:id/can-disburse", usersH.ToggleCanDisburse)

	// Comptes — Admin + Accountant
	accounts := auth.Group("/accounts")
	accounts.Use(middlewares.RequireRoles(models.RoleAdmin, models.RoleAccountant, models.RoleSuperAdmin))
	accounts.GET("", accountsH.List)
	accounts.GET("/new", accountsH.NewPage)
	accounts.POST("", accountsH.Create)
	accounts.GET("/:id", accountsH.Detail)
	accounts.POST("/:id/fund", middlewares.RequireRoles(models.RoleAccountant), accountsH.Fund)

	// Réquisitions
	req := auth.Group("/requisitions")
	req.GET("", reqH.List)
	req.GET("/new", middlewares.RequireRoles(models.RoleStaff, models.RoleAdmin, models.RoleSuperAdmin), reqH.NewPage)
	req.POST("", middlewares.RequireRoles(models.RoleStaff, models.RoleAdmin, models.RoleSuperAdmin), reqH.Create)
	req.GET("/:id", reqH.Detail)
	req.POST("/:id/validate", reqH.Validate)
	req.POST("/:id/cancel", reqH.Cancel)
	req.POST("/:id/delete", middlewares.RequireRoles(models.RoleAdmin, models.RoleSuperAdmin), reqH.Delete)
	req.POST("/:id/disburse", reqH.Disburse)

	// Rapports
	reports := auth.Group("/reports")
	reports.Use(middlewares.RequireRoles(models.RoleAdmin, models.RoleAccountant, models.RoleSuperAdmin, models.RoleCashier))
	reports.GET("", reportsH.Index)
}
