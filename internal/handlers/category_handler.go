package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"eglise_ujn/internal/httputil"
	"eglise_ujn/internal/middlewares"
	"eglise_ujn/internal/services"
	"eglise_ujn/templates/pages"

	"github.com/gin-gonic/gin"
)

type CategoryHandler struct {
	svc *services.CategoryService
}

func NewCategoryHandler(svc *services.CategoryService) *CategoryHandler {
	return &CategoryHandler{svc: svc}
}

func (h *CategoryHandler) List(c *gin.Context) {
	list, _ := h.svc.List(c.Request.Context(), false)
	httputil.Render(c, http.StatusOK, pages.Categories(layoutFromCtx(c, "Catégories", "categories"), mapCategories(list)))
}

func (h *CategoryHandler) Create(c *gin.Context) {
	cur := middlewares.CurrentUser(c)
	by := ""
	if cur != nil {
		by = cur.FullName()
	}
	_, err := h.svc.Create(c.Request.Context(),
		strings.TrimSpace(c.PostForm("name")),
		strings.TrimSpace(c.PostForm("description")),
		by,
	)
	if err != nil {
		httputil.RedirectFlash(c, "/categories", "Erreur lors de la création")
		return
	}
	httputil.RedirectFlash(c, "/categories", "Catégorie créée")
}

func (h *CategoryHandler) Toggle(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	cur := middlewares.CurrentUser(c)
	by := ""
	if cur != nil {
		by = cur.FullName()
	}
	if err := h.svc.Toggle(c.Request.Context(), uint(id), by); err != nil {
		httputil.RedirectFlash(c, "/categories", "Erreur lors de la mise à jour")
		return
	}
	httputil.RedirectFlash(c, "/categories", "Statut mis à jour")
}

func (h *CategoryHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.svc.Delete(c.Request.Context(), uint(id)); err != nil {
		httputil.RedirectFlash(c, "/categories", "Erreur lors de la suppression")
		return
	}
	httputil.RedirectFlash(c, "/categories", "Catégorie supprimée")
}
