package models

type Supplier struct {
	BaseModel
	Name      string `gorm:"type:varchar(191);not null;index"`
	Phone     string `gorm:"type:varchar(64)"`
	Address   string `gorm:"type:varchar(255)"`
	Email     string `gorm:"type:varchar(255)"`
	IsActive  bool   `gorm:"not null;default:true;index"`
}

func (Supplier) TableName() string { return "suppliers" }
