package services

import (
	"context"
	"strconv"
	"time"

	"eglise_ujn/internal/models"

	"gorm.io/gorm"
)

type ReminderService struct {
	db       *gorm.DB
	settings *SettingsService
	notifSvc *NotificationService
	emailSvc *EmailService
}

func NewReminderService(db *gorm.DB, settings *SettingsService, notifSvc *NotificationService, emailSvc *EmailService) *ReminderService {
	return &ReminderService{db: db, settings: settings, notifSvc: notifSvc, emailSvc: emailSvc}
}

func (s *ReminderService) ProcessReminders(ctx context.Context) (int, error) {
	daysStr, _ := s.settings.Get(ctx, SettingReminderDays)
	days, err := strconv.Atoi(daysStr)
	if err != nil || days <= 0 {
		days = 3
	}
	cutoff := time.Now().AddDate(0, 0, -days)

	var reqs []models.Requisition
	err = s.db.WithContext(ctx).
		Preload("User").
		Where("status = ? AND current_step IN ?", models.ReqStatusOpen, []models.RequisitionStep{
			models.StepPendingAccountant,
			models.StepPendingAdmin,
			models.StepPendingSuperAdmin,
			models.StepPendingDisbursement,
		}).
		Where("updated_at <= ?", cutoff).
		Find(&reqs).Error
	if err != nil {
		return 0, err
	}

	sent := 0
	for _, req := range reqs {
		if s.notifSvc != nil {
			if err := s.notifSvc.NotifyStep(ctx, req.ID, req.CurrentStep, "Système", req.Title); err == nil {
				sent++
			}
		}
	}
	return sent, nil
}
