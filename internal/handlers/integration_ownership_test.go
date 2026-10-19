package handlers

import (
	"testing"
	"time"

	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
	"github.com/teslacost/teslacost/internal/services"
)

func leasedSummary() *services.TCOSummary {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC)
	months := 36
	rent, option := money.Cents(35000), money.Cents(1800000)
	perYear, total, price := 10000.0, 30000.0, 0.12
	perKm := 0.52
	return &services.TCOSummary{
		AcquisitionType: models.AcquisitionLOA, FullCost: 2500000, FullCostPerKm: &perKm, DepreciationKnown: true, DistanceBasisKm: 24000,
		ContractStartDate: &start, ContractEndDate: &end, ContractDurationMonths: &months,
		LeaseMonthlyRent: &rent, LeasePurchaseOptionPrice: &option, LeaseIncludesMaintenance: true,
		LeaseKmAllowancePerYear: &perYear, LeaseKmAllowanceTotal: &total, LeaseKmAllowanceToDate: 20000, LeaseKmDriven: 24000,
		LeaseExcessKmPrice: &price, LeaseExcessKmCost: 48000, LeaseExcessKmProjected: 150000,
	}
}

func TestIntegrationOwnershipSaysWhereALeaseStandsAgainstItsMileage(t *testing.T) {
	v := &models.Vehicle{ID: "v1", Currency: "CHF"}
	out := buildIntegrationOwnership(v, leasedSummary(), nil, "Set the purchase price first")

	if out.AcquisitionType == nil || *out.AcquisitionType != "LOA" || out.Economic.FullCost != (IntegrationAmount{Amount: "25000.00", Currency: "CHF"}) ||
		out.Economic.FullCostPerKm == nil || out.Economic.FullCostPerKm.Amount != "0.5200" || !out.Economic.DepreciationKnown {
		t.Fatalf("economic cost: %+v", out)
	}
	c := out.Contract
	if c == nil || c.Start == nil || *c.Start != "2025-01-01" || *c.End != "2028-01-01" || c.MonthlyRent == nil || c.MonthlyRent.Amount != "350.00" ||
		c.PurchaseOption == nil || c.PurchaseOption.Amount != "18000.00" || !c.IncludesMaintenance || c.IncludesInsurance || c.DownPayment != nil || c.OptionExercisedOn != nil {
		t.Fatalf("contract: %+v", c)
	}
	m := c.Mileage
	if m == nil || m.Driven.Value != 24000 || m.AllowanceToDate.Value != 20000 || m.AllowanceTotal.Value != 30000 ||
		m.ExcessPrice == nil || m.ExcessPrice.Amount != "0.1200" || m.ExcessAccrued.Amount != "480.00" || m.ExcessProjected.Amount != "1500.00" {
		t.Errorf("mileage allowance used and its excess: %+v", m)
	}
	if out.Resale != nil || out.ResaleUnavailable == nil || *out.ResaleUnavailable != "Set the purchase price first" {
		t.Errorf("a missing estimate says why: %+v %v", out.Resale, out.ResaleUnavailable)
	}
}

func TestIntegrationOwnershipOfACashPurchaseHasNoContractAndTheResaleEstimate(t *testing.T) {
	v := &models.Vehicle{ID: "v1", Currency: "EUR"}
	sum := &services.TCOSummary{AcquisitionType: models.AcquisitionCash, AcquisitionCost: 3000000, DepreciationCost: 1000000, FullCost: 1500000}
	health := 92.5
	residual := &models.ResidualValue{PurchasePrice: 3000000, ExpectedValue: 1500000, CurrentValue: 2000000, Depreciation: 1000000,
		HoldingMonths: 60, AgeMonths: 24, Progress: 0.4, HealthPercent: &health}
	out := buildIntegrationOwnership(v, sum, residual, "")

	if out.Contract != nil || out.Acquisition.Amount != "30000.00" || out.Economic.DepreciationKnown || out.Economic.Depreciation.Amount != "10000.00" {
		t.Errorf("a bought vehicle has no contract; unknown depreciation is flagged: %+v", out)
	}
	r := out.Resale
	if r == nil || r.CurrentValue.Amount != "20000.00" || r.StartValue.Amount != "30000.00" || r.ExpectedValue.Amount != "15000.00" ||
		r.Depreciation.Amount != "10000.00" || r.HealthPercent == nil || *r.HealthPercent != 92.5 || r.UnderLease || r.OptionMargin != nil || out.ResaleUnavailable != nil {
		t.Errorf("resale: %+v", r)
	}

	none := buildIntegrationOwnership(v, &services.TCOSummary{}, nil, "")
	if none.AcquisitionType != nil || none.Resale != nil || none.ResaleUnavailable != nil {
		t.Errorf("without an acquisition nothing is invented: %+v", none)
	}
}
