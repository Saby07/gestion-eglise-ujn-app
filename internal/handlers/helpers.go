package handlers

import (
	"strconv"
	"strings"
	"time"

	"eglise_ujn/internal/models"
	"eglise_ujn/internal/services"
	"eglise_ujn/templates/viewmodels"

	"github.com/gin-gonic/gin"
)

func mapCategories(list []models.ExpenseCategory) []viewmodels.CategoryVM {
	out := make([]viewmodels.CategoryVM, 0, len(list))
	for _, c := range list {
		out = append(out, viewmodels.CategoryVM{
			ID:          c.ID,
			Name:        c.Name,
			Description: c.Description,
			Active:      c.IsActive,
		})
	}
	return out
}

func mapSuppliers(list []models.Supplier) []viewmodels.SupplierVM {
	out := make([]viewmodels.SupplierVM, 0, len(list))
	for _, s := range list {
		out = append(out, viewmodels.SupplierVM{
			ID:      s.ID,
			Name:    s.Name,
			Phone:   s.Phone,
			Email:   s.Email,
			Address: s.Address,
			Active:  s.IsActive,
		})
	}
	return out
}

func mapAuditLogs(list []models.AuditLog) []viewmodels.AuditLogRow {
	out := make([]viewmodels.AuditLogRow, 0, len(list))
	for _, a := range list {
		userName := "—"
		if a.User.ID > 0 {
			userName = a.User.FullName()
		}
		out = append(out, viewmodels.AuditLogRow{
			ID:        a.ID,
			UserName:  userName,
			Action:    a.Action,
			Entity:    a.Entity,
			Details:   a.Summary,
			IP:        a.IP,
			CreatedAt: a.CreatedAt.Format("02/01/2006 15:04"),
		})
	}
	return out
}

func mapBudgets(list []services.BudgetWithUsage) []viewmodels.BudgetVM {
	out := make([]viewmodels.BudgetVM, 0, len(list))
	for _, b := range list {
		pct := 0.0
		if b.Amount > 0 {
			pct = (b.Spent / b.Amount) * 100
		}
		name := "—"
		if b.Category.ID > 0 {
			name = b.Category.Name
		}
		out = append(out, viewmodels.BudgetVM{
			ID:           b.ID,
			Year:         b.Year,
			CategoryName: name,
			Amount:       b.Amount,
			Spent:        b.Spent,
			UsagePercent: pct,
		})
	}
	return out
}

func mapStatsVM(stats *services.AdvancedStats) viewmodels.StatsVM {
	vm := viewmodels.StatsVM{
		ValidationDelayLabels: []string{"Délai moyen (jours)"},
		ValidationDelayValues: []float64{stats.AvgValidationDays},
	}
	for _, c := range stats.ByCategory {
		vm.CategoryLabels = append(vm.CategoryLabels, c.CategoryName)
		vm.CategoryValues = append(vm.CategoryValues, c.Total)
	}
	for _, m := range stats.MonthlyTrend {
		vm.MonthlyLabels = append(vm.MonthlyLabels, m.Month)
		vm.MonthlyValues = append(vm.MonthlyValues, m.Total)
	}
	return vm
}

func parseOptionalUint(s string) *uint {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil || v == 0 {
		return nil
	}
	u := uint(v)
	return &u
}

func parseSearchFilter(c *gin.Context) services.SearchFilter {
	f := services.SearchFilter{Q: strings.TrimSpace(c.Query("q"))}
	if fromStr := strings.TrimSpace(c.Query("from")); fromStr != "" {
		if t, err := time.Parse("2006-01-02", fromStr); err == nil {
			f.From = &t
		}
	}
	if toStr := strings.TrimSpace(c.Query("to")); toStr != "" {
		if t, err := time.Parse("2006-01-02", toStr); err == nil {
			end := t.AddDate(0, 0, 1)
			f.To = &end
		}
	}
	f.CategoryID = parseOptionalUint(c.Query("category_id"))
	if minStr := strings.TrimSpace(c.Query("min_amount")); minStr != "" {
		if v, err := strconv.ParseFloat(minStr, 64); err == nil {
			f.MinAmount = &v
		}
	}
	if maxStr := strings.TrimSpace(c.Query("max_amount")); maxStr != "" {
		if v, err := strconv.ParseFloat(maxStr, 64); err == nil {
			f.MaxAmount = &v
		}
	}
	return f
}

func clientIP(c *gin.Context) string {
	if ip := strings.TrimSpace(c.GetHeader("X-Forwarded-For")); ip != "" {
		if idx := strings.Index(ip, ","); idx > 0 {
			return strings.TrimSpace(ip[:idx])
		}
		return ip
	}
	return c.ClientIP()
}
