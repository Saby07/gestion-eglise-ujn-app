package handlers

import (
	"net/http"
	"strconv"

	"eglise_ujn/internal/httputil"
	"eglise_ujn/internal/middlewares"
	"eglise_ujn/internal/services"
	"eglise_ujn/templates/pages"
	"eglise_ujn/templates/viewmodels"

	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	svc *services.NotificationService
}

func NewNotificationHandler(svc *services.NotificationService) *NotificationHandler {
	return &NotificationHandler{svc: svc}
}

func (h *NotificationHandler) ListPage(c *gin.Context) {
	user := middlewares.CurrentUser(c)
	if user == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}
	list, _ := h.svc.ListForUser(c.Request.Context(), user.ID, 100)
	httputil.Render(c, http.StatusOK, pages.NotificationsList(layoutFromCtx(c, "Notifications", "notifications"), viewmodels.MapNotifications(list)))
}

func (h *NotificationHandler) Read(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.Redirect(http.StatusFound, "/notifications")
		return
	}
	user := middlewares.CurrentUser(c)
	if user == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}
	n, err := h.svc.MarkRead(c.Request.Context(), uint(id), user.ID)
	if err != nil {
		c.Redirect(http.StatusFound, "/notifications")
		return
	}
	c.Redirect(http.StatusFound, "/requisitions/"+strconv.FormatUint(uint64(n.RequisitionID), 10))
}

func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	user := middlewares.CurrentUser(c)
	if user == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}
	_ = h.svc.MarkAllRead(c.Request.Context(), user.ID)
	httputil.RedirectFlash(c, "/notifications", "Toutes les notifications ont été marquées comme lues")
}
