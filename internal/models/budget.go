package models

type Budget struct {
	BaseModel
	CategoryID uint            `gorm:"not null;uniqueIndex:idx_budget_category_year"`
	Category   ExpenseCategory `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Year       int             `gorm:"not null;uniqueIndex:idx_budget_category_year;index"`
	Amount     float64         `gorm:"type:decimal(15,2);not null;default:0"`
}

func (Budget) TableName() string { return "budgets" }
