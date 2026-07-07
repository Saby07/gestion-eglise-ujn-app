package models

import "time"

type BaseModel struct {
	ID        uint       `gorm:"primaryKey"`
	CreatedAt time.Time  `gorm:"not null"`
	UpdatedAt time.Time  `gorm:"not null"`
	DeletedAt *time.Time `gorm:"index"`
	CreatedBy string     `gorm:"type:varchar(100)"`
	UpdatedBy string     `gorm:"type:varchar(100)"`
	DeletedBy string     `gorm:"type:varchar(100)"`
}
