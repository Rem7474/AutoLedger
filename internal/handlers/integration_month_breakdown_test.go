package handlers

import (
	"testing"
	"time"

	"github.com/teslacost/teslacost/internal/services"
)

func TestIntegrationMonthBreakdownListsTheRecordsWithoutIdentifiersOrPositions(t *testing.T) {
	at := time.Date(2026, 3, 9, 18, 30, 0, 0, time.UTC)
	out := buildIntegrationMonthBreakdown("v1", "CHF", services.MonthlyContributions{
		Month: "2026-03", Category: services.ContributionEnergy, Total: 9500,
		Items: []services.MonthlyContributionItem{
			{Date: at, SourceTable: "charge_logs", SourceID: "secret-id", Label: "Rue de la Gare 1, Lausanne", Amount: 4000},
			{Date: at, SourceTable: "charge_logs", SourceID: "secret-id-2", Label: "46.5197, 6.6323", Amount: 1500},
			{Date: at, SourceTable: "maintenance_expenses", SourceID: "secret-id-3", Label: "  ", Amount: 3000, Recurring: true, IntervalMonths: 12},
		},
		Estimate: &services.MonthlyContributionEstimate{Amount: 1000, Km: 120, Kwh: 20},
	})

	if out.Category != "energy" || out.Total.Amount != "95.00" || out.Matching != 3 || out.Truncated || len(out.Entries) != 3 {
		t.Fatalf("breakdown: %+v", out)
	}
	if e := out.Entries[0]; e.Kind != "charge" || e.Label == nil || *e.Label != "Rue de la Gare 1, Lausanne" || e.Amount.Amount != "40.00" || e.Date != "2026-03-09T18:30:00Z" {
		t.Errorf("a charge: %+v", e)
	}
	if out.Entries[1].Label != nil {
		t.Errorf("a charge located only by a position has no label: %+v", out.Entries[1])
	}
	if e := out.Entries[2]; e.Kind != "expense" || e.Label != nil || !e.Recurring || e.IntervalMonths == nil || *e.IntervalMonths != 12 {
		t.Errorf("a recurring expense without a description: %+v", e)
	}
	if out.Estimate == nil || out.Estimate.Amount.Amount != "10.00" || out.Estimate.EnergyKwh != 20 {
		t.Errorf("estimate: %+v", out.Estimate)
	}
}

func TestIntegrationMonthBreakdownCapsTheListAndSaysSo(t *testing.T) {
	items := make([]services.MonthlyContributionItem, integrationMonthEntriesMax+5)
	out := buildIntegrationMonthBreakdown("v1", "EUR", services.MonthlyContributions{Month: "2026-03", Category: services.ContributionInsurance, Items: items})
	if len(out.Entries) != integrationMonthEntriesMax || out.Matching != len(items) || !out.Truncated || out.Estimate != nil {
		t.Errorf("cap: %d listed, %d matching, truncated %v", len(out.Entries), out.Matching, out.Truncated)
	}
	if none := buildIntegrationMonthBreakdown("v1", "EUR", services.MonthlyContributions{Month: "2026-03", Category: services.ContributionInsurance}); none.Entries == nil {
		t.Error("no record is an empty list")
	}
}
