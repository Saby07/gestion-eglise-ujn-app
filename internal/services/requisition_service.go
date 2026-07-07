package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"eglise_ujn/internal/models"

	"gorm.io/gorm"
)

type RequisitionService struct {
	db       *gorm.DB
	notifSvc *NotificationService
}

func NewRequisitionService(db *gorm.DB, notifSvc *NotificationService) *RequisitionService {
	return &RequisitionService{db: db, notifSvc: notifSvc}
}

type RequisitionItemInput struct {
	Designation string
	Quantity    float64
	UnitPrice   float64
}

func (s *RequisitionService) List(ctx context.Context, filter string, user *models.User) ([]models.Requisition, error) {
	q := s.db.WithContext(ctx).
		Preload("User").
		Preload("Items").
		Preload("Validations.ValidatedBy").
		Preload("Disbursement").
		Order("created_at desc")

	if user != nil && s.isStaffOnly(user) {
		q = q.Where("user_id = ?", user.ID)
	}

	switch filter {
	case "pending_validation":
		step := s.pendingStepForUser(user)
		if step != "" {
			q = q.Where("status = ? AND current_step = ?", models.ReqStatusOpen, step)
		}
	case "pending_disbursement":
		q = q.Where("status = ? AND current_step = ?", models.ReqStatusOpen, models.StepPendingDisbursement)
	case "mine":
		q = q.Where("user_id = ?", user.ID)
	case "cancelled":
		q = q.Where("status = ?", models.ReqStatusCancelled)
	case "completed":
		q = q.Where("status = ?", models.ReqStatusCompleted)
	}

	var list []models.Requisition
	return list, q.Find(&list).Error
}

func (s *RequisitionService) isStaffOnly(user *models.User) bool {
	return user.HasRole(models.RoleStaff) &&
		!user.HasRole(models.RoleAdmin) &&
		!user.HasRole(models.RoleSuperAdmin) &&
		!user.HasRole(models.RoleAccountant) &&
		!user.HasRole(models.RoleCashier)
}

func (s *RequisitionService) CanView(user *models.User, req *models.Requisition) bool {
	if user == nil || req == nil {
		return false
	}
	if s.isStaffOnly(user) {
		return req.UserID == user.ID
	}
	return true
}

func (s *RequisitionService) pendingStepForUser(user *models.User) models.RequisitionStep {
	if user.HasRole(models.RoleAccountant) {
		return models.StepPendingAccountant
	}
	if user.HasRole(models.RoleAdmin) {
		return models.StepPendingAdmin
	}
	if user.HasRole(models.RoleSuperAdmin) {
		return models.StepPendingSuperAdmin
	}
	return ""
}

func (s *RequisitionService) GetByID(ctx context.Context, id uint) (*models.Requisition, error) {
	var req models.Requisition
	err := s.db.WithContext(ctx).
		Preload("User").
		Preload("Items").
		Preload("Validations.ValidatedBy").
		Preload("Disbursement.DisbursedBy").
		Preload("Account").
		First(&req, id).Error
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func (s *RequisitionService) Create(ctx context.Context, user *models.User, title string, date time.Time, items []RequisitionItemInput, accountID *uint) (*models.Requisition, error) {
	if title == "" || len(items) == 0 {
		return nil, errors.New("titre et items requis")
	}
	creatorRole := models.CreatorStaff
	if user.HasRole(models.RoleAdmin) || user.HasRole(models.RoleSuperAdmin) {
		creatorRole = models.CreatorAdmin
	}
	var total float64
	reqItems := make([]models.RequisitionItem, 0, len(items))
	for _, it := range items {
		if it.Designation == "" || it.Quantity <= 0 || it.UnitPrice < 0 {
			return nil, errors.New("item invalide")
		}
		line := it.Quantity * it.UnitPrice
		total += line
		reqItems = append(reqItems, models.RequisitionItem{
			Designation: it.Designation,
			Quantity:    it.Quantity,
			UnitPrice:   it.UnitPrice,
			TotalPrice:  line,
		})
	}
	req := models.Requisition{
		Title:           title,
		RequisitionDate: date,
		TotalAmount:     total,
		CreatorRole:     creatorRole,
		CurrentStep:     models.StepPendingAccountant,
		Status:          models.ReqStatusOpen,
		UserID:          user.ID,
		AccountID:       accountID,
		Items:           reqItems,
	}
	req.CreatedBy = user.FullName()
	if err := s.db.WithContext(ctx).Create(&req).Error; err != nil {
		return nil, err
	}
	if s.notifSvc != nil {
		_ = s.notifSvc.NotifyStep(ctx, req.ID, models.StepPendingAccountant, user.FullName(), req.Title)
	}
	return &req, nil
}

type AccountantExtras struct {
	InvoicePath, DeliveryNotePath, PurchaseOrderPath, ReceptionNotePath string
	SupplierName, SupplierPhone, SupplierAddress                        string
}

func (s *RequisitionService) Validate(ctx context.Context, reqID uint, validator *models.User, comment string, extras *AccountantExtras) error {
	var validatedStep models.RequisitionStep
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var req models.Requisition
		if err := tx.First(&req, reqID).Error; err != nil {
			return err
		}
		var validations []models.RequisitionValidation
		if err := tx.Where("requisition_id = ?", reqID).Find(&validations).Error; err != nil {
			return err
		}
		req.Validations = validations
		if req.Status != models.ReqStatusOpen {
			return errors.New("réquisition non modifiable")
		}
		step := req.NextValidationStep()
		if step == "" {
			return errors.New("déjà validée")
		}
		if !s.canValidate(validator, step) {
			return errors.New("non autorisé pour cette étape")
		}
		if req.CurrentStep != s.stepToRequisitionStep(step) {
			return errors.New("étape de validation incorrecte")
		}
		if step == models.ValAccountant && extras != nil {
			req.InvoicePath = extras.InvoicePath
			req.DeliveryNotePath = extras.DeliveryNotePath
			req.PurchaseOrderPath = extras.PurchaseOrderPath
			req.ReceptionNotePath = extras.ReceptionNotePath
			req.SupplierName = extras.SupplierName
			req.SupplierPhone = extras.SupplierPhone
			req.SupplierAddress = extras.SupplierAddress
			if err := tx.Save(&req).Error; err != nil {
				return err
			}
		}
		val := models.RequisitionValidation{
			RequisitionID: req.ID,
			Step:          step,
			ValidatedByID: validator.ID,
			ValidatedAt:   time.Now(),
			Comment:       comment,
		}
		val.CreatedBy = validator.FullName()
		if err := tx.Create(&val).Error; err != nil {
			return err
		}
		req.Validations = append(req.Validations, val)
		validatedStep = s.stepToRequisitionStep(step)
		req.CurrentStep = computeStepAfterValidation(&req)
		return tx.Model(&req).Update("current_step", req.CurrentStep).Error
	})
	if err != nil {
		return err
	}
	if s.notifSvc != nil {
		_ = s.notifSvc.MarkStepRead(ctx, reqID, validatedStep)
		updated, loadErr := s.GetByID(ctx, reqID)
		if loadErr == nil && updated.Status == models.ReqStatusOpen {
			_ = s.notifSvc.NotifyStep(ctx, updated.ID, updated.CurrentStep, validator.FullName(), updated.Title)
		}
	}
	return nil
}

// computeStepAfterValidation calcule l'étape suivante en mémoire (évite une
// requête DB pendant une transaction SQLite à connexion unique → deadlock).
func computeStepAfterValidation(req *models.Requisition) models.RequisitionStep {
	next := req.NextValidationStep()
	if next == "" {
		return models.StepPendingDisbursement
	}
	switch next {
	case models.ValAccountant:
		return models.StepPendingAccountant
	case models.ValAdmin:
		return models.StepPendingAdmin
	case models.ValSuperAdmin:
		return models.StepPendingSuperAdmin
	default:
		return models.StepPendingDisbursement
	}
}

func (s *RequisitionService) stepToRequisitionStep(step models.ValidationStepKey) models.RequisitionStep {
	switch step {
	case models.ValAccountant:
		return models.StepPendingAccountant
	case models.ValAdmin:
		return models.StepPendingAdmin
	case models.ValSuperAdmin:
		return models.StepPendingSuperAdmin
	default:
		return ""
	}
}

func (s *RequisitionService) canValidate(user *models.User, step models.ValidationStepKey) bool {
	switch step {
	case models.ValAccountant:
		return user.HasRole(models.RoleAccountant)
	case models.ValAdmin:
		return user.HasRole(models.RoleAdmin) || user.HasRole(models.RoleSuperAdmin)
	case models.ValSuperAdmin:
		return user.HasRole(models.RoleSuperAdmin)
	default:
		return false
	}
}

func (s *RequisitionService) CanCancel(user *models.User, req *models.Requisition) bool {
	if req.Status != models.ReqStatusOpen {
		return false
	}
	if user.HasRole(models.RoleSuperAdmin) || user.HasRole(models.RoleAdmin) {
		return true
	}
	if user.HasRole(models.RoleAccountant) {
		return !req.IsValidated(models.ValAdmin)
	}
	return false
}

func (s *RequisitionService) Cancel(ctx context.Context, reqID uint, user *models.User, reason string) error {
	req, err := s.GetByID(ctx, reqID)
	if err != nil {
		return err
	}
	if !s.CanCancel(user, req) {
		return errors.New("annulation non autorisée")
	}
	now := time.Now()
	uid := user.ID
	err = s.db.WithContext(ctx).Model(req).Updates(map[string]interface{}{
		"status":        models.ReqStatusCancelled,
		"current_step":  models.StepCancelled,
		"cancel_reason": reason,
		"cancelled_at":  &now,
		"cancelled_by":  &uid,
		"updated_by":    user.FullName(),
	}).Error
	if err == nil && s.notifSvc != nil {
		_ = s.notifSvc.MarkAllReadForRequisition(ctx, reqID)
	}
	return err
}

func (s *RequisitionService) Delete(ctx context.Context, reqID uint, user *models.User) error {
	req, err := s.GetByID(ctx, reqID)
	if err != nil {
		return err
	}
	if req.Status == models.ReqStatusCompleted {
		return errors.New("impossible de supprimer une réquisition décaissée")
	}
	if !user.HasRole(models.RoleSuperAdmin) && !user.HasRole(models.RoleAdmin) {
		return errors.New("non autorisé")
	}
	return s.db.WithContext(ctx).Delete(req).Error
}

func (s *RequisitionService) SetAccountantCanDisburse(ctx context.Context, userID uint, allowed bool) error {
	return s.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).
		Update("can_disburse", allowed).Error
}

func (s *RequisitionService) Disburse(ctx context.Context, reqID uint, disburser *models.User, accountID uint, mode models.PaymentMode, receiptPath, note string, accountSvc *AccountService) error {
	if !disburser.HasRole(models.RoleCashier) && !(disburser.HasRole(models.RoleAccountant) && disburser.CanDisburse) {
		return errors.New("seul le caissier (ou comptable autorisé) peut décaisser")
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var req models.Requisition
		if err := tx.Preload("Disbursement").First(&req, reqID).Error; err != nil {
			return err
		}
		if req.CurrentStep != models.StepPendingDisbursement || req.Status != models.ReqStatusOpen {
			return errors.New("réquisition non prête pour décaissement")
		}
		var existing int64
		if err := tx.Model(&models.Disbursement{}).Where("requisition_id = ?", reqID).Count(&existing).Error; err != nil {
			return err
		}
		if existing > 0 {
			return errors.New("déjà décaissée")
		}
		if err := accountSvc.DeductForDisbursementWithTx(tx, accountID, req.TotalAmount, req.ID, disburser.ID, disburser.FullName()); err != nil {
			return err
		}
		d := models.Disbursement{
			RequisitionID: req.ID,
			AccountID:     accountID,
			Amount:        req.TotalAmount,
			PaymentMode:   mode,
			ReceiptPath:   receiptPath,
			DisbursedByID: disburser.ID,
			DisbursedAt:   time.Now(),
			Note:          note,
		}
		d.CreatedBy = disburser.FullName()
		if err := tx.Create(&d).Error; err != nil {
			return err
		}
		return tx.Model(&req).Updates(map[string]interface{}{
			"current_step": models.StepDisbursed,
			"status":       models.ReqStatusCompleted,
			"account_id":   accountID,
			"updated_by":   disburser.FullName(),
		}).Error
	})
	if err != nil {
		return err
	}
	if s.notifSvc != nil {
		_ = s.notifSvc.MarkStepRead(ctx, reqID, models.StepPendingDisbursement)
	}
	return nil
}

func (s *RequisitionService) CountByStep(ctx context.Context) (map[string]int64, error) {
	out := make(map[string]int64)
	type row struct {
		CurrentStep string
		Count       int64
	}
	var rows []row
	err := s.db.WithContext(ctx).Model(&models.Requisition{}).
		Where("status = ?", models.ReqStatusOpen).
		Select("current_step, count(*) as count").
		Group("current_step").Scan(&rows).Error
	for _, r := range rows {
		out[r.CurrentStep] = r.Count
	}
	return out, err
}

func (s *RequisitionService) TotalDisbursed(ctx context.Context, from, to time.Time) (float64, error) {
	var total float64
	err := s.db.WithContext(ctx).Model(&models.Disbursement{}).
		Where("disbursed_at >= ? AND disbursed_at < ?", from, to).
		Select("COALESCE(SUM(amount),0)").Scan(&total).Error
	return total, err
}

func FormatPeriodLabel(from, to time.Time) string {
	return fmt.Sprintf("%s — %s", from.Format("02/01/2006"), to.Add(-time.Nanosecond).Format("02/01/2006"))
}
