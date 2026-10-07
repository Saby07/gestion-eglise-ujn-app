package services

import (
	"testing"

	"eglise_ujn/internal/models"
)

func TestDefaultSchemaValid(t *testing.T) {
	schema := models.DefaultWorkflowSchema()
	if err := validateSchema(normalizeSchema(schema)); err != nil {
		t.Fatal(err)
	}
	chain := EffectiveChain(schema)
	if len(chain) < 2 {
		t.Fatalf("chain too short: %v", chain)
	}
	if chain[len(chain)-1] != models.ValSuperAdmin {
		t.Fatalf("super_admin must be last, got %v", chain)
	}
}

func TestValidateSchemaRequiresCreateAndDisburse(t *testing.T) {
	schema := models.DefaultWorkflowSchema()
	for k, v := range schema.Capabilities {
		v.Create = false
		schema.Capabilities[k] = v
	}
	if err := validateSchema(normalizeSchema(schema)); err == nil {
		t.Fatal("expected error when no create")
	}

	schema = models.DefaultWorkflowSchema()
	for k, v := range schema.Capabilities {
		v.Disburse = false
		schema.Capabilities[k] = v
	}
	if err := validateSchema(normalizeSchema(schema)); err == nil {
		t.Fatal("expected error when no disburse")
	}
}

func TestNormalizeDropsSuperAdminFromChain(t *testing.T) {
	schema := models.DefaultWorkflowSchema()
	schema.ValidationChain = []models.RoleKey{models.RoleAccountant, models.RoleSuperAdmin, models.RoleAdmin}
	schema = normalizeSchema(schema)
	for _, r := range schema.ValidationChain {
		if r == models.RoleSuperAdmin {
			t.Fatal("super_admin should be stripped from intermediate chain")
		}
	}
	chain := EffectiveChain(schema)
	if chain[len(chain)-1] != models.ValSuperAdmin {
		t.Fatal("effective chain must end with super_admin")
	}
}

func TestUserHasCapability(t *testing.T) {
	schema := models.DefaultWorkflowSchema()
	user := &models.User{}
	user.Roles = []models.UserRole{{Role: models.Role{Key: models.RoleStaff}}}
	if !UserHasCapability(schema, user, models.CapCreate) {
		t.Fatal("staff should create")
	}
	if UserHasCapability(schema, user, models.CapDisburse) {
		t.Fatal("staff should not disburse by default")
	}
	acc := &models.User{}
	acc.Roles = []models.UserRole{{Role: models.Role{Key: models.RoleAccountant}}}
	if !UserHasCapability(schema, acc, models.CapFund) {
		t.Fatal("accountant should fund by default")
	}
	admin := &models.User{}
	admin.Roles = []models.UserRole{{Role: models.Role{Key: models.RoleAdmin}}}
	if UserHasCapability(schema, admin, models.CapFund) {
		t.Fatal("admin should not fund by default")
	}
	caps := schema.Capabilities[models.RoleAdmin]
	caps.Fund = true
	schema.Capabilities[models.RoleAdmin] = caps
	if !UserHasCapability(schema, admin, models.CapFund) {
		t.Fatal("admin should fund when enabled")
	}
}

func TestNormalizeMigratesCapFundOnce(t *testing.T) {
	schema := models.WorkflowSchema{
		Version: 1,
		Capabilities: map[models.RoleKey]models.RoleCapabilities{
			models.RoleStaff:      {Create: true},
			models.RoleAccountant: {AttachDocs: true},
			models.RoleAdmin:      {Create: true},
			models.RoleCashier:    {Disburse: true},
			models.RoleSuperAdmin: {Create: true},
		},
		ValidationChain: []models.RoleKey{models.RoleAccountant, models.RoleAdmin},
	}
	schema = normalizeSchema(schema)
	if schema.Version != 2 {
		t.Fatalf("expected version 2, got %d", schema.Version)
	}
	if !schema.Capabilities[models.RoleAccountant].Fund {
		t.Fatal("accountant fund should be migrated on")
	}
	// Intentional disable of all fund must stick after v2.
	acc := schema.Capabilities[models.RoleAccountant]
	acc.Fund = false
	schema.Capabilities[models.RoleAccountant] = acc
	schema = normalizeSchema(schema)
	if schema.Capabilities[models.RoleAccountant].Fund {
		t.Fatal("fund must not be re-enabled after v2")
	}
}

func TestUserCanValidateStep(t *testing.T) {
	admin := &models.User{}
	admin.Roles = []models.UserRole{{Role: models.Role{Key: models.RoleAdmin}}}
	if !UserCanValidateStep(admin, models.ValAdmin) {
		t.Fatal("admin should validate admin step")
	}
	if UserCanValidateStep(admin, models.ValSuperAdmin) {
		t.Fatal("admin should not validate super_admin step")
	}
}
