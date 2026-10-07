package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"eglise_ujn/internal/models"

	"gorm.io/gorm"
)

const SettingWorkflowSchema = "workflow_schema"

type WorkflowService struct {
	db       *gorm.DB
	settings *SettingsService
}

func NewWorkflowService(db *gorm.DB, settings *SettingsService) *WorkflowService {
	return &WorkflowService{db: db, settings: settings}
}

func (s *WorkflowService) Load(ctx context.Context) (models.WorkflowSchema, error) {
	raw, err := s.settings.Get(ctx, SettingWorkflowSchema)
	if err != nil {
		return models.WorkflowSchema{}, err
	}
	if strings.TrimSpace(raw) == "" {
		def := models.DefaultWorkflowSchema()
		_ = s.Save(ctx, def)
		return def, nil
	}
	var schema models.WorkflowSchema
	if err := json.Unmarshal([]byte(raw), &schema); err != nil {
		return models.DefaultWorkflowSchema(), nil
	}
	schema = normalizeSchema(schema)
	return schema, nil
}

func (s *WorkflowService) Save(ctx context.Context, schema models.WorkflowSchema) error {
	schema = normalizeSchema(schema)
	if err := validateSchema(schema); err != nil {
		return err
	}
	b, err := json.Marshal(schema)
	if err != nil {
		return err
	}
	return s.settings.Set(ctx, SettingWorkflowSchema, string(b))
}

func (s *WorkflowService) EnsureDefault(ctx context.Context) error {
	raw, err := s.settings.Get(ctx, SettingWorkflowSchema)
	if err != nil {
		return err
	}
	if strings.TrimSpace(raw) != "" {
		return nil
	}
	return s.Save(ctx, models.DefaultWorkflowSchema())
}

// EffectiveChain returns intermediate validators + super_admin (always last).
func EffectiveChain(schema models.WorkflowSchema) []models.ValidationStepKey {
	out := make([]models.ValidationStepKey, 0, len(schema.ValidationChain)+1)
	seen := map[models.RoleKey]bool{}
	for _, r := range schema.ValidationChain {
		if r == models.RoleSuperAdmin || seen[r] || !isIntermediateRole(r) {
			continue
		}
		seen[r] = true
		out = append(out, models.ValidationStepKey(r))
	}
	out = append(out, models.ValSuperAdmin)
	return out
}

func (s *WorkflowService) EffectiveChain(ctx context.Context) ([]models.ValidationStepKey, error) {
	schema, err := s.Load(ctx)
	if err != nil {
		return nil, err
	}
	return EffectiveChain(schema), nil
}

func (s *WorkflowService) Can(ctx context.Context, user *models.User, cap models.WorkflowCapability) (bool, error) {
	if user == nil {
		return false, nil
	}
	schema, err := s.Load(ctx)
	if err != nil {
		return false, err
	}
	return UserHasCapability(schema, user, cap), nil
}

func UserHasCapability(schema models.WorkflowSchema, user *models.User, cap models.WorkflowCapability) bool {
	if user == nil {
		return false
	}
	for _, role := range models.AllRoles {
		if !user.HasRole(role) {
			continue
		}
		caps, ok := schema.Capabilities[role]
		if ok && caps.Has(cap) {
			return true
		}
	}
	return false
}

func UserCanValidateStep(user *models.User, step models.ValidationStepKey) bool {
	if user == nil || step == "" {
		return false
	}
	role := models.RoleKey(step)
	if !role.Valid() {
		return false
	}
	return user.HasRole(role)
}

func RolesWithCapability(schema models.WorkflowSchema, cap models.WorkflowCapability) []models.RoleKey {
	out := make([]models.RoleKey, 0)
	for _, role := range models.AllRoles {
		if schema.Capabilities[role].Has(cap) {
			out = append(out, role)
		}
	}
	return out
}

func normalizeSchema(schema models.WorkflowSchema) models.WorkflowSchema {
	if schema.Version == 0 {
		schema.Version = 1 // pre-CapFund schemas; bumped to 2 after fund migration below
	}
	if schema.Capabilities == nil {
		schema.Capabilities = map[models.RoleKey]models.RoleCapabilities{}
	}
	def := models.DefaultWorkflowSchema()
	for _, role := range models.AllRoles {
		if _, ok := schema.Capabilities[role]; !ok {
			schema.Capabilities[role] = def.Capabilities[role]
		}
	}
	filtered := make([]models.RoleKey, 0, len(schema.ValidationChain))
	seen := map[models.RoleKey]bool{}
	for _, r := range schema.ValidationChain {
		if r == models.RoleSuperAdmin || !isIntermediateRole(r) || seen[r] {
			continue
		}
		seen[r] = true
		filtered = append(filtered, r)
	}
	schema.ValidationChain = filtered
	// v2: CapFund — migrate older schemas that never had a fund flag.
	if schema.Version < 2 {
		hasFund := false
		for _, caps := range schema.Capabilities {
			if caps.Fund {
				hasFund = true
				break
			}
		}
		if !hasFund {
			acc := schema.Capabilities[models.RoleAccountant]
			acc.Fund = def.Capabilities[models.RoleAccountant].Fund
			schema.Capabilities[models.RoleAccountant] = acc
		}
		schema.Version = 2
	}
	return schema
}

func validateSchema(schema models.WorkflowSchema) error {
	hasCreate := false
	hasDisburse := false
	for _, role := range models.AllRoles {
		caps := schema.Capabilities[role]
		if caps.Create {
			hasCreate = true
		}
		if caps.Disburse {
			hasDisburse = true
		}
	}
	if !hasCreate {
		return errors.New("au moins un rôle doit pouvoir créer une réquisition")
	}
	if !hasDisburse {
		return errors.New("au moins un rôle doit pouvoir décaisser")
	}
	for _, r := range schema.ValidationChain {
		if r == models.RoleSuperAdmin {
			return errors.New("super_admin ne peut pas figurer dans la chaîne intermédiaire")
		}
		if !isIntermediateRole(r) {
			return fmt.Errorf("rôle invalide dans la chaîne: %s", r)
		}
	}
	return nil
}

func isIntermediateRole(r models.RoleKey) bool {
	for _, ir := range models.IntermediateRoles {
		if ir == r {
			return true
		}
	}
	return false
}

// ChainPreviewLabels returns human-readable labels for UI preview.
func ChainPreviewLabels(schema models.WorkflowSchema) []string {
	chain := EffectiveChain(schema)
	labels := make([]string, 0, len(chain)+1)
	for _, step := range chain {
		labels = append(labels, models.RoleKey(step).Label())
	}
	labels = append(labels, "Décaissement")
	return labels
}
