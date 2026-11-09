package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/teslacost/teslacost/internal/apierror"
	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
	"github.com/teslacost/teslacost/internal/servertext"
	"github.com/teslacost/teslacost/internal/storage"
)

type ExpenseHandler struct {
	repo           *database.Repository
	storageService *storage.FileStorageService
}

func NewExpenseHandler(repo *database.Repository, storageService *storage.FileStorageService) *ExpenseHandler {
	return &ExpenseHandler{repo: repo, storageService: storageService}
}

type CreateDriveExpenseRequest struct {
	ExpectedVersion *int64      `json:"expected_version"` // The version the client read; absent: not checked
	TripGroupID     *string     `json:"trip_group_id"`
	DriveID         *string     `json:"drive_id"`
	DriveIDs        []string    `json:"drive_ids"`
	Type            string      `json:"type"` // TOLL, PARKING, etc.
	Amount          money.Cents `json:"amount"`
	Currency        string      `json:"currency"`
	FxRate          *float64    `json:"fx_rate"`
	Date            string      `json:"date"`
	Notes           *string     `json:"notes"`
	DocumentID      *string     `json:"document_id"`
}

// buildDriveExpense validates a drive expense payload.
func buildDriveExpense(vehicleID, baseCurrency string, req *CreateDriveExpenseRequest) (*models.DriveExpense, error) {
	if err := validateAmount(req.Amount, false); err != nil {
		return nil, err
	}
	expDate, err := parseDate(req.Date)
	if err != nil {
		return nil, err
	}
	curr, fxRate, err := normalizeCurrency(req.Currency, baseCurrency, req.FxRate)
	if err != nil {
		return nil, err
	}
	expType := strings.ToUpper(strings.TrimSpace(req.Type))
	if expType == "" {
		expType = "TOLL"
	}
	if !driveExpenseTypes[expType] {
		return nil, apierror.New("expense.type_invalid", "Invalid expense type")
	}

	return &models.DriveExpense{
		ExpectedVersion: req.ExpectedVersion,
		VehicleID:       vehicleID,
		TripGroupID:     req.TripGroupID,
		DriveID:         req.DriveID,
		Type:            expType,
		Amount:          req.Amount,
		Currency:        curr,
		FxRate:          fxRate,
		Date:            expDate,
		Notes:           req.Notes,
		DocumentID:      req.DocumentID,
	}, nil
}

func expenseGroupName(notes *string, lang string) string {
	if notes != nil && strings.TrimSpace(*notes) != "" {
		return *notes
	}
	return servertext.Text(lang, "expense.multi_leg_trip")
}

func (h *ExpenseHandler) CreateDriveExpense(w http.ResponseWriter, r *http.Request) {
	vehicleID := chi.URLParam(r, "vehicleId")
	v := requireVehicleAccess(w, r, h.repo, vehicleID, models.RoleEditor)
	if v == nil {
		return
	}

	var req CreateDriveExpenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, apierror.New("request.invalid_body", "Invalid request body"))
		return
	}

	exp, err := buildDriveExpense(vehicleID, v.Currency, &req)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}

	if err := h.repo.SaveDriveExpense(r.Context(), exp, req.DriveIDs, expenseGroupName(req.Notes, requestLanguage(r, h.repo))); err != nil {
		writeRepoError(w, r, err, "Failed to record expense")
		return
	}

	writeJSON(w, http.StatusCreated, exp)
}

func (h *ExpenseHandler) ListDriveExpenses(w http.ResponseWriter, r *http.Request) {
	vehicleID := chi.URLParam(r, "vehicleId")
	if v := requireVehicleAccess(w, r, h.repo, vehicleID, models.RoleViewer); v == nil {
		return
	}

	list, err := h.repo.ListDriveExpenses(r.Context(), vehicleID, requestLanguage(r, h.repo))
	if err != nil {
		writeRepoError(w, r, err, "Failed to list expenses")
		return
	}
	if list == nil {
		list = []models.DriveExpense{}
	}

	writeJSON(w, http.StatusOK, list)
}

// GetDriveExpense answers one travel expense of the vehicle, as the list gives it.
func (h *ExpenseHandler) GetDriveExpense(w http.ResponseWriter, r *http.Request) {
	vehicleID := chi.URLParam(r, "vehicleId")
	if v := requireVehicleAccess(w, r, h.repo, vehicleID, models.RoleViewer); v == nil {
		return
	}
	expense, err := h.repo.GetDriveExpense(r.Context(), vehicleID, chi.URLParam(r, "expenseId"), requestLanguage(r, h.repo))
	if err != nil {
		writeRepoError(w, r, err, "Failed to read the expense")
		return
	}
	writeJSON(w, http.StatusOK, expense)
}

func (h *ExpenseHandler) UpdateDriveExpense(w http.ResponseWriter, r *http.Request) {
	vehicleID := chi.URLParam(r, "vehicleId")
	expenseID := chi.URLParam(r, "expenseId")
	v := requireVehicleAccess(w, r, h.repo, vehicleID, models.RoleEditor)
	if v == nil {
		return
	}

	var req CreateDriveExpenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, apierror.New("request.invalid_body", "Invalid request body"))
		return
	}

	exp, err := buildDriveExpense(vehicleID, v.Currency, &req)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	exp.ID = expenseID

	if err := h.repo.SaveDriveExpense(r.Context(), exp, req.DriveIDs, expenseGroupName(req.Notes, requestLanguage(r, h.repo))); err != nil {
		writeRepoError(w, r, err, "Failed to update expense")
		return
	}

	writeJSON(w, http.StatusOK, exp)
}

func (h *ExpenseHandler) DeleteDriveExpense(w http.ResponseWriter, r *http.Request) {
	vehicleID := chi.URLParam(r, "vehicleId")
	expenseID := chi.URLParam(r, "expenseId")
	if v := requireVehicleAccess(w, r, h.repo, vehicleID, models.RoleEditor); v == nil {
		return
	}

	if err := h.repo.DeleteDriveExpense(r.Context(), vehicleID, expenseID); err != nil {
		writeRepoError(w, r, err, "Failed to delete expense")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Expense deleted successfully"})
}
