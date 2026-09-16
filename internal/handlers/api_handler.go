package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"eglise_ujn/internal/middlewares"
	"eglise_ujn/internal/services"

	"github.com/gin-gonic/gin"
)

type APIHandler struct {
	reqSvc     *services.RequisitionService
	accountSvc *services.AccountService
	statsSvc   *services.StatsService
}

func NewAPIHandler(reqSvc *services.RequisitionService, accountSvc *services.AccountService, statsSvc *services.StatsService) *APIHandler {
	return &APIHandler{reqSvc: reqSvc, accountSvc: accountSvc, statsSvc: statsSvc}
}

func apiError(c *gin.Context, status int, message string) {
	if wantsJSON(c) {
		c.JSON(status, gin.H{"error": message})
		return
	}
	c.String(status, message)
}

func wantsJSON(c *gin.Context) bool {
	accept := c.GetHeader("Accept")
	return strings.Contains(accept, "application/json") || strings.HasPrefix(c.Request.URL.Path, "/api/")
}

func (h *APIHandler) ListRequisitions(c *gin.Context) {
	user := middlewares.CurrentUser(c)
	list, err := h.reqSvc.ListWithSearch(c.Request.Context(), parseSearchFilter(c), user)
	if err != nil {
		apiError(c, http.StatusInternalServerError, "Erreur serveur")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

func (h *APIHandler) GetRequisition(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		apiError(c, http.StatusBadRequest, "ID invalide")
		return
	}
	user := middlewares.CurrentUser(c)
	req, err := h.reqSvc.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		apiError(c, http.StatusNotFound, "Réquisition introuvable")
		return
	}
	if !h.reqSvc.CanView(user, req) {
		apiError(c, http.StatusForbidden, "Accès non autorisé")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": req})
}

func (h *APIHandler) ListAccounts(c *gin.Context) {
	user := middlewares.CurrentUser(c)
	if user == nil || !user.CanViewAccountBalance() {
		apiError(c, http.StatusForbidden, "Accès non autorisé")
		return
	}
	list, err := h.accountSvc.List(c.Request.Context())
	if err != nil {
		apiError(c, http.StatusInternalServerError, "Erreur serveur")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

func (h *APIHandler) Stats(c *gin.Context) {
	year := time.Now().Year()
	if yStr := c.Query("year"); yStr != "" {
		if y, err := strconv.Atoi(yStr); err == nil && y > 0 {
			year = y
		}
	}
	stats, err := h.statsSvc.AdvancedStats(c.Request.Context(), year)
	if err != nil {
		apiError(c, http.StatusInternalServerError, "Erreur serveur")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": stats, "year": year})
}
