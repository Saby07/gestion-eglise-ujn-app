package models

import (
	"strings"
	"time"
)

type CreatorRole string

const (
	CreatorStaff CreatorRole = "staff"
	CreatorAdmin CreatorRole = "admin"
)

type RequisitionStep string

const (
	StepPendingAccountant   RequisitionStep = "pending_accountant"
	StepPendingAdmin        RequisitionStep = "pending_admin"
	StepPendingSuperAdmin   RequisitionStep = "pending_super_admin"
	StepPendingDisbursement RequisitionStep = "pending_disbursement"
	StepDisbursed           RequisitionStep = "disbursed"
	StepCancelled           RequisitionStep = "cancelled"
)

// PendingStepFor returns the current_step value for a validation role.
func PendingStepFor(role RoleKey) RequisitionStep {
	return RequisitionStep("pending_" + string(role))
}

// RoleFromPendingStep extracts the role key from a pending_* step.
func RoleFromPendingStep(step RequisitionStep) (RoleKey, bool) {
	s := string(step)
	if step == StepPendingDisbursement || !strings.HasPrefix(s, "pending_") {
		return "", false
	}
	role := RoleKey(strings.TrimPrefix(s, "pending_"))
	if !role.Valid() {
		return "", false
	}
	return role, true
}

type RequisitionStatus string

const (
	ReqStatusDraft     RequisitionStatus = "draft"
	ReqStatusOpen      RequisitionStatus = "open"
	ReqStatusCancelled RequisitionStatus = "cancelled"
	ReqStatusCompleted RequisitionStatus = "completed"
)

type Requisition struct {
	BaseModel
	Title           string            `gorm:"type:varchar(255);not null"`
	RequisitionDate time.Time         `gorm:"not null;index"`
	TotalAmount     float64           `gorm:"type:decimal(15,2);not null;default:0"`
	CreatorRole     CreatorRole       `gorm:"type:varchar(16);not null"`
	CurrentStep     RequisitionStep   `gorm:"type:varchar(32);not null;index"`
	Status          RequisitionStatus `gorm:"type:varchar(16);not null;default:'open';index"`
	UserID          uint              `gorm:"not null;index"`
	User            User              `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	AccountID       *uint             `gorm:"index"`
	Account         *ChurchAccount    `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	CategoryID      *uint             `gorm:"index"`
	Category        *ExpenseCategory  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	SupplierID      *uint             `gorm:"index"`
	Supplier        *Supplier         `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`

	ReturnReason string     `gorm:"type:text"`
	ReturnedAt   *time.Time `gorm:"index"`
	ReturnedBy   *uint      `gorm:"index"`

	// Justifications fournisseur (optionnelles)
	InvoicePath       string `gorm:"type:varchar(500)"`
	DeliveryNotePath  string `gorm:"type:varchar(500)"`
	PurchaseOrderPath string `gorm:"type:varchar(500)"`
	ReceptionNotePath string `gorm:"type:varchar(500)"`
	SupplierName      string `gorm:"type:varchar(191)"`
	SupplierPhone     string `gorm:"type:varchar(64)"`
	SupplierAddress   string `gorm:"type:varchar(255)"`

	CancelReason string     `gorm:"type:text"`
	CancelledAt  *time.Time `gorm:"index"`
	CancelledBy  *uint      `gorm:"index"`

	Items        []RequisitionItem       `gorm:"foreignKey:RequisitionID"`
	Validations  []RequisitionValidation `gorm:"foreignKey:RequisitionID"`
	Disbursement *Disbursement           `gorm:"foreignKey:RequisitionID"`

	// ActiveChain is set at runtime for NextValidationStep (not persisted).
	ActiveChain []ValidationStepKey `gorm:"-"`
}

func (Requisition) TableName() string { return "requisitions" }

type RequisitionItem struct {
	BaseModel
	RequisitionID uint    `gorm:"not null;index"`
	Designation   string  `gorm:"type:varchar(255);not null"`
	Quantity      float64 `gorm:"type:decimal(12,2);not null"`
	UnitPrice     float64 `gorm:"type:decimal(15,2);not null"`
	TotalPrice    float64 `gorm:"type:decimal(15,2);not null"`
}

func (RequisitionItem) TableName() string { return "requisition_items" }

type ValidationStepKey string

const (
	ValAccountant ValidationStepKey = "accountant"
	ValAdmin      ValidationStepKey = "admin"
	ValSuperAdmin ValidationStepKey = "super_admin"
)

type RequisitionValidation struct {
	BaseModel
	RequisitionID uint              `gorm:"not null;index"`
	Step          ValidationStepKey `gorm:"type:varchar(32);not null;index"`
	ValidatedByID uint              `gorm:"not null"`
	ValidatedBy   User              `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	ValidatedAt   time.Time         `gorm:"not null"`
	Comment       string            `gorm:"type:text"`
	SignaturePath string            `gorm:"type:varchar(500)"`
}

func (RequisitionValidation) TableName() string { return "requisition_validations" }

// WorkflowSteps returns the configured chain if set, else legacy creator-based chain.
func (r *Requisition) WorkflowSteps() []ValidationStepKey {
	if len(r.ActiveChain) > 0 {
		return r.ActiveChain
	}
	return LegacyWorkflowSteps(r.CreatorRole)
}

// LegacyWorkflowSteps keeps old behaviour as fallback when ActiveChain is unset.
func LegacyWorkflowSteps(creator CreatorRole) []ValidationStepKey {
	if creator == CreatorAdmin {
		return []ValidationStepKey{ValAccountant, ValSuperAdmin}
	}
	return []ValidationStepKey{ValAccountant, ValAdmin, ValSuperAdmin}
}

func (r *Requisition) IsValidated(step ValidationStepKey) bool {
	for _, v := range r.Validations {
		if v.Step == step {
			return true
		}
	}
	return false
}

func (r *Requisition) NextValidationStep() ValidationStepKey {
	for _, step := range r.WorkflowSteps() {
		if !r.IsValidated(step) {
			return step
		}
	}
	return ""
}

func (r *Requisition) SetActiveChain(chain []ValidationStepKey) {
	r.ActiveChain = chain
}
