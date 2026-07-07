package handlers

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"eglise_ujn/internal/httputil"
	"eglise_ujn/internal/middlewares"
	"eglise_ujn/internal/models"
	"eglise_ujn/internal/services"
	"eglise_ujn/internal/upload"
	"eglise_ujn/templates/pages"

	"github.com/gin-gonic/gin"
)

type AccountsHandler struct {
	svc *services.AccountService
}

func NewAccountsHandler(svc *services.AccountService) *AccountsHandler {
	return &AccountsHandler{svc: svc}
}

func (h *AccountsHandler) List(c *gin.Context) {
	list, _ := h.svc.List(c.Request.Context())
	httputil.Render(c, http.StatusOK, pages.AccountsList(layoutFromCtx(c, "Comptes", "accounts"), list))
}

func (h *AccountsHandler) NewPage(c *gin.Context) {
	httputil.Render(c, http.StatusOK, pages.AccountForm(layoutFromCtx(c, "Nouveau compte", "accounts")))
}

func (h *AccountsHandler) Create(c *gin.Context) {
	cur := middlewares.CurrentUser(c)
	amount, _ := strconv.ParseFloat(strings.TrimSpace(c.PostForm("initial_amount")), 64)
	acc, err := h.svc.Create(c.Request.Context(),
		strings.TrimSpace(c.PostForm("name")),
		strings.TrimSpace(c.PostForm("description")),
		strings.TrimSpace(c.PostForm("currency")),
		cur.FullName(),
	)
	if err != nil {
		c.Redirect(http.StatusFound, "/accounts/new?alert=Erreur")
		return
	}
	if amount > 0 {
		_ = h.svc.Fund(c.Request.Context(), acc.ID, amount, "Approvisionnement initial", cur.ID, cur.FullName())
	}
	c.Redirect(http.StatusFound, "/accounts?alert=Compte+cree")
}

func (h *AccountsHandler) Detail(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	acc, err := h.svc.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.Redirect(http.StatusFound, "/accounts")
		return
	}
	hist, _ := h.svc.History(c.Request.Context(), uint(id))
	bal, projected, _ := h.svc.ProjectedBalance(c.Request.Context(), uint(id))
	httputil.Render(c, http.StatusOK, pages.AccountDetail(layoutFromCtx(c, acc.Name, "accounts"), acc, hist, bal, projected))
}

func (h *AccountsHandler) Fund(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	amount, _ := strconv.ParseFloat(strings.TrimSpace(c.PostForm("amount")), 64)
	cur := middlewares.CurrentUser(c)
	err := h.svc.Fund(c.Request.Context(), uint(id), amount, strings.TrimSpace(c.PostForm("label")), cur.ID, cur.FullName())
	if err != nil {
		c.Redirect(http.StatusFound, "/accounts/"+c.Param("id")+"?alert=Erreur+approvisionnement")
		return
	}
	c.Redirect(http.StatusFound, "/accounts/"+c.Param("id")+"?alert=Compte+approvisionne")
}

type RequisitionsHandler struct {
	svc        *services.RequisitionService
	accountSvc *services.AccountService
}

func NewRequisitionsHandler(svc *services.RequisitionService, accountSvc *services.AccountService) *RequisitionsHandler {
	return &RequisitionsHandler{svc: svc, accountSvc: accountSvc}
}

func (h *RequisitionsHandler) List(c *gin.Context) {
	user := middlewares.CurrentUser(c)
	filter := c.DefaultQuery("filter", "all")
	list, _ := h.svc.List(c.Request.Context(), filter, user)
	httputil.Render(c, http.StatusOK, pages.RequisitionsList(layoutFromCtx(c, "Réquisitions", "requisitions"), list, filter))
}

func (h *RequisitionsHandler) NewPage(c *gin.Context) {
	httputil.Render(c, http.StatusOK, pages.RequisitionForm(layoutFromCtx(c, "Nouvelle réquisition", "requisitions")))
}

func (h *RequisitionsHandler) Create(c *gin.Context) {
	user := middlewares.CurrentUser(c)
	title := strings.TrimSpace(c.PostForm("title"))
	dateStr := strings.TrimSpace(c.PostForm("date"))
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		date = time.Now()
	}
	items := parseItems(c)
	_, err = h.svc.Create(c.Request.Context(), user, title, date, items, nil)
	if err != nil {
		c.Redirect(http.StatusFound, "/requisitions/new?alert="+err.Error())
		return
	}
	c.Redirect(http.StatusFound, "/requisitions?alert=Requisition+creee")
}

func parseItems(c *gin.Context) []services.RequisitionItemInput {
	designations := c.PostFormArray("designation")
	quantities := c.PostFormArray("quantity")
	prices := c.PostFormArray("unit_price")
	n := len(designations)
	if len(quantities) < n {
		n = len(quantities)
	}
	if len(prices) < n {
		n = len(prices)
	}
	out := make([]services.RequisitionItemInput, 0, n)
	for i := 0; i < n; i++ {
		q, _ := strconv.ParseFloat(strings.TrimSpace(quantities[i]), 64)
		p, _ := strconv.ParseFloat(strings.TrimSpace(prices[i]), 64)
		out = append(out, services.RequisitionItemInput{
			Designation: strings.TrimSpace(designations[i]),
			Quantity:    q,
			UnitPrice:   p,
		})
	}
	return out
}

func (h *RequisitionsHandler) Detail(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	req, err := h.svc.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.Redirect(http.StatusFound, "/requisitions")
		return
	}
	user := middlewares.CurrentUser(c)
	if !h.svc.CanView(user, req) {
		c.Redirect(http.StatusFound, "/requisitions?alert="+url.QueryEscape("Accès non autorisé"))
		return
	}
	accounts, _ := h.accountSvc.List(c.Request.Context())
	canValidate := canUserValidate(user, req)
	canCancel := h.svc.CanCancel(user, req)
	canDisburse := canUserDisburse(user, req)
	httputil.Render(c, http.StatusOK, pages.RequisitionDetail(layoutFromCtx(c, req.Title, "requisitions"), req, user, accounts, canValidate, canCancel, canDisburse))
}

func canUserValidate(user *models.User, req *models.Requisition) bool {
	step := req.NextValidationStep()
	if step == "" {
		return false
	}
	switch step {
	case models.ValAccountant:
		return user.HasRole(models.RoleAccountant)
	case models.ValAdmin:
		return user.HasRole(models.RoleAdmin) || user.HasRole(models.RoleSuperAdmin)
	case models.ValSuperAdmin:
		return user.HasRole(models.RoleSuperAdmin)
	}
	return false
}

func canUserDisburse(user *models.User, req *models.Requisition) bool {
	if req.CurrentStep != models.StepPendingDisbursement {
		return false
	}
	if user.HasRole(models.RoleCashier) {
		return true
	}
	return user.HasRole(models.RoleAccountant) && user.CanDisburse
}

func (h *RequisitionsHandler) Validate(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	user := middlewares.CurrentUser(c)
	var extras *services.AccountantExtras
	if user.HasRole(models.RoleAccountant) {
		inv, _ := upload.SaveFormFile(c, "invoice", "invoice")
		dn, _ := upload.SaveFormFile(c, "delivery_note", "delivery")
		po, _ := upload.SaveFormFile(c, "purchase_order", "po")
		rn, _ := upload.SaveFormFile(c, "reception_note", "reception")
		extras = &services.AccountantExtras{
			InvoicePath: inv, DeliveryNotePath: dn, PurchaseOrderPath: po, ReceptionNotePath: rn,
			SupplierName:    strings.TrimSpace(c.PostForm("supplier_name")),
			SupplierPhone:   strings.TrimSpace(c.PostForm("supplier_phone")),
			SupplierAddress: strings.TrimSpace(c.PostForm("supplier_address")),
		}
	}
	err := h.svc.Validate(c.Request.Context(), uint(id), user, strings.TrimSpace(c.PostForm("comment")), extras)
	if err != nil {
		c.Redirect(http.StatusFound, "/requisitions/"+c.Param("id")+"?alert="+url.QueryEscape(err.Error()))
		return
	}
	c.Redirect(http.StatusFound, "/requisitions/"+c.Param("id")+"?alert="+url.QueryEscape("Réquisition validée"))
}

func (h *RequisitionsHandler) Cancel(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	user := middlewares.CurrentUser(c)
	err := h.svc.Cancel(c.Request.Context(), uint(id), user, strings.TrimSpace(c.PostForm("reason")))
	if err != nil {
		c.Redirect(http.StatusFound, "/requisitions/"+c.Param("id")+"?alert="+err.Error())
		return
	}
	c.Redirect(http.StatusFound, "/requisitions?alert=Annulee")
}

func (h *RequisitionsHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	user := middlewares.CurrentUser(c)
	_ = h.svc.Delete(c.Request.Context(), uint(id), user)
	c.Redirect(http.StatusFound, "/requisitions?alert=Supprimee")
}

func (h *RequisitionsHandler) Disburse(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	user := middlewares.CurrentUser(c)
	accountID, _ := strconv.ParseUint(c.PostForm("account_id"), 10, 64)
	mode := models.PaymentMode(c.PostForm("payment_mode"))
	receipt, _ := upload.SaveFormFile(c, "receipt", "receipt")
	err := h.svc.Disburse(c.Request.Context(), uint(id), user, uint(accountID), mode, receipt, strings.TrimSpace(c.PostForm("note")), h.accountSvc)
	if err != nil {
		c.Redirect(http.StatusFound, "/requisitions/"+c.Param("id")+"?alert="+err.Error())
		return
	}
	c.Redirect(http.StatusFound, "/requisitions/"+c.Param("id")+"?alert=Decaissee")
}

type ReportsHandler struct {
	svc *services.ReportService
}

func NewReportsHandler(svc *services.ReportService) *ReportsHandler {
	return &ReportsHandler{svc: svc}
}

func (h *ReportsHandler) Index(c *gin.Context) {
	period := c.DefaultQuery("period", "daily")
	from, to, label := periodRange(period)
	rows, total, _ := h.svc.Disbursements(c.Request.Context(), from, to)
	httputil.Render(c, http.StatusOK, pages.Reports(layoutFromCtx(c, "Rapports", "reports"), period, label, rows, total))
}

func periodRange(period string) (time.Time, time.Time, string) {
	now := time.Now()
	loc := now.Location()
	startDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	switch period {
	case "weekly":
		from := startDay.AddDate(0, 0, -int(now.Weekday()))
		to := from.AddDate(0, 0, 7)
		return from, to, "Hebdomadaire"
	case "monthly":
		from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		return from, from.AddDate(0, 1, 0), "Mensuel"
	case "yearly":
		from := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, loc)
		return from, from.AddDate(1, 0, 0), "Annuel"
	default:
		to := startDay.AddDate(0, 0, 1)
		return startDay, to, "Journalier"
	}
}
