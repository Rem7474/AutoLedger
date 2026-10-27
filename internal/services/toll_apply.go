package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
	"github.com/teslacost/teslacost/internal/servertext"
	"github.com/teslacost/teslacost/internal/tolldata"
)

// TollDatasetCurrency is the currency the toll dataset (French autoroutes, OpenTollData) is priced in. An estimate is
// only recorded for a vehicle that keeps its costs in that currency: there is no exchange rate to convert it with, and
// an expense in another currency than its vehicle's without an fx_rate stays out of the cost ledger.
const TollDatasetCurrency = "EUR"

// MaxBulkTollDrives bounds a bulk application: each drive triggers a TeslaMateAPI call.
const MaxBulkTollDrives = 100

// ApplyStatus is the outcome of applying the estimated toll to one drive.
type ApplyStatus string

const (
	ApplyCreated        ApplyStatus = "created"
	ApplyUpdated        ApplyStatus = "updated"
	ApplySkippedManual  ApplyStatus = "skipped_manual"
	ApplySkippedGroup   ApplyStatus = "skipped_trip_group"
	ApplySkippedNoPrice ApplyStatus = "skipped_no_price"
	ApplySkippedNoGPS   ApplyStatus = "skipped_no_gps"
	// ApplySkippedCurrency is a vehicle whose currency is not the dataset's: its tolls are entered by hand.
	ApplySkippedCurrency ApplyStatus = "skipped_currency"
	ApplyFailed          ApplyStatus = "failed"
)

// ApplyResult reports what happened for one drive.
type ApplyResult struct {
	DriveID string       `json:"drive_id"`
	Status  ApplyStatus  `json:"status"`
	Amount  *money.Cents `json:"amount,omitempty"`
	Error   string       `json:"error,omitempty"`
}

// BulkApplyResult aggregates per-drive outcomes.
type BulkApplyResult struct {
	Created          int           `json:"created"`
	Updated          int           `json:"updated"`
	SkippedManual    int           `json:"skipped_manual"`
	SkippedTripGroup int           `json:"skipped_trip_group"`
	SkippedNoPrice   int           `json:"skipped_no_price"`
	SkippedNoGPS     int           `json:"skipped_no_gps"`
	SkippedCurrency  int           `json:"skipped_currency"`
	Failed           int           `json:"failed"`
	Results          []ApplyResult `json:"results"`
}

type tollAction int

const (
	tollActionCreate tollAction = iota
	tollActionUpdate
	tollActionSkipManual
	tollActionSkipGroup
)

// decideTollAction inspects the TOLL expenses already covering a drive. A manual toll is never
// overwritten, trip-group tolls are never touched, and an earlier auto toll is updated in place.
func decideTollAction(existing []models.DriveExpense) (tollAction, string) {
	var autoID string
	for _, e := range existing {
		if e.Type != "TOLL" {
			continue
		}
		if e.TripGroupID != nil {
			return tollActionSkipGroup, ""
		}
		if e.Source != models.ExpenseSourceAutoToll {
			return tollActionSkipManual, ""
		}
		autoID = e.ID
	}
	if autoID != "" {
		return tollActionUpdate, autoID
	}
	return tollActionCreate, ""
}

// sumEstimatedPrice totals the priced segments; ok is false when none has a price.
func sumEstimatedPrice(segments []models.TollSegment) (money.Cents, bool) {
	var total money.Cents
	priced := false
	for _, s := range segments {
		if s.EstimatedPrice != nil {
			total += *s.EstimatedPrice
			priced = true
		}
	}
	return total, priced && total > 0
}

// autoTollNotes describes the detected crossings, e.g. "Auto toll: A → B, Barrier C", in the
// acting user's language: the result is stored as plain text on the expense, so there is no
// code left for the frontend to translate later.
func autoTollNotes(lang string, segments []models.TollSegment) string {
	parts := make([]string, 0, len(segments))
	for _, s := range segments {
		part := ""
		switch {
		case s.Type == "close" && s.Exit != nil:
			part = s.Entry + " → " + *s.Exit
		case s.Type == "close":
			part = s.Entry + " " + servertext.Text(lang, "toll.unidentified_exit")
		default:
			part = servertext.Text(lang, "toll.barrier") + " " + s.Entry
		}
		switch {
		case s.Spans > 1 && s.FullPrice != nil:
			amount := s.FullPrice.String()
			if lang == "fr" {
				amount = strings.Replace(amount, ".", ",", 1)
			}
			part += " " + servertext.Text(lang, "toll.shared_ticket", amount, s.Spans)
		case s.Spans > 1:
			part += " " + servertext.Text(lang, "toll.shared")
		}
		parts = append(parts, part)
	}
	return servertext.Text(lang, "toll.auto_prefix") + strings.Join(parts, ", ")
}

// language is the stored language of userID ("en" or "fr", defaulting to "en"), used for the
// notes attached to an auto-detected toll expense and the bulk-apply error text: both are plain
// strings with no request/response cycle left for the frontend to translate through.
func (s *TollDetectionService) language(ctx context.Context, userID string) string {
	if userID == "" {
		return "en"
	}
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil || user == nil || user.Language != "fr" {
		return "en"
	}
	return "fr"
}

// sharedTicketPaidByHand reports whether a ticket of the drive that also runs over other drives of its journey has a
// toll entered by hand (or for a trip group) on one of them: the owner already recorded that journey, and an
// automatic share on this drive would count it twice.
func (s *TollDetectionService) sharedTicketPaidByHand(ctx context.Context, vehicle *models.Vehicle, chain *tollChain) (bool, error) {
	for i, other := range chain.drives {
		if i == chain.target || !sharesATicket(chain.targetSegs, chain.result.PerDrive[i]) {
			continue
		}
		existing, err := s.repo.GetDriveExpensesByDriveID(ctx, vehicle.ID, other.ID, "en")
		if err != nil {
			return false, err
		}
		if action, _ := decideTollAction(existing); action == tollActionSkipManual || action == tollActionSkipGroup {
			return true, nil
		}
	}
	return false, nil
}

// sharesATicket reports whether both drives carry a ticket spanning several drives at the same place on the trace.
func sharesATicket(a, b []tolldata.DriveSegment) bool {
	for _, x := range a {
		for _, y := range b {
			if x.Spans > 1 && y.Spans > 1 && x.StartMeters == y.StartMeters && x.EndMeters == y.EndMeters {
				return true
			}
		}
	}
	return false
}

// ApplyTollEstimate re-detects the drive's tolls and records the estimated total as an
// AUTO_TOLL expense, without ever overwriting a manual or trip-group toll.
func (s *TollDetectionService) ApplyTollEstimate(ctx context.Context, vehicle *models.Vehicle, driveID string) (ApplyResult, error) {
	res := ApplyResult{DriveID: driveID}
	if vehicle.Currency != TollDatasetCurrency {
		res.Status = ApplySkippedCurrency
		return res, nil
	}

	drive, err := s.repo.GetDriveByID(ctx, driveID, vehicle.ID)
	if err != nil {
		return res, err
	}

	// decideTollAction only looks at Type/TripGroupID/Source, never the translated DriveTitle fallback.
	existing, err := s.repo.GetDriveExpensesByDriveID(ctx, vehicle.ID, driveID, "en")
	if err != nil {
		return res, err
	}
	action, expenseID := decideTollAction(existing)
	switch action {
	case tollActionSkipManual:
		res.Status = ApplySkippedManual
		return res, nil
	case tollActionSkipGroup:
		res.Status = ApplySkippedGroup
		return res, nil
	}

	chain, err := s.detectChain(ctx, vehicle, driveID)
	if err != nil {
		if errors.Is(err, ErrNoGPSTrace) {
			res.Status = ApplySkippedNoGPS
			return res, nil
		}
		return res, err
	}
	detection := chain.persisted

	covered, err := s.sharedTicketPaidByHand(ctx, vehicle, chain)
	if err != nil {
		return res, err
	}
	if covered {
		res.Status = ApplySkippedManual
		return res, nil
	}

	total, ok := sumEstimatedPrice(detection.Segments)
	if !ok {
		res.Status = ApplySkippedNoPrice
		return res, nil
	}

	notes := autoTollNotes(s.language(ctx, vehicle.UserID), detection.Segments)
	exp := &models.DriveExpense{
		ID:        expenseID,
		VehicleID: vehicle.ID,
		DriveID:   &drive.ID,
		Type:      "TOLL",
		Amount:    total,
		Currency:  TollDatasetCurrency,
		Date:      drive.StartTime,
		Notes:     &notes,
		Source:    models.ExpenseSourceAutoToll,
	}
	if err := s.repo.SaveDriveExpense(ctx, exp, nil, ""); err != nil {
		return res, fmt.Errorf("failed to save auto toll expense: %w", err)
	}

	res.Amount = &total
	res.Status = ApplyCreated
	if action == tollActionUpdate {
		res.Status = ApplyUpdated
	}
	return res, nil
}

// ApplyTollEstimatesBulk applies the estimate to each drive in turn; a failure on one drive
// is recorded and does not stop the others.
func (s *TollDetectionService) ApplyTollEstimatesBulk(ctx context.Context, vehicle *models.Vehicle, driveIDs []string) BulkApplyResult {
	lang := s.language(ctx, vehicle.UserID)
	out := BulkApplyResult{Results: make([]ApplyResult, 0, len(driveIDs))}
	for _, id := range driveIDs {
		res, err := s.ApplyTollEstimate(ctx, vehicle, id)
		if err != nil {
			res.Status = ApplyFailed
			if errors.Is(err, database.ErrNotFound) {
				res.Error = servertext.Text(lang, "toll.trip_not_found")
			} else {
				res.Error = servertext.Text(lang, "toll.detection_failed")
			}
		}
		switch res.Status {
		case ApplyCreated:
			out.Created++
		case ApplyUpdated:
			out.Updated++
		case ApplySkippedManual:
			out.SkippedManual++
		case ApplySkippedGroup:
			out.SkippedTripGroup++
		case ApplySkippedNoPrice:
			out.SkippedNoPrice++
		case ApplySkippedNoGPS:
			out.SkippedNoGPS++
		case ApplySkippedCurrency:
			out.SkippedCurrency++
		default:
			out.Failed++
		}
		out.Results = append(out.Results, res)
	}
	return out
}
