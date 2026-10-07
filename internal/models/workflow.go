package models

// WorkflowCapability describes what a role may do in the requisition lifecycle.
type WorkflowCapability string

const (
	CapCreate     WorkflowCapability = "create"
	CapAttachDocs WorkflowCapability = "attach_docs"
	CapDisburse   WorkflowCapability = "disburse"
	CapFund       WorkflowCapability = "fund"
)

// RoleCapabilities holds per-role flags for the dynamic workflow.
type RoleCapabilities struct {
	Create     bool `json:"create"`
	AttachDocs bool `json:"attach_docs"`
	Disburse   bool `json:"disburse"`
	Fund       bool `json:"fund"`
}

// WorkflowSchema is the Super-Admin-configurable validation / capability schema.
type WorkflowSchema struct {
	Version         int                          `json:"version"`
	Capabilities    map[RoleKey]RoleCapabilities `json:"capabilities"`
	ValidationChain []RoleKey                    `json:"validation_chain"`
}

// IntermediateRoles are roles allowed in validation_chain (not super_admin).
var IntermediateRoles = []RoleKey{RoleStaff, RoleAccountant, RoleAdmin, RoleCashier}

// DefaultWorkflowSchema mirrors the historical behaviour approximately.
func DefaultWorkflowSchema() WorkflowSchema {
	return WorkflowSchema{
		Version: 2,
		Capabilities: map[RoleKey]RoleCapabilities{
			RoleStaff:      {Create: true},
			RoleAccountant: {AttachDocs: true, Fund: true},
			RoleAdmin:      {Create: true},
			RoleCashier:    {Disburse: true},
			RoleSuperAdmin: {Create: true},
		},
		ValidationChain: []RoleKey{RoleAccountant, RoleAdmin},
	}
}

// Has reports whether the role has the given capability.
func (c RoleCapabilities) Has(cap WorkflowCapability) bool {
	switch cap {
	case CapCreate:
		return c.Create
	case CapAttachDocs:
		return c.AttachDocs
	case CapDisburse:
		return c.Disburse
	case CapFund:
		return c.Fund
	default:
		return false
	}
}
