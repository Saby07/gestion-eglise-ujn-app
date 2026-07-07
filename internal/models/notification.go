package models

import "time"

type Notification struct {
	BaseModel
	UserID        uint        `gorm:"not null;index:idx_notif_user_read"`
	User          User        `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	RequisitionID uint        `gorm:"not null;index"`
	Requisition   Requisition `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Step          string      `gorm:"type:varchar(32);not null;index"`
	Message       string      `gorm:"type:varchar(500);not null"`
	ReadAt        *time.Time  `gorm:"index:idx_notif_user_read"`
}

func (Notification) TableName() string { return "notifications" }

func (n Notification) IsUnread() bool { return n.ReadAt == nil }
