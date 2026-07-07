package handlers

import (
	"eglise_ujn/internal/middlewares"
	"eglise_ujn/internal/models"
	"eglise_ujn/internal/services"
	"eglise_ujn/templates/viewmodels"

	"github.com/gin-gonic/gin"
)

const notifSvcKey = "notifSvc"

func SetNotificationService(svc *services.NotificationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(notifSvcKey, svc)
		c.Next()
	}
}

func layoutFromCtx(c *gin.Context, title, menu string) viewmodels.LayoutVM {
	u := middlewares.CurrentUser(c)
	vm := viewmodels.LayoutVM{
		Title:      title,
		CSRF:       middlewares.SetCSRFCookie(c),
		Alert:      c.Query("alert"),
		ActiveMenu: menu,
	}
	if u != nil {
		vm.CurrentUserID = u.ID
		vm.UserName = u.FullName()
		vm.UserRole = u.PrimaryRole().Label()
		vm.IsSuperAdmin = u.HasRole(models.RoleSuperAdmin)
		vm.IsAdmin = u.HasRole(models.RoleAdmin)
		vm.IsAccountant = u.HasRole(models.RoleAccountant)
		vm.IsCashier = u.HasRole(models.RoleCashier)
		vm.IsStaff = u.HasRole(models.RoleStaff)
		if raw, ok := c.Get(notifSvcKey); ok {
			if notifSvc, ok := raw.(*services.NotificationService); ok {
				list, _ := notifSvc.ListForUser(c.Request.Context(), u.ID, 15)
				vm.Notifications = viewmodels.MapNotifications(list)
				vm.UnreadCount, _ = notifSvc.CountUnread(c.Request.Context(), u.ID)
			}
		}
	}
	return vm
}
