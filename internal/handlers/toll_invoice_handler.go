package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/teslacost/teslacost/internal/apierror"
	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
)

// TollInvoiceHandler manages the monthly toll invoices of a vehicle. A month with an invoice counts its total, and
// only that, as the tolls of the vehicle in the monthly and global totals.
type TollInvoiceHandler struct {
	calendarDates
	repo *database.Repository
}

func NewTollInvoiceHandler(repo *database.Repository) *TollInvoiceHandler {
	return &TollInvoiceHandler{repo: repo}
}

// tollInvoiceMonthBounds returns the instants a month key covers in the reporting timezone: [start, end).
func tollInvoiceMonthBounds(month string, loc *time.Location) (start, end time.Time, err error) {
	first, perr := time.ParseInLocation("2006-01", strings.TrimSpace(month), loc)
	if perr != nil {
		return start, end, apierror.New("toll_invoice.invalid_month", "Invalid month: expected YYYY-MM")
	}
	return first, first.AddDate(0, 1, 0), nil
}

type tollInvoiceList struct {
	Currency string               `json:"currency"`
	Invoices []models.TollInvoice `json:"invoices"`
}

// List returns the invoices of the vehicle, the latest month first, each with the tolls entered individually in its
// month and the gap between the two.
func (h *TollInvoiceHandler) List(w http.ResponseWriter, r *http.Request) {
	v := requireVehicleAccess(w, r, h.repo, chi.URLParam(r, "vehicleId"), models.RoleViewer)
	if v == nil {
		return
	}
	invoices, err := h.repo.ListTollInvoices(r.Context(), v.ID)
	if err != nil {
		writeRepoError(w, r, err, "Failed to list toll invoices")
		return
	}
	writeJSON(w, http.StatusOK, tollInvoiceList{Currency: v.Currency, Invoices: invoices})
}

// SaveTollInvoiceRequest is the invoice of a month. The total is in the vehicle's currency.
type SaveTollInvoiceRequest struct {
	Total           money.Cents `json:"total"`
	Note            *string     `json:"note"`
	ExpectedVersion *int64      `json:"expected_version"` // The version the client read; absent: not checked
}

// Save creates the invoice of a month or replaces it. An invoice of 0 is valid: it says the month had no toll.
func (h *TollInvoiceHandler) Save(w http.ResponseWriter, r *http.Request) {
	v := requireVehicleAccess(w, r, h.repo, chi.URLParam(r, "vehicleId"), models.RoleEditor)
	if v == nil {
		return
	}
	var req SaveTollInvoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, apierror.New("request.invalid_body", "Invalid request body"))
		return
	}
	month := chi.URLParam(r, "month")
	loc := h.dateLocation()
	start, end, err := tollInvoiceMonthBounds(month, loc)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if start.After(time.Now().In(loc)) {
		writeAPIError(w, http.StatusBadRequest, apierror.New("toll_invoice.future_month", "A month that has not started cannot have an invoice"))
		return
	}
	if req.Total < 0 || req.Total > money.Max {
		writeAPIError(w, http.StatusBadRequest, apierror.New("toll_invoice.invalid_total", "Invalid invoice total"))
		return
	}
	var note *string
	if req.Note != nil {
		if t := strings.TrimSpace(*req.Note); t != "" {
			note = &t
		}
	}
	saved, err := h.repo.SaveTollInvoice(r.Context(), v.ID, start.Format("2006-01"), start, end, req.Total, note, req.ExpectedVersion)
	if err != nil {
		writeRepoError(w, r, err, "Failed to save the toll invoice")
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

// Delete removes the invoice of a month: the tolls entered individually count again.
func (h *TollInvoiceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	v := requireVehicleAccess(w, r, h.repo, chi.URLParam(r, "vehicleId"), models.RoleEditor)
	if v == nil {
		return
	}
	month := chi.URLParam(r, "month")
	if _, _, err := tollInvoiceMonthBounds(month, h.dateLocation()); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	var expected *int64
	if raw := r.URL.Query().Get("expected_version"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, apierror.New("request.invalid_version", "Invalid expected_version"))
			return
		}
		expected = &n
	}
	if err := h.repo.DeleteTollInvoice(r.Context(), v.ID, month, expected); err != nil {
		writeRepoError(w, r, err, "Failed to delete the toll invoice")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
