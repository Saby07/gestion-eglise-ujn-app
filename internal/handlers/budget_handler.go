package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"eglise_ujn/internal/httputil"
	"eglise_ujn/internal/middlewares"
	"eglise_ujn/internal/services"
	"eglise_ujn/templates/pages"

	"github.com/gin-gonic/gin"
)

type BudgetHandler struct {
	svc         *services.BudgetService
	categorySvc *services.CategoryService
}

func NewBudgetHandler(svc *services.BudgetService, categorySvc *services.CategoryService) *BudgetHandler {
	return &BudgetHandler{svc: svc, categorySvc: categorySvc}
}

func (h *BudgetHandler) List(c *gin.Context) {
	year := time.Now().Year()
	if yStr := strings.TrimSpace(c.Query("year")); yStr != "" {
		if y, err := strconv.Atoi(yStr); err == nil && y > 0 {
			year = y
		}
	}
	list, _ := h.svc.ListWithUsage(c.Request.Context(), year)
	httputil.Render(c, http.StatusOK, pages.Budgets(layoutFromCtx(c, "Budgets", "budgets"), year, mapBudgets(list)))
}

func (h *BudgetHandler) NewPage(c *gin.Context) {
	year := time.Now().Year()
	if yStr := strings.TrimSpace(c.Query("year")); yStr != "" {
		if y, err := strconv.Atoi(yStr); err == nil && y > 0 {
			year = y
		}
	}
	cats, _ := h.categorySvc.List(c.Request.Context(), true)
	httputil.Render(c, http.StatusOK, pages.BudgetNew(layoutFromCtx(c, "Nouveau budget", "budgets"), year, mapCategories(cats)))
}

func (h *BudgetHandler) Create(c *gin.Context) {
	cur := middlewares.CurrentUser(c)
	by := ""
	if cur != nil {
		by = cur.FullName()
	}
	categoryID, _ := strconv.ParseUint(c.PostForm("category_id"), 10, 64)
	year, _ := strconv.Atoi(strings.TrimSpace(c.PostForm("year")))
	amount, _ := strconv.ParseFloat(strings.TrimSpace(c.PostForm("amount")), 64)
	_, err := h.svc.Create(c.Request.Context(), uint(categoryID), year, amount, by)
	if err != nil {
		httputil.RedirectFlash(c, "/budgets/new", "Erreur lors de la création")
		return
	}
	httputil.RedirectFlash(c, "/budgets?year="+strconv.Itoa(year), "Budget créé")
}

func (h *BudgetHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	cur := middlewares.CurrentUser(c)
	by := ""
	if cur != nil {
		by = cur.FullName()
	}
	amount, _ := strconv.ParseFloat(strings.TrimSpace(c.PostForm("amount")), 64)
	b, err := h.svc.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		httputil.RedirectFlash(c, "/budgets", "Budget introuvable")
		return
	}
	if err := h.svc.Update(c.Request.Context(), uint(id), amount, by); err != nil {
		httputil.RedirectFlash(c, "/budgets?year="+strconv.Itoa(b.Year), "Erreur lors de la mise à jour")
		return
	}
	httputil.RedirectFlash(c, "/budgets?year="+strconv.Itoa(b.Year), "Budget mis à jour")
}

func (h *BudgetHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	b, err := h.svc.GetByID(c.Request.Context(), uint(id))
	year := time.Now().Year()
	if err == nil {
		year = b.Year
	}
	if err := h.svc.Delete(c.Request.Context(), uint(id)); err != nil {
		httputil.RedirectFlash(c, "/budgets", "Erreur lors de la suppression")
		return
	}
	httputil.RedirectFlash(c, "/budgets?year="+strconv.Itoa(year), "Budget supprimé")
}
