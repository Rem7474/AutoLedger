package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/teslacost/teslacost/internal/apierror"
	"github.com/teslacost/teslacost/internal/auth"
	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/middleware"
)

// currentFamily returns the session (refresh token family) of the request, or "" for a client that does not send
// the refresh cookie (a script using a bearer token).
func (h *AuthHandler) currentFamily(r *http.Request) string {
	cookie, err := r.Cookie(refreshTokenCookie)
	if err != nil || cookie.Value == "" {
		return ""
	}
	token, err := h.repo.GetRefreshTokenByHash(r.Context(), auth.HashRefreshToken(cookie.Value))
	if err != nil {
		return ""
	}
	// A token of someone else's session does not make it this user's current one
	if token.UserID != middleware.GetUserID(r.Context()) {
		return ""
	}
	return token.FamilyID
}

// ListSessions lists the devices signed in to the account, the current one flagged.
func (h *AuthHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	sessions, err := h.repo.ListSessions(r.Context(), userID)
	if err != nil {
		writeRepoError(w, r, err, "Could not list sessions")
		return
	}
	current := h.currentFamily(r)
	for i := range sessions {
		sessions[i].Current = current != "" && sessions[i].ID == current
	}
	writeJSON(w, http.StatusOK, sessions)
}

// RevokeSession signs a device out. Closing the current session also clears its cookies.
func (h *AuthHandler) RevokeSession(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "sessionId")
	current := h.currentFamily(r)

	found, err := h.repo.RevokeUserSession(r.Context(), userID, id)
	if err != nil {
		writeRepoError(w, r, err, "Could not revoke the session")
		return
	}
	if !found {
		writeAPIError(w, http.StatusNotFound, apierror.New("auth.session_not_found", "Session not found"))
		return
	}
	if id == current {
		h.clearRefreshTokenCookie(w)
		h.clearAccessTokenCookie(w)
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Session revoked"})
}

// LogoutAll signs every device out, the current one included.
func (h *AuthHandler) LogoutAll(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	revoked, err := h.repo.RevokeUserSessionsExcept(r.Context(), userID, "")
	if err != nil {
		writeRepoError(w, r, err, "Could not revoke the sessions")
		return
	}
	h.clearRefreshTokenCookie(w)
	h.clearAccessTokenCookie(w)
	writeJSON(w, http.StatusOK, map[string]any{"message": "All sessions revoked", "sessions_revoked": revoked})
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ChangePassword replaces the local password after checking the current one, and signs out every other device: the
// usual reason to change a password is that someone else may have it. Wrong answers are refused with 403, not 401,
// which the frontend would take for an expired session.
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	if h.cfg.OIDCEnabled && h.cfg.OIDCDisableLocalAuth {
		writeAPIError(w, http.StatusForbidden, apierror.New("auth.local_disabled_password", "Local authentication is disabled: manage your password with your SSO provider"))
		return
	}
	userID := middleware.GetUserID(r.Context())

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, apierror.New("request.invalid_body", "Invalid request body"))
		return
	}
	user, err := h.repo.GetUserByID(r.Context(), userID)
	if err != nil {
		writeAPIError(w, http.StatusUnauthorized, apierror.New("auth.unauthorized", "Unauthorized"))
		return
	}
	if user.PasswordHash == nil {
		writeAPIError(w, http.StatusBadRequest, apierror.New("auth.no_local_password", "This account has no local password: it signs in with SSO"))
		return
	}

	// Checking the current password is a guessing surface for a stolen session, so it shares the sign-in limit.
	if blocked, retryAfter := h.throttle.Blocked(user.Email); blocked {
		w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Seconds())+1))
		writeAPIError(w, http.StatusTooManyRequests, apierror.New("auth.too_many_attempts", "Too many attempts: try again later"))
		return
	}
	if !auth.CheckPassword(req.CurrentPassword, *user.PasswordHash) {
		h.throttle.Fail(user.Email)
		writeAPIError(w, http.StatusForbidden, apierror.New("auth.current_password_wrong", "Current password is incorrect"))
		return
	}
	h.throttle.Reset(user.Email)

	switch {
	case len(req.NewPassword) < 8:
		writeAPIError(w, http.StatusBadRequest, apierror.New("auth.new_password_too_short", "The new password must be at least 8 characters long"))
		return
	case len(req.NewPassword) > auth.MaxPasswordBytes:
		writeAPIError(w, http.StatusBadRequest, apierror.New("auth.new_password_too_long", "The new password must not exceed 72 bytes"))
		return
	case req.NewPassword == req.CurrentPassword:
		writeAPIError(w, http.StatusBadRequest, apierror.New("auth.password_unchanged", "The new password must differ from the current one"))
		return
	}

	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if err := h.repo.UpdatePasswordHash(r.Context(), userID, hash); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			writeAPIError(w, http.StatusUnauthorized, apierror.New("auth.unauthorized", "Unauthorized"))
			return
		}
		writeRepoError(w, r, err, "Could not change the password")
		return
	}

	// Without the refresh cookie this device cannot be told apart: every session is closed, this one included.
	revoked, err := h.repo.RevokeUserSessionsExcept(r.Context(), userID, h.currentFamily(r))
	if err != nil {
		slog.ErrorContext(r.Context(), "password changed but the other sessions could not be revoked", "component", "auth", "error", err)
		writeAPIError(w, http.StatusInternalServerError, apierror.New("auth.password_changed_sessions_failed", "Password changed, but the other devices could not be signed out"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "Password changed", "sessions_revoked": revoked})
}

type UpdateLanguageRequest struct {
	Language string `json:"language"`
}

// UpdateLanguage stores the language used for text the server builds outside a request (reminder
// webhooks, sync failure alerts, auto-generated toll notes): the UI's own language is chosen
// client-side and does not go through this endpoint.
func (h *AuthHandler) UpdateLanguage(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var req UpdateLanguageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, apierror.New("request.invalid_body", "Invalid request body"))
		return
	}
	if req.Language != "en" && req.Language != "fr" {
		writeAPIError(w, http.StatusBadRequest, apierror.New("account.language_invalid", "Language must be \"en\" or \"fr\""))
		return
	}
	if err := h.repo.UpdateUserLanguage(r.Context(), userID, req.Language); err != nil {
		writeRepoError(w, r, err, "Could not update the language")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Language updated"})
}

type UpdateDistanceUnitRequest struct {
	DistanceUnit string `json:"distance_unit"`
}

// UpdateDistanceUnit stores the unit ("km" or "mi") the frontend converts distances to for
// display and form input. Stored distances stay in km either way: this is a display preference,
// not a data migration.
func (h *AuthHandler) UpdateDistanceUnit(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var req UpdateDistanceUnitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, apierror.New("request.invalid_body", "Invalid request body"))
		return
	}
	if req.DistanceUnit != "km" && req.DistanceUnit != "mi" {
		writeAPIError(w, http.StatusBadRequest, apierror.New("account.distance_unit_invalid", "Distance unit must be \"km\" or \"mi\""))
		return
	}
	if err := h.repo.UpdateUserDistanceUnit(r.Context(), userID, req.DistanceUnit); err != nil {
		writeRepoError(w, r, err, "Could not update the distance unit")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Distance unit updated"})
}

type UpdateVolumeUnitRequest struct {
	VolumeUnit string `json:"volume_unit"`
}

// UpdateVolumeUnit stores the unit ("l", "gal_us" or "gal_uk") the frontend converts fuel volumes
// to for display and form input. Stored volumes stay in litres either way, like distances in km.
func (h *AuthHandler) UpdateVolumeUnit(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var req UpdateVolumeUnitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, apierror.New("request.invalid_body", "Invalid request body"))
		return
	}
	if req.VolumeUnit != "l" && req.VolumeUnit != "gal_us" && req.VolumeUnit != "gal_uk" {
		writeAPIError(w, http.StatusBadRequest, apierror.New("account.volume_unit_invalid", "Volume unit must be \"l\", \"gal_us\" or \"gal_uk\""))
		return
	}
	if err := h.repo.UpdateUserVolumeUnit(r.Context(), userID, req.VolumeUnit); err != nil {
		writeRepoError(w, r, err, "Could not update the volume unit")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Volume unit updated"})
}
