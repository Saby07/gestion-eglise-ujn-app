package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"eglise_ujn/internal/httputil"
	"eglise_ujn/internal/middlewares"
	"eglise_ujn/internal/models"
	"eglise_ujn/internal/services"
	"eglise_ujn/internal/upload"
	"eglise_ujn/templates/pages"
	"eglise_ujn/templates/viewmodels"

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
	_, err := h.svc.Create(c.Request.Context(),
		strings.TrimSpace(c.PostForm("name")),
		strings.TrimSpace(c.PostForm("description")),
		strings.TrimSpace(c.PostForm("currency")),
		cur.FullName(),
	)
	if err != nil {
		httputil.RedirectFlash(c, "/accounts/new", "Erreur lors de la création")
		return
	}
	httputil.RedirectFlash(c, "/accounts", "Compte créé")
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
		httputil.RedirectFlash(c, "/accounts/"+c.Param("id"), "Erreur lors de l'approvisionnement")
		return
	}
	httputil.RedirectFlash(c, "/accounts/"+c.Param("id"), "Compte approvisionné")
}

type RequisitionsHandler struct {
	svc         *services.RequisitionService
	accountSvc  *services.AccountService
	categorySvc *services.CategoryService
	supplierSvc *services.SupplierService
	auditSvc    *services.AuditService
}

func NewRequisitionsHandler(
	svc *services.RequisitionService,
	accountSvc *services.AccountService,
	categorySvc *services.CategoryService,
	supplierSvc *services.SupplierService,
	auditSvc *services.AuditService,
) *RequisitionsHandler {
	return &RequisitionsHandler{
		svc: svc, accountSvc: accountSvc,
		categorySvc: categorySvc, supplierSvc: supplierSvc, auditSvc: auditSvc,
	}
}

func (h *RequisitionsHandler) List(c *gin.Context) {
	user := middlewares.CurrentUser(c)
	filter := c.DefaultQuery("filter", "all")
	searchFilter := parseSearchFilter(c)
	var list []models.Requisition
	var err error
	if hasSearchQuery(c) {
		list, err = h.svc.ListWithSearch(c.Request.Context(), searchFilter, user)
	} else {
		list, err = h.svc.List(c.Request.Context(), filter, user)
	}
	if err != nil {
		list = nil
	}
	categories, _ := h.categorySvc.List(c.Request.Context(), true)
	search := viewmodels.RequisitionSearchVM{
		Q:          searchFilter.Q,
		From:       strings.TrimSpace(c.Query("from")),
		To:         strings.TrimSpace(c.Query("to")),
		CategoryID: strings.TrimSpace(c.Query("category_id")),
	}
	httputil.Render(c, http.StatusOK, pages.RequisitionsList(layoutFromCtx(c, "Réquisitions", "requisitions"), list, filter, search, mapCategories(categories)))
}

func hasSearchQuery(c *gin.Context) bool {
	return strings.TrimSpace(c.Query("q")) != "" ||
		strings.TrimSpace(c.Query("from")) != "" ||
		strings.TrimSpace(c.Query("to")) != "" ||
		strings.TrimSpace(c.Query("category_id")) != "" ||
		strings.TrimSpace(c.Query("min_amount")) != "" ||
		strings.TrimSpace(c.Query("max_amount")) != ""
}

func (h *RequisitionsHandler) NewPage(c *gin.Context) {
	accounts, _ := h.accountSvc.List(c.Request.Context())
	categories, _ := h.categorySvc.List(c.Request.Context(), true)
	suppliers, _ := h.supplierSvc.List(c.Request.Context(), true)
	httputil.Render(c, http.StatusOK, pages.RequisitionForm(
		layoutFromCtx(c, "Nouvelle réquisition", "requisitions"),
		accounts, mapCategories(categories), mapSuppliers(suppliers),
	))
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
	categoryID := parseOptionalUint(c.PostForm("category_id"))
	accountID := parseOptionalUint(c.PostForm("account_id"))
	supplierID := parseOptionalUint(c.PostForm("supplier_id"))
	_, err = h.svc.Create(c.Request.Context(), user, title, date, items, categoryID, accountID, supplierID)
	if err != nil {
		httputil.RedirectFlash(c, "/requisitions/new", "Erreur lors de la création")
		return
	}
	httputil.RedirectFlash(c, "/requisitions", "Réquisition créée")
}

func (h *RequisitionsHandler) EditPage(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	user := middlewares.CurrentUser(c)
	req, err := h.svc.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		httputil.RedirectFlash(c, "/requisitions", "Réquisition introuvable")
		return
	}
	if !h.svc.CanEdit(user, req) {
		httputil.RedirectFlash(c, "/requisitions/"+c.Param("id"), "Modification non autorisée")
		return
	}
	accounts, _ := h.accountSvc.List(c.Request.Context())
	categories, _ := h.categorySvc.List(c.Request.Context(), true)
	suppliers, _ := h.supplierSvc.List(c.Request.Context(), true)
	httputil.Render(c, http.StatusOK, pages.RequisitionEditForm(
		layoutFromCtx(c, "Modifier réquisition", "requisitions"),
		req, accounts, mapCategories(categories), mapSuppliers(suppliers),
	))
}

func (h *RequisitionsHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	user := middlewares.CurrentUser(c)
	title := strings.TrimSpace(c.PostForm("title"))
	dateStr := strings.TrimSpace(c.PostForm("date"))
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		date = time.Now()
	}
	items := parseItems(c)
	categoryID := parseOptionalUint(c.PostForm("category_id"))
	accountID := parseOptionalUint(c.PostForm("account_id"))
	supplierID := parseOptionalUint(c.PostForm("supplier_id"))
	err = h.svc.Update(c.Request.Context(), uint(id), user, title, date, items, categoryID, accountID, supplierID)
	if err != nil {
		httputil.RedirectFlash(c, "/requisitions/"+c.Param("id")+"/edit", requisitionErrMsg(err))
		return
	}
	httputil.RedirectFlash(c, "/requisitions/"+c.Param("id"), "Réquisition mise à jour")
}

func (h *RequisitionsHandler) Return(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	user := middlewares.CurrentUser(c)
	err := h.svc.ReturnForCorrection(c.Request.Context(), uint(id), user, strings.TrimSpace(c.PostForm("reason")))
	if err != nil {
		httputil.RedirectFlash(c, "/requisitions/"+c.Param("id"), requisitionErrMsg(err))
		return
	}
	httputil.RedirectFlash(c, "/requisitions/"+c.Param("id"), "Réquisition retournée pour correction")
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
		httputil.RedirectFlash(c, "/requisitions", "Accès non autorisé")
		return
	}
	accounts, _ := h.accountSvc.List(c.Request.Context())
	canValidate := canUserValidate(user, req)
	canCancel := h.svc.CanCancel(user, req)
	canDisburse := canUserDisburse(user, req)
	canEdit := h.svc.CanEdit(user, req)
	var projected float64
	balanceInsufficient := false
	if req.AccountID != nil {
		_, projected, _ = h.accountSvc.ProjectedBalance(c.Request.Context(), *req.AccountID)
		balanceInsufficient = projected < req.TotalAmount
	}
	httputil.Render(c, http.StatusOK, pages.RequisitionDetail(layoutFromCtx(c, req.Title, "requisitions"), req, user, accounts, canValidate, canCancel, canDisburse, canEdit, projected, balanceInsufficient))
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

func requisitionErrMsg(err error) string {
	if errors.Is(err, services.ErrInsufficientBalance) {
		return "Solde insuffisant pour effectuer cette opération"
	}
	return "Opération impossible"
}

func (h *RequisitionsHandler) Validate(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	user := middlewares.CurrentUser(c)
	sigPath, _ := parseSignature(c)
	extras := &services.AccountantExtras{SignaturePath: sigPath}
	if user.HasRole(models.RoleAccountant) {
		inv, _ := upload.SaveFormFile(c, "invoice", "invoice")
		dn, _ := upload.SaveFormFile(c, "delivery_note", "delivery")
		po, _ := upload.SaveFormFile(c, "purchase_order", "po")
		rn, _ := upload.SaveFormFile(c, "reception_note", "reception")
		extras.InvoicePath = inv
		extras.DeliveryNotePath = dn
		extras.PurchaseOrderPath = po
		extras.ReceptionNotePath = rn
		extras.SupplierName = strings.TrimSpace(c.PostForm("supplier_name"))
		extras.SupplierPhone = strings.TrimSpace(c.PostForm("supplier_phone"))
		extras.SupplierAddress = strings.TrimSpace(c.PostForm("supplier_address"))
	}
	err := h.svc.Validate(c.Request.Context(), uint(id), user, strings.TrimSpace(c.PostForm("comment")), extras)
	if err != nil {
		httputil.RedirectFlash(c, "/requisitions/"+c.Param("id"), requisitionErrMsg(err))
		return
	}
	httputil.RedirectFlash(c, "/requisitions/"+c.Param("id"), "Réquisition validée")
}

func (h *RequisitionsHandler) Cancel(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	user := middlewares.CurrentUser(c)
	err := h.svc.Cancel(c.Request.Context(), uint(id), user, strings.TrimSpace(c.PostForm("reason")))
	if err != nil {
		httputil.RedirectFlash(c, "/requisitions/"+c.Param("id"), requisitionErrMsg(err))
		return
	}
	httputil.RedirectFlash(c, "/requisitions", "Réquisition annulée")
}

func (h *RequisitionsHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	user := middlewares.CurrentUser(c)
	if err := h.svc.Delete(c.Request.Context(), uint(id), user); err != nil {
		httputil.RedirectFlash(c, "/requisitions", requisitionErrMsg(err))
		return
	}
	httputil.RedirectFlash(c, "/requisitions", "Réquisition supprimée")
}

func (h *RequisitionsHandler) Disburse(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	user := middlewares.CurrentUser(c)
	accountID, _ := strconv.ParseUint(c.PostForm("account_id"), 10, 64)
	mode := models.PaymentMode(c.PostForm("payment_mode"))
	if !mode.Valid() {
		httputil.RedirectFlash(c, "/requisitions/"+c.Param("id"), "Mode de paiement invalide")
		return
	}
	receipt, _ := upload.SaveFormFile(c, "receipt", "receipt")
	req, _ := h.svc.GetByID(c.Request.Context(), uint(id))
	err := h.svc.Disburse(c.Request.Context(), uint(id), user, uint(accountID), mode, receipt, strings.TrimSpace(c.PostForm("note")), h.accountSvc)
	if err != nil {
		httputil.RedirectFlash(c, "/requisitions/"+c.Param("id"), requisitionErrMsg(err))
		return
	}
	msg := "Réquisition décaissée"
	if req != nil {
		if warning, _ := h.accountSvc.CheckDisbursementBalance(c.Request.Context(), uint(accountID), req.TotalAmount); warning != "" {
			msg = msg + " — " + warning
		}
	}
	httputil.RedirectFlash(c, "/requisitions/"+c.Param("id"), msg)
}

func parseSignature(c *gin.Context) (string, error) {
	if path, err := upload.SaveFormFile(c, "signature_file", "signature"); err != nil {
		return "", err
	} else if path != "" {
		return path, nil
	}
	return upload.SaveBase64Image(c.PostForm("signature"), "signature")
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
