package handlers

import (
	"bytes"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"

	"eglise_ujn/internal/middlewares"
	"eglise_ujn/internal/services"

	"github.com/gin-gonic/gin"
)

type ExportHandler struct {
	export *services.ExportService
	report *services.ReportService
	reqSvc *services.RequisitionService
}

func NewExportHandler(export *services.ExportService, report *services.ReportService, reqSvc *services.RequisitionService) *ExportHandler {
	return &ExportHandler{export: export, report: report, reqSvc: reqSvc}
}

func (h *ExportHandler) ReportsExport(c *gin.Context) {
	period := c.DefaultQuery("period", "daily")
	format := strings.ToLower(c.DefaultQuery("format", "csv"))
	from, to, label := periodRange(period)

	switch format {
	case "html":
		rows, total, err := h.report.Disbursements(c.Request.Context(), from, to)
		if err != nil {
			c.String(http.StatusInternalServerError, "Erreur export")
			return
		}
		body := renderDisbursementsHTML(label, rows, total)
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=rapport_%s.html", period))
		c.Data(http.StatusOK, "text/html; charset=utf-8", body)
	default:
		data, err := h.export.ExportDisbursementsCSV(c.Request.Context(), from, to)
		if err != nil {
			c.String(http.StatusInternalServerError, "Erreur export")
			return
		}
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=rapport_%s.csv", period))
		c.Data(http.StatusOK, "text/csv; charset=utf-8", data)
	}
}

func (h *ExportHandler) RequisitionExport(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	user := middlewares.CurrentUser(c)
	req, err := h.reqSvc.GetByID(c.Request.Context(), uint(id))
	if err != nil || !h.reqSvc.CanView(user, req) {
		c.Redirect(http.StatusFound, "/requisitions")
		return
	}
	format := strings.ToLower(c.DefaultQuery("format", "html"))
	if format != "html" {
		c.String(http.StatusBadRequest, "Format non supporté")
		return
	}
	data, err := h.export.ExportRequisitionHTML(c.Request.Context(), uint(id))
	if err != nil {
		c.String(http.StatusInternalServerError, "Erreur export")
		return
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=requisition_%d.html", id))
	c.Data(http.StatusOK, "text/html; charset=utf-8", data)
}

func renderDisbursementsHTML(label string, rows []services.DisbursementReportRow, total float64) []byte {
	var buf bytes.Buffer
	buf.WriteString("<!DOCTYPE html><html><head><meta charset=\"utf-8\"><title>Rapport ")
	buf.WriteString(html.EscapeString(label))
	buf.WriteString("</title><style>body{font-family:sans-serif;margin:2rem}table{border-collapse:collapse;width:100%}th,td{border:1px solid #ccc;padding:8px}th{background:#f5f5f5}</style></head><body>")
	buf.WriteString("<h1>Rapport de décaissements — ")
	buf.WriteString(html.EscapeString(label))
	buf.WriteString("</h1>")
	buf.WriteString(fmt.Sprintf("<p>Total : <strong>%.2f USD</strong></p>", total))
	buf.WriteString("<table><tr><th>ID</th><th>Réquisition</th><th>Montant</th><th>Mode</th><th>Date</th><th>Par</th></tr>")
	for _, r := range rows {
		buf.WriteString(fmt.Sprintf("<tr><td>%d</td><td>%s</td><td>%.2f</td><td>%s</td><td>%s</td><td>%s</td></tr>",
			r.ID, html.EscapeString(r.Title), r.Amount, html.EscapeString(string(r.PaymentMode)),
			r.DisbursedAt.Format("02/01/2006 15:04"), html.EscapeString(r.DisbursedBy)))
	}
	buf.WriteString("</table></body></html>")
	return buf.Bytes()
}
