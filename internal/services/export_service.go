package services

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"html"
	"time"

	"eglise_ujn/internal/models"

	"gorm.io/gorm"
)

type ExportService struct {
	db *gorm.DB
}

func NewExportService(db *gorm.DB) *ExportService {
	return &ExportService{db: db}
}

func (s *ExportService) ExportDisbursementsCSV(ctx context.Context, from, to time.Time) ([]byte, error) {
	var list []models.Disbursement
	err := s.db.WithContext(ctx).
		Preload("Requisition").
		Preload("DisbursedBy").
		Preload("Account").
		Where("disbursed_at >= ? AND disbursed_at < ?", from, to).
		Order("disbursed_at desc").
		Find(&list).Error
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"ID", "Réquisition", "Montant", "Mode", "Compte", "Décaissé par", "Date"})
	for _, d := range list {
		_ = w.Write([]string{
			fmt.Sprintf("%d", d.ID),
			d.Requisition.Title,
			fmt.Sprintf("%.2f", d.Amount),
			string(d.PaymentMode),
			d.Account.Name,
			d.DisbursedBy.FullName(),
			d.DisbursedAt.Format("02/01/2006 15:04"),
		})
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}

func (s *ExportService) ExportRequisitionHTML(ctx context.Context, reqID uint) ([]byte, error) {
	var req models.Requisition
	err := s.db.WithContext(ctx).
		Preload("User").
		Preload("Items").
		Preload("Validations.ValidatedBy").
		Preload("Category").
		Preload("Supplier").
		Preload("Account").
		First(&req, reqID).Error
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	buf.WriteString("<!DOCTYPE html><html><head><meta charset=\"utf-8\"><title>Réquisition ")
	buf.WriteString(html.EscapeString(req.Title))
	buf.WriteString("</title><style>body{font-family:sans-serif;margin:2rem}table{border-collapse:collapse;width:100%}th,td{border:1px solid #ccc;padding:8px;text-align:left}th{background:#f5f5f5}</style></head><body>")
	buf.WriteString("<h1>Réquisition — ")
	buf.WriteString(html.EscapeString(req.Title))
	buf.WriteString("</h1>")
	buf.WriteString(fmt.Sprintf("<p><strong>Date:</strong> %s</p>", req.RequisitionDate.Format("02/01/2006")))
	buf.WriteString(fmt.Sprintf("<p><strong>Auteur:</strong> %s</p>", html.EscapeString(req.User.FullName())))
	buf.WriteString(fmt.Sprintf("<p><strong>Montant total:</strong> %.2f</p>", req.TotalAmount))
	if req.Category != nil {
		buf.WriteString(fmt.Sprintf("<p><strong>Catégorie:</strong> %s</p>", html.EscapeString(req.Category.Name)))
	}

	buf.WriteString("<h2>Détail</h2><table><tr><th>Désignation</th><th>Qté</th><th>P.U.</th><th>Total</th></tr>")
	for _, it := range req.Items {
		buf.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%.2f</td><td>%.2f</td><td>%.2f</td></tr>",
			html.EscapeString(it.Designation), it.Quantity, it.UnitPrice, it.TotalPrice))
	}
	buf.WriteString("</table>")

	if len(req.Validations) > 0 {
		buf.WriteString("<h2>Validations</h2><table><tr><th>Étape</th><th>Par</th><th>Date</th><th>Commentaire</th></tr>")
		for _, v := range req.Validations {
			buf.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>",
				html.EscapeString(string(v.Step)),
				html.EscapeString(v.ValidatedBy.FullName()),
				v.ValidatedAt.Format("02/01/2006 15:04"),
				html.EscapeString(v.Comment)))
		}
		buf.WriteString("</table>")
	}

	buf.WriteString("</body></html>")
	return buf.Bytes(), nil
}
