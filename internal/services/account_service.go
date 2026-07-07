package services

import (
	"context"
	"errors"
	"fmt"

	"eglise_ujn/internal/models"

	"gorm.io/gorm"
)

type AccountService struct {
	db *gorm.DB
}

func NewAccountService(db *gorm.DB) *AccountService {
	return &AccountService{db: db}
}

func (s *AccountService) List(ctx context.Context) ([]models.ChurchAccount, error) {
	var accounts []models.ChurchAccount
	err := s.db.WithContext(ctx).Order("created_at desc").Find(&accounts).Error
	return accounts, err
}

func (s *AccountService) GetByID(ctx context.Context, id uint) (*models.ChurchAccount, error) {
	var acc models.ChurchAccount
	if err := s.db.WithContext(ctx).First(&acc, id).Error; err != nil {
		return nil, err
	}
	return &acc, nil
}

func (s *AccountService) Create(ctx context.Context, name, description, currency, by string) (*models.ChurchAccount, error) {
	if currency == "" {
		currency = "USD"
	}
	acc := models.ChurchAccount{
		Name:        name,
		Description: description,
		Currency:    currency,
		Balance:     0,
		IsActive:    true,
	}
	acc.CreatedBy = by
	return &acc, s.db.WithContext(ctx).Create(&acc).Error
}

func (s *AccountService) Fund(ctx context.Context, accountID uint, amount float64, label string, userID uint, by string) error {
	if amount <= 0 {
		return errors.New("montant invalide")
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var acc models.ChurchAccount
		if err := tx.First(&acc, accountID).Error; err != nil {
			return err
		}
		before := acc.Balance
		acc.Balance += amount
		if err := tx.Save(&acc).Error; err != nil {
			return err
		}
		tr := models.AccountTransaction{
			AccountID:     acc.ID,
			Type:          models.TxFunding,
			Amount:        amount,
			BalanceBefore: before,
			BalanceAfter:  acc.Balance,
			Label:         label,
			UserID:        &userID,
		}
		tr.CreatedBy = by
		return tx.Create(&tr).Error
	})
}

func (s *AccountService) History(ctx context.Context, accountID uint) ([]models.AccountTransaction, error) {
	var txs []models.AccountTransaction
	err := s.db.WithContext(ctx).
		Preload("User").
		Where("account_id = ?", accountID).
		Order("created_at desc").
		Find(&txs).Error
	return txs, err
}

func (s *AccountService) ProjectedBalance(ctx context.Context, accountID uint) (float64, float64, error) {
	acc, err := s.GetByID(ctx, accountID)
	if err != nil {
		return 0, 0, err
	}
	var pending float64
	// Réquisitions ouvertes (validation + décaissement) non encore soldées.
	// account_id est NULL tant que le décaissement n'a pas choisi le compte.
	err = s.db.WithContext(ctx).Model(&models.Requisition{}).
		Where("status = ? AND (account_id = ? OR account_id IS NULL)", models.ReqStatusOpen, accountID).
		Select("COALESCE(SUM(total_amount),0)").Scan(&pending).Error
	if err != nil {
		return 0, 0, err
	}
	return acc.Balance, acc.Balance - pending, nil
}

func (s *AccountService) DeductForDisbursement(ctx context.Context, accountID uint, amount float64, reqID uint, userID uint, by string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.DeductForDisbursementWithTx(tx, accountID, amount, reqID, userID, by)
	})
}

func (s *AccountService) DeductForDisbursementWithTx(tx *gorm.DB, accountID uint, amount float64, reqID uint, userID uint, by string) error {
	var acc models.ChurchAccount
	if err := tx.First(&acc, accountID).Error; err != nil {
		return err
	}
	if acc.Balance < amount {
		return fmt.Errorf("solde insuffisant (disponible: %.2f)", acc.Balance)
	}
	before := acc.Balance
	acc.Balance -= amount
	if err := tx.Save(&acc).Error; err != nil {
		return err
	}
	tr := models.AccountTransaction{
		AccountID:     acc.ID,
		Type:          models.TxDisbursement,
		Amount:        amount,
		BalanceBefore: before,
		BalanceAfter:  acc.Balance,
		Label:         fmt.Sprintf("Décaissement réquisition #%d", reqID),
		RequisitionID: &reqID,
		UserID:        &userID,
	}
	tr.CreatedBy = by
	return tx.Create(&tr).Error
}
