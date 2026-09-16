package services

import (
	"context"
	"errors"

	"eglise_ujn/internal/models"

	"gorm.io/gorm"
)

const (
	SettingChurchName    = "church_name"
	SettingTheme         = "theme"
	SettingReminderDays  = "reminder_days"
	SettingSMTPHost      = "smtp_host"
	SettingSMTPPort      = "smtp_port"
	SettingSMTPUser      = "smtp_user"
	SettingSMTPPassword  = "smtp_password"
	SettingSMTPFrom      = "smtp_from"
	SettingLogoPath      = "logo_path"
)

var knownSettings = []string{
	SettingChurchName,
	SettingTheme,
	SettingReminderDays,
	SettingSMTPHost,
	SettingSMTPPort,
	SettingSMTPUser,
	SettingSMTPPassword,
	SettingSMTPFrom,
	SettingLogoPath,
}

type SettingsService struct {
	db *gorm.DB
}

func NewSettingsService(db *gorm.DB) *SettingsService {
	return &SettingsService{db: db}
}

func (s *SettingsService) Get(ctx context.Context, key string) (string, error) {
	var setting models.AppSetting
	err := s.db.WithContext(ctx).Where("key = ?", key).First(&setting).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return setting.Value, nil
}

func (s *SettingsService) Set(ctx context.Context, key, value string) error {
	if !isKnownSetting(key) {
		return errors.New("clé de paramètre inconnue")
	}
	var setting models.AppSetting
	err := s.db.WithContext(ctx).Where("key = ?", key).First(&setting).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		setting = models.AppSetting{Key: key, Value: value}
		return s.db.WithContext(ctx).Create(&setting).Error
	}
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Model(&setting).Update("value", value).Error
}

func (s *SettingsService) GetAll(ctx context.Context) (map[string]string, error) {
	var settings []models.AppSetting
	if err := s.db.WithContext(ctx).Find(&settings).Error; err != nil {
		return nil, err
	}
	out := make(map[string]string, len(settings))
	for _, st := range settings {
		out[st.Key] = st.Value
	}
	return out, nil
}

func isKnownSetting(key string) bool {
	for _, k := range knownSettings {
		if k == key {
			return true
		}
	}
	return false
}
