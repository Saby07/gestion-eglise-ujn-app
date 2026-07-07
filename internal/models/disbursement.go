package models

import "time"

type PaymentMode string

const (
	PaymentCash         PaymentMode = "cash"
	PaymentBankTransfer PaymentMode = "bank_transfer"
	PaymentCheque       PaymentMode = "cheque"
	PaymentMomo         PaymentMode = "momo"
)

func AllPaymentModes() []PaymentMode {
	return []PaymentMode{PaymentCash, PaymentBankTransfer, PaymentCheque, PaymentMomo}
}

func (p PaymentMode) Label() string {
	switch p {
	case PaymentCash:
		return "Espèces"
	case PaymentBankTransfer:
		return "Virement bancaire"
	case PaymentCheque:
		return "Chèque"
	case PaymentMomo:
		return "Momo"
	default:
		return string(p)
	}
}

type Disbursement struct {
	BaseModel
	RequisitionID uint          `gorm:"not null;uniqueIndex"`
	Requisition   Requisition   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	AccountID     uint          `gorm:"not null;index"`
	Account       ChurchAccount `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Amount        float64       `gorm:"type:decimal(15,2);not null"`
	PaymentMode   PaymentMode   `gorm:"type:varchar(32);not null"`
	ReceiptPath   string        `gorm:"type:varchar(500)"`
	DisbursedByID uint          `gorm:"not null;index"`
	DisbursedBy   User          `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	DisbursedAt   time.Time     `gorm:"not null;index"`
	Note          string        `gorm:"type:text"`
}

func (Disbursement) TableName() string { return "disbursements" }
