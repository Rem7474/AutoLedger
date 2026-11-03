package database

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/teslacost/teslacost/internal/models"
)

// Vehicle ownership/financing contracts (loan, lease, cash purchase).
// GetVehicleOwnership returns the ownership contract of a vehicle, or ErrNotFound.
func (r *Repository) GetVehicleOwnership(ctx context.Context, vehicleID string) (*models.VehicleOwnership, error) {
	var o models.VehicleOwnership
	err := scanOwnership(r.pool.QueryRow(ctx, `SELECT `+ownershipColumns+` FROM vehicle_ownership WHERE vehicle_id = $1;`, vehicleID), &o)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &o, nil
}

// SaveVehicleOwnership creates or replaces the ownership contract of a vehicle. A replace that names the version it read
// is refused with ErrStaleWrite when the contract has been saved since.
func (r *Repository) SaveVehicleOwnership(ctx context.Context, o *models.VehicleOwnership) error {
	err := scanOwnership(r.pool.QueryRow(ctx, `
		INSERT INTO vehicle_ownership (
			vehicle_id, acquisition_type, start_date, start_odometer,
			purchase_price, purchase_fees, incentives, expected_resale_value, expected_holding_months,
			loan_amount, loan_rate_pct, loan_duration_months, loan_fees, loan_insurance_monthly,
			lease_down_payment, lease_monthly_rent, lease_duration_months, lease_fees, lease_deposit,
			lease_km_allowance_per_year, lease_excess_km_price, lease_end_fees_estimate, lease_purchase_option_price,
			lease_includes_maintenance, lease_includes_insurance, lease_includes_tires, option_exercised_date,
			end_date, sale_price
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19,
		          $20, $21, $22, $23, $24, $25, $26, $27, $28, $29)
		ON CONFLICT (vehicle_id) DO UPDATE SET
			acquisition_type = EXCLUDED.acquisition_type, start_date = EXCLUDED.start_date,
			start_odometer = EXCLUDED.start_odometer, purchase_price = EXCLUDED.purchase_price,
			purchase_fees = EXCLUDED.purchase_fees, incentives = EXCLUDED.incentives,
			expected_resale_value = EXCLUDED.expected_resale_value, expected_holding_months = EXCLUDED.expected_holding_months,
			loan_amount = EXCLUDED.loan_amount, loan_rate_pct = EXCLUDED.loan_rate_pct,
			loan_duration_months = EXCLUDED.loan_duration_months, loan_fees = EXCLUDED.loan_fees,
			loan_insurance_monthly = EXCLUDED.loan_insurance_monthly, lease_down_payment = EXCLUDED.lease_down_payment,
			lease_monthly_rent = EXCLUDED.lease_monthly_rent, lease_duration_months = EXCLUDED.lease_duration_months,
			lease_fees = EXCLUDED.lease_fees, lease_deposit = EXCLUDED.lease_deposit,
			lease_km_allowance_per_year = EXCLUDED.lease_km_allowance_per_year,
			lease_excess_km_price = EXCLUDED.lease_excess_km_price, lease_end_fees_estimate = EXCLUDED.lease_end_fees_estimate,
			lease_purchase_option_price = EXCLUDED.lease_purchase_option_price,
			lease_includes_maintenance = EXCLUDED.lease_includes_maintenance,
			lease_includes_insurance = EXCLUDED.lease_includes_insurance, lease_includes_tires = EXCLUDED.lease_includes_tires,
			option_exercised_date = EXCLUDED.option_exercised_date, end_date = EXCLUDED.end_date,
			sale_price = EXCLUDED.sale_price, version = vehicle_ownership.version + 1, updated_at = NOW()
		WHERE $30::bigint IS NULL OR vehicle_ownership.version = $30::bigint
		RETURNING `+ownershipColumns+`;
	`,
		o.VehicleID, o.AcquisitionType, o.StartDate, o.StartOdometer,
		o.PurchasePrice, o.PurchaseFees, o.Incentives, o.ExpectedResaleValue, o.ExpectedHoldingMonths,
		o.LoanAmount, o.LoanRatePct, o.LoanDurationMonths, o.LoanFees, o.LoanInsuranceMonthly,
		o.LeaseDownPayment, o.LeaseMonthlyRent, o.LeaseDurationMonths, o.LeaseFees, o.LeaseDeposit,
		o.LeaseKmAllowancePerYear, o.LeaseExcessKmPrice, o.LeaseEndFeesEstimate, o.LeasePurchaseOptionPrice,
		o.LeaseIncludesMaintenance, o.LeaseIncludesInsurance, o.LeaseIncludesTires, o.OptionExercisedDate,
		o.EndDate, o.SalePrice, o.ExpectedVersion,
	), o)
	// The conflict clause matched a contract whose version moved on: nothing was written and nothing returned.
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrStaleWrite
	}
	return err
}

// DeleteVehicleOwnership removes the ownership contract of a vehicle.
func (r *Repository) DeleteVehicleOwnership(ctx context.Context, vehicleID string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM vehicle_ownership WHERE vehicle_id = $1;`, vehicleID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
