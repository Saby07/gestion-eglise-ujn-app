package models

// TransactionType décrit le sens d'une écriture sur le compte.
type TransactionType string

const (
	TxFunding      TransactionType = "funding"
	TxDisbursement TransactionType = "disbursement"
	TxAdjustment   TransactionType = "adjustment"
)

func (t TransactionType) Label() string {
	switch t {
	case TxFunding:
		return "Approvisionnement"
	case TxDisbursement:
		return "Décaissement"
	case TxAdjustment:
		return "Ajustement"
	default:
		return string(t)
	}
}

type ChurchAccount struct {
	BaseModel
	Name         string               `gorm:"type:varchar(191);not null"`
	Description  string               `gorm:"type:text"`
	Balance      float64              `gorm:"type:decimal(15,2);not null;default:0"`
	Currency     string               `gorm:"type:varchar(8);not null;default:'USD'"`
	IsActive     bool                 `gorm:"not null;default:true"`
	Transactions []AccountTransaction `gorm:"foreignKey:AccountID"`
}

func (ChurchAccount) TableName() string { return "church_accounts" }

type AccountTransaction struct {
	BaseModel
	AccountID     uint            `gorm:"not null;index"`
	Account       ChurchAccount   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Type          TransactionType `gorm:"type:varchar(32);not null;index"`
	Amount        float64         `gorm:"type:decimal(15,2);not null"`
	BalanceBefore float64         `gorm:"type:decimal(15,2);not null"`
	BalanceAfter  float64         `gorm:"type:decimal(15,2);not null"`
	Label         string          `gorm:"type:varchar(255)"`
	RequisitionID *uint           `gorm:"index"`
	UserID        *uint           `gorm:"index"`
	User          *User           `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
}

func (AccountTransaction) TableName() string { return "account_transactions" }
