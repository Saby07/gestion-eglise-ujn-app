package models

type AuditLog struct {
	BaseModel
	UserID   uint   `gorm:"not null;index"`
	User     User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Action   string `gorm:"type:varchar(64);not null;index"`
	Entity   string `gorm:"type:varchar(64);not null;index"`
	EntityID uint   `gorm:"not null;index"`
	Summary  string `gorm:"type:text"`
	IP       string `gorm:"type:varchar(45)"`
}

func (AuditLog) TableName() string { return "audit_logs" }
