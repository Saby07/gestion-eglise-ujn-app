package viewmodels

import (
	"eglise_ujn/internal/models"
	"eglise_ujn/internal/services"
)

type NotificationVM struct {
	ID            uint
	Message       string
	RequisitionID uint
	IsUnread      bool
	TimeAgo       string
	IconClass     string
	IconBg        string
}

func MapNotifications(list []models.Notification) []NotificationVM {
	out := make([]NotificationVM, 0, len(list))
	for _, n := range list {
		icon, bg := notificationIcon(n.Step)
		out = append(out, NotificationVM{
			ID:            n.ID,
			Message:       n.Message,
			RequisitionID: n.RequisitionID,
			IsUnread:      n.IsUnread(),
			TimeAgo:       services.FormatTimeAgo(n.CreatedAt),
			IconClass:     icon,
			IconBg:        bg,
		})
	}
	return out
}

func notificationIcon(step string) (icon, bg string) {
	switch models.RequisitionStep(step) {
	case models.StepPendingAccountant:
		return "ri-calculator-line", "bg-success-subtle text-success"
	case models.StepPendingAdmin:
		return "ri-shield-user-line", "bg-primary-subtle text-primary"
	case models.StepPendingSuperAdmin:
		return "ri-vip-crown-line", "bg-info-subtle text-info"
	case models.StepPendingDisbursement:
		return "ri-wallet-3-line", "bg-warning-subtle text-warning"
	default:
		return "ri-notification-3-line", "bg-secondary-subtle text-secondary"
	}
}

type LayoutVM struct {
	Title         string
	CSRF          string
	Alert         string
	ActiveMenu    string
	CurrentUserID uint
	UserName      string
	UserRole      string
	ChurchName    string
	DarkMode      bool
	IsSuperAdmin  bool
	IsAdmin       bool
	IsAccountant  bool
	IsCashier     bool
	IsStaff       bool
	Notifications []NotificationVM
	UnreadCount   int
}

func (v LayoutVM) CanManageUsers() bool       { return v.IsSuperAdmin }
func (v LayoutVM) CanManageAccounts() bool    { return v.IsSuperAdmin || v.IsAdmin || v.IsAccountant }
func (v LayoutVM) CanCreateRequisition() bool { return v.IsStaff || v.IsAdmin || v.IsSuperAdmin }
func (v LayoutVM) CanViewReports() bool {
	return v.IsSuperAdmin || v.IsAdmin || v.IsAccountant || v.IsCashier
}

func (v LayoutVM) CanManageCategories() bool { return v.IsSuperAdmin || v.IsAdmin }

func (v LayoutVM) CanManageSuppliers() bool {
	return v.IsSuperAdmin || v.IsAdmin || v.IsAccountant
}

func (v LayoutVM) CanViewAudit() bool { return v.IsSuperAdmin }

func (v LayoutVM) CanManageSettings() bool { return v.IsSuperAdmin }

func (v LayoutVM) IsStaffOnly() bool {
	return v.IsStaff && !v.IsAdmin && !v.IsSuperAdmin && !v.IsAccountant && !v.IsCashier
}
