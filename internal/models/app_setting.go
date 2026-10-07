package models

type AppSetting struct {
	BaseModel
	// Key is quoted by GORM; avoid raw SQL "key = ?" (reserved in MySQL).
	Key   string `gorm:"column:key;type:varchar(64);uniqueIndex;not null"`
	Value string `gorm:"column:value;type:text"`
}

func (AppSetting) TableName() string { return "app_settings" }
