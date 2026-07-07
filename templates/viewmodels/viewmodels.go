package viewmodels

import (
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
	steps := []struct {
		key   models.ValidationStepKey
		label string
	}{
		{models.ValAccountant, "Comptable"},
	}
	if req.CreatorRole == models.CreatorStaff {
		steps = append(steps, struct {
			key   models.ValidationStepKey
			label string
		}{models.ValAdmin, "Admin"})
	}
	steps = append(steps, struct {
		key   models.ValidationStepKey
		label string
	}{models.ValSuperAdmin, "Super Admin"})
	steps = append(steps, struct {
		key   models.ValidationStepKey
		label string
	}{"cashier", "Caissier"})

	out := make([]WorkflowStepVM, 0, len(steps))
	next := req.NextValidationStep()
	for _, s := range steps {
		vm := WorkflowStepVM{Key: string(s.key), Label: s.label}
		if s.key == "cashier" {
			vm.Done = req.CurrentStep == models.StepDisbursed || req.Status == models.ReqStatusCompleted
			vm.Active = req.CurrentStep == models.StepPendingDisbursement
			if req.Disbursement != nil && req.Disbursement.DisbursedBy.ID > 0 {
				vm.Validator = req.Disbursement.DisbursedBy.FullName()
				vm.Date = req.Disbursement.DisbursedAt.Format("02/01/2006 15:04")
			}
		} else {
			for _, v := range req.Validations {
				if v.Step == s.key {
					vm.Done = true
					vm.Validator = v.ValidatedBy.FullName()
					vm.Date = v.ValidatedAt.Format("02/01/2006 15:04")
				}
			}
			if !vm.Done && string(s.key) == string(next) {
				vm.Active = true
			}
		}
		out = append(out, vm)
	}
	return out
}

type UserRow struct {
	ID          uint
	FullName    string
	Email       string
	Phone       string
	Roles       string
	IsActive    bool
	CanDisburse bool
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
			Roles: join(roles), IsActive: u.IsActive, CanDisburse: u.CanDisburse,
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
	ID          uint
	Title       string
	Author      string
	Date        string
	Amount      float64
	Step        string
	StepLabel   string
	Status      string
	StatusClass string
}

func StepLabel(step models.RequisitionStep) string {
	switch step {
	case models.StepPendingAccountant:
		return "En attente Comptable"
	case models.StepPendingAdmin:
		return "En attente Admin"
	case models.StepPendingSuperAdmin:
		return "En attente Super Admin"
	case models.StepPendingDisbursement:
		return "Prête au décaissement"
	case models.StepDisbursed:
		return "Décaissée"
	case models.StepCancelled:
		return "Annulée"
	default:
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
			Step: string(r.CurrentStep), StepLabel: StepLabel(r.CurrentStep),
			Status: string(r.Status), StatusClass: StatusClass(r.Status, r.CurrentStep),
		})
	}
	return rows
}

type ReportRow = services.DisbursementReportRow
