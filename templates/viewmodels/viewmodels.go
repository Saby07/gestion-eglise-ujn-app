package viewmodels

import (
	"encoding/json"

	"eglise_ujn/internal/models"
	"eglise_ujn/internal/services"
)

type LayoutData struct {
	Title      string
	User       *models.User
	CSRF       string
	Alert      string
	ActiveMenu string
}

type WorkflowStepVM struct {
	Key       string
	Label     string
	Done      bool
	Active    bool
	Validator string
	Date      string
}

func BuildWorkflowSteps(req *models.Requisition) []WorkflowStepVM {
	chain := req.WorkflowSteps()
	out := make([]WorkflowStepVM, 0, len(chain)+1)
	next := req.NextValidationStep()
	for _, step := range chain {
		vm := WorkflowStepVM{
			Key:   string(step),
			Label: models.RoleKey(step).Label(),
		}
		for _, v := range req.Validations {
			if v.Step == step {
				vm.Done = true
				vm.Validator = v.ValidatedBy.FullName()
				vm.Date = v.ValidatedAt.Format("02/01/2006 15:04")
			}
		}
		if !vm.Done && step == next {
			vm.Active = true
		}
		out = append(out, vm)
	}
	disburse := WorkflowStepVM{Key: "disburse", Label: "Décaissement"}
	disburse.Done = req.CurrentStep == models.StepDisbursed || req.Status == models.ReqStatusCompleted
	disburse.Active = req.CurrentStep == models.StepPendingDisbursement
	if req.Disbursement != nil && req.Disbursement.DisbursedBy.ID > 0 {
		disburse.Validator = req.Disbursement.DisbursedBy.FullName()
		disburse.Date = req.Disbursement.DisbursedAt.Format("02/01/2006 15:04")
	}
	out = append(out, disburse)
	return out
}

type UserRow struct {
	ID       uint
	FullName string
	Email    string
	Phone    string
	Roles    string
	IsActive bool
}

func MapUsers(users []models.User) []UserRow {
	rows := make([]UserRow, 0, len(users))
	for _, u := range users {
		roles := make([]string, 0, len(u.Roles))
		for _, ur := range u.Roles {
			roles = append(roles, ur.Role.Name)
		}
		rows = append(rows, UserRow{
			ID: u.ID, FullName: u.FullName(), Email: u.Email, Phone: u.Phone,
			Roles: join(roles), IsActive: u.IsActive,
		})
	}
	return rows
}

func join(parts []string) string {
	if len(parts) == 0 {
		return "—"
	}
	s := parts[0]
	for i := 1; i < len(parts); i++ {
		s += ", " + parts[i]
	}
	return s
}

type AccountRow struct {
	ID       uint
	Name     string
	Balance  float64
	Currency string
	Active   bool
}

type RequisitionRow struct {
	ID           uint
	Title        string
	Author       string
	Date         string
	Amount       float64
	CategoryName string
	Step         string
	StepLabel    string
	Status       string
	StatusClass  string
}

type RequisitionSearchVM struct {
	Q          string
	From       string
	To         string
	CategoryID string
}

type TimelineEventVM struct {
	Icon        string
	IconBg      string
	Title       string
	Description string
	Actor       string
	Date        string
	Done        bool
}

func BuildTimeline(req *models.Requisition) []TimelineEventVM {
	if req == nil {
		return nil
	}
	events := []TimelineEventVM{
		{
			Icon: "ri-file-add-line", IconBg: "bg-primary-subtle text-primary",
			Title: "Création", Description: req.Title,
			Actor: req.User.FullName(), Date: req.CreatedAt.Format("02/01/2006 15:04"), Done: true,
		},
	}
	for _, v := range req.Validations {
		label := validationStepLabel(v.Step)
		events = append(events, TimelineEventVM{
			Icon: "ri-checkbox-circle-line", IconBg: "bg-success-subtle text-success",
			Title: "Validation — " + label, Description: v.Comment,
			Actor: v.ValidatedBy.FullName(), Date: v.ValidatedAt.Format("02/01/2006 15:04"), Done: true,
		})
	}
	if req.CurrentStep == models.StepPendingDisbursement || req.Disbursement != nil {
		active := req.Disbursement == nil
		ev := TimelineEventVM{
			Icon: "ri-wallet-3-line", IconBg: "bg-info-subtle text-info",
			Title: "En attente de décaissement", Done: !active,
		}
		if req.Disbursement != nil {
			ev.Title = "Décaissement effectué"
			ev.Actor = req.Disbursement.DisbursedBy.FullName()
			ev.Date = req.Disbursement.DisbursedAt.Format("02/01/2006 15:04")
			ev.Done = true
		}
		events = append(events, ev)
	}
	if req.Status == models.ReqStatusCancelled {
		date := ""
		if req.CancelledAt != nil {
			date = req.CancelledAt.Format("02/01/2006 15:04")
		}
		events = append(events, TimelineEventVM{
			Icon: "ri-close-circle-line", IconBg: "bg-danger-subtle text-danger",
			Title: "Annulation", Description: req.CancelReason, Date: date, Done: true,
		})
	}
	return events
}

func validationStepLabel(step models.ValidationStepKey) string {
	role := models.RoleKey(step)
	if role.Valid() {
		return role.Label()
	}
	return string(step)
}

func StepLabel(step models.RequisitionStep) string {
	switch step {
	case models.StepPendingDisbursement:
		return "Prête au décaissement"
	case models.StepDisbursed:
		return "Décaissée"
	case models.StepCancelled:
		return "Annulée"
	default:
		if role, ok := models.RoleFromPendingStep(step); ok {
			return "En attente " + role.Label()
		}
		return string(step)
	}
}

func WorkflowStepClass(key string) string {
	switch key {
	case string(models.ValAccountant):
		return "workflow-step--accountant"
	case string(models.ValAdmin):
		return "workflow-step--admin"
	case string(models.ValSuperAdmin):
		return "workflow-step--super-admin"
	case "cashier":
		return "workflow-step--cashier"
	default:
		return "workflow-step--default"
	}
}

func WorkflowStepIcon(key string) string {
	switch key {
	case string(models.ValAccountant):
		return "ri-calculator-line"
	case string(models.ValAdmin):
		return "ri-shield-user-line"
	case string(models.ValSuperAdmin):
		return "ri-vip-crown-line"
	case "cashier":
		return "ri-wallet-3-line"
	default:
		return "ri-checkbox-circle-line"
	}
}

func StatusClass(status models.RequisitionStatus, step models.RequisitionStep) string {
	if status == models.ReqStatusCancelled {
		return "bg-danger"
	}
	if status == models.ReqStatusCompleted {
		return "bg-success"
	}
	if step == models.StepPendingDisbursement {
		return "bg-info"
	}
	return "bg-warning"
}

func MapRequisitions(list []models.Requisition) []RequisitionRow {
	rows := make([]RequisitionRow, 0, len(list))
	for _, r := range list {
		rows = append(rows, RequisitionRow{
			ID: r.ID, Title: r.Title, Author: r.User.FullName(),
			Date: r.RequisitionDate.Format("02/01/2006"), Amount: r.TotalAmount,
			CategoryName: requisitionCategoryName(r),
			Step:         string(r.CurrentStep), StepLabel: StepLabel(r.CurrentStep),
			Status: string(r.Status), StatusClass: StatusClass(r.Status, r.CurrentStep),
		})
	}
	return rows
}

func requisitionCategoryName(r models.Requisition) string {
	// Placeholder until Category relation is wired; handlers may override via MapRequisitions.
	return ""
}

type ReportRow = services.DisbursementReportRow

type AuditLogRow struct {
	ID        uint
	UserName  string
	Action    string
	Entity    string
	Details   string
	IP        string
	CreatedAt string
}

type CategoryVM struct {
	ID          uint
	Name        string
	Description string
	Active      bool
}

type BudgetVM struct {
	ID           uint
	Year         int
	CategoryName string
	Amount       float64
	Spent        float64
	UsagePercent float64
}

func BudgetUsageClass(pct float64) string {
	if pct >= 90 {
		return "bg-danger"
	}
	if pct >= 75 {
		return "bg-warning"
	}
	return "bg-success"
}

type SupplierVM struct {
	ID      uint
	Name    string
	Phone   string
	Email   string
	Address string
	Active  bool
}

type SettingsVM struct {
	ChurchName      string
	DarkMode        bool
	SMTPHost        string
	SMTPPort        string
	SMTPUser        string
	SMTPFrom        string
	ReminderDays    int
	WorkflowRoles   []WorkflowRoleCapVM
	WorkflowChain   []WorkflowChainItemVM
	WorkflowPreview []string
}

type WorkflowRoleCapVM struct {
	Key        string
	Label      string
	Create     bool
	AttachDocs bool
	Disburse   bool
	Fund       bool
	IsSuper    bool
}

type WorkflowChainItemVM struct {
	Key      string
	Label    string
	Selected bool
	Order    int
}

type StatsVM struct {
	CategoryLabels        []string
	CategoryValues        []float64
	MonthlyLabels         []string
	MonthlyValues         []float64
	ValidationDelayLabels []string
	ValidationDelayValues []float64
}

func StatsJSONArray(labels []string) string {
	if len(labels) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(labels)
	return string(b)
}

func StatsJSONFloats(values []float64) string {
	if len(values) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(values)
	return string(b)
}
