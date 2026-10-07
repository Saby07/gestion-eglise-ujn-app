package services

import (
	"context"
	"fmt"
	"time"

	"eglise_ujn/internal/models"

	"gorm.io/gorm"
)

type NotificationService struct {
	db          *gorm.DB
	emailSvc    *EmailService
	workflowSvc *WorkflowService
}

func NewNotificationService(db *gorm.DB, emailSvc *EmailService, workflowSvc *WorkflowService) *NotificationService {
	return &NotificationService{db: db, emailSvc: emailSvc, workflowSvc: workflowSvc}
}

func (s *NotificationService) ListForUser(ctx context.Context, userID uint, limit int) ([]models.Notification, error) {
	if limit <= 0 {
		limit = 20
	}
	var list []models.Notification
	err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at desc").
		Limit(limit).
		Find(&list).Error
	return list, err
}

func (s *NotificationService) CountUnread(ctx context.Context, userID uint) (int, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&models.Notification{}).
		Where("user_id = ? AND read_at IS NULL", userID).
		Count(&count).Error
	return int(count), err
}

func (s *NotificationService) MarkRead(ctx context.Context, notifID, userID uint) (*models.Notification, error) {
	var n models.Notification
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", notifID, userID).First(&n).Error; err != nil {
		return nil, err
	}
	if n.ReadAt == nil {
		now := time.Now()
		if err := s.db.WithContext(ctx).Model(&n).Update("read_at", &now).Error; err != nil {
			return nil, err
		}
		n.ReadAt = &now
	}
	return &n, nil
}

func (s *NotificationService) MarkStepRead(ctx context.Context, requisitionID uint, step models.RequisitionStep) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&models.Notification{}).
		Where("requisition_id = ? AND step = ? AND read_at IS NULL", requisitionID, string(step)).
		Update("read_at", &now).Error
}

func (s *NotificationService) MarkAllRead(ctx context.Context, userID uint) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&models.Notification{}).
		Where("user_id = ? AND read_at IS NULL", userID).
		Update("read_at", &now).Error
}

func (s *NotificationService) MarkAllReadForRequisition(ctx context.Context, requisitionID uint) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&models.Notification{}).
		Where("requisition_id = ? AND read_at IS NULL", requisitionID).
		Update("read_at", &now).Error
}

func (s *NotificationService) NotifyStep(ctx context.Context, requisitionID uint, step models.RequisitionStep, actorName, title string) error {
	users, err := s.recipientsForStep(ctx, step)
	if err != nil {
		return err
	}
	if len(users) == 0 {
		return nil
	}
	msg := stepNotificationMessage(step, title, actorName)
	if err := s.createForUsers(ctx, requisitionID, step, msg, users); err != nil {
		return err
	}
	if s.emailSvc != nil && s.emailSvc.IsEnabled(ctx) {
		subject := fmt.Sprintf("Réquisition — %s", title)
		for _, u := range users {
			if u.Email != "" {
				_ = s.emailSvc.Send(ctx, u.Email, subject, msg)
			}
		}
	}
	return nil
}

func (s *NotificationService) createForUsers(ctx context.Context, requisitionID uint, step models.RequisitionStep, message string, users []models.User) error {
	stepKey := string(step)
	for _, u := range users {
		var existing int64
		if err := s.db.WithContext(ctx).Model(&models.Notification{}).
			Where("user_id = ? AND requisition_id = ? AND step = ? AND read_at IS NULL", u.ID, requisitionID, stepKey).
			Count(&existing).Error; err != nil {
			return err
		}
		if existing > 0 {
			continue
		}
		n := models.Notification{
			UserID:        u.ID,
			RequisitionID: requisitionID,
			Step:          stepKey,
			Message:       message,
		}
		if err := s.db.WithContext(ctx).Create(&n).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *NotificationService) recipientsForStep(ctx context.Context, step models.RequisitionStep) ([]models.User, error) {
	if step == models.StepPendingDisbursement {
		return s.findDisbursers(ctx)
	}
	if role, ok := models.RoleFromPendingStep(step); ok {
		return s.findUsersByRoles(ctx, role)
	}
	return nil, nil
}

func (s *NotificationService) findDisbursers(ctx context.Context) ([]models.User, error) {
	roles := []models.RoleKey{models.RoleCashier}
	if s.workflowSvc != nil {
		schema, err := s.workflowSvc.Load(ctx)
		if err == nil {
			roles = RolesWithCapability(schema, models.CapDisburse)
		}
	}
	if len(roles) == 0 {
		roles = []models.RoleKey{models.RoleCashier}
	}
	return s.findUsersByRoles(ctx, roles...)
}

func (s *NotificationService) findUsersByRoles(ctx context.Context, roles ...models.RoleKey) ([]models.User, error) {
	if len(roles) == 0 {
		return nil, nil
	}
	keys := make([]string, len(roles))
	for i, r := range roles {
		keys[i] = string(r)
	}
	var users []models.User
	err := s.db.WithContext(ctx).
		Preload("Roles.Role").
		Joins("JOIN user_roles ON user_roles.user_id = users.id AND user_roles.deleted_at IS NULL").
		Joins("JOIN roles ON roles.id = user_roles.role_id AND roles.deleted_at IS NULL").
		Where("users.is_active = ? AND users.deleted_at IS NULL", true).
		Where("roles.key IN ?", keys).
		Distinct().
		Find(&users).Error
	return users, err
}

func stepNotificationMessage(step models.RequisitionStep, title, actor string) string {
	if step == models.StepPendingDisbursement {
		return fmt.Sprintf("Réquisition « %s » validée par %s — prête au décaissement", title, actor)
	}
	if role, ok := models.RoleFromPendingStep(step); ok {
		return fmt.Sprintf("Réquisition « %s » — validation %s requise (par %s)", title, role.Label(), actor)
	}
	return fmt.Sprintf("Réquisition « %s » — action requise", title)
}

func FormatTimeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "À l'instant"
	case d < time.Hour:
		return fmt.Sprintf("Il y a %d min", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("Il y a %dh", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("Il y a %d j", int(d.Hours()/24))
	default:
		return t.Format("02/01/2006")
	}
}
