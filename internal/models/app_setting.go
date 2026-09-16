package models

type AppSetting struct {
	BaseModel
	Key   string `gorm:"type:varchar(64);uniqueIndex;not null"`
	Value string `gorm:"type:text"`
}

func (AppSetting) TableName() string { return "app_settings" }
