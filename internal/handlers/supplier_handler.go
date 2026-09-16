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

	"github.com/gin-gonic/gin"
)

type SupplierHandler struct {
	svc *services.SupplierService
}

func NewSupplierHandler(svc *services.SupplierService) *SupplierHandler {
	return &SupplierHandler{svc: svc}
}

func (h *SupplierHandler) List(c *gin.Context) {
	list, _ := h.svc.List(c.Request.Context(), false)
	httputil.Render(c, http.StatusOK, pages.Suppliers(layoutFromCtx(c, "Fournisseurs", "suppliers"), mapSuppliers(list)))
}

func (h *SupplierHandler) Create(c *gin.Context) {
	cur := middlewares.CurrentUser(c)
	by := ""
	if cur != nil {
		by = cur.FullName()
	}
	_, err := h.svc.Create(c.Request.Context(),
		strings.TrimSpace(c.PostForm("name")),
		strings.TrimSpace(c.PostForm("phone")),
		strings.TrimSpace(c.PostForm("address")),
		strings.TrimSpace(c.PostForm("email")),
		by,
	)
	if err != nil {
		httputil.RedirectFlash(c, "/suppliers", "Erreur lors de la création")
		return
	}
	httputil.RedirectFlash(c, "/suppliers", "Fournisseur créé")
}

func (h *SupplierHandler) EditPage(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	sup, err := h.svc.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		httputil.RedirectFlash(c, "/suppliers", "Fournisseur introuvable")
		return
	}
	httputil.Render(c, http.StatusOK, pages.SupplierEdit(layoutFromCtx(c, "Modifier fournisseur", "suppliers"), mapSuppliers([]models.Supplier{*sup})[0]))
}

func (h *SupplierHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	cur := middlewares.CurrentUser(c)
	by := ""
	if cur != nil {
		by = cur.FullName()
	}
	err := h.svc.Update(c.Request.Context(), uint(id),
		strings.TrimSpace(c.PostForm("name")),
		strings.TrimSpace(c.PostForm("phone")),
		strings.TrimSpace(c.PostForm("address")),
		strings.TrimSpace(c.PostForm("email")),
		by,
	)
	if err != nil {
		httputil.RedirectFlash(c, "/suppliers/"+c.Param("id")+"/edit", "Erreur lors de la mise à jour")
		return
	}
	httputil.RedirectFlash(c, "/suppliers", "Fournisseur mis à jour")
}

func (h *SupplierHandler) Toggle(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	cur := middlewares.CurrentUser(c)
	by := ""
	if cur != nil {
		by = cur.FullName()
	}
	if err := h.svc.Toggle(c.Request.Context(), uint(id), by); err != nil {
		httputil.RedirectFlash(c, "/suppliers", "Erreur lors de la mise à jour")
		return
	}
	httputil.RedirectFlash(c, "/suppliers", "Statut mis à jour")
}

func (h *SupplierHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.svc.Delete(c.Request.Context(), uint(id)); err != nil {
		httputil.RedirectFlash(c, "/suppliers", "Erreur lors de la suppression")
		return
	}
	httputil.RedirectFlash(c, "/suppliers", "Fournisseur supprimé")
}
