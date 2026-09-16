package models

type ExpenseCategory struct {
	BaseModel
	Name        string `gorm:"type:varchar(128);not null;uniqueIndex"`
	Description string `gorm:"type:varchar(255)"`
	IsActive    bool   `gorm:"not null;default:true;index"`
}

func (ExpenseCategory) TableName() string { return "expense_categories" }
