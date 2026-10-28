package handlers

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"

	"github.com/teslacost/teslacost/internal/apierror"
	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/middleware"
)

// maxIdempotentBody bounds the stored response size.
const maxIdempotentBody = 64 << 10

type withheldKey struct{}

// withholdFromIdempotency tells the Idempotency middleware not to keep the response of this request. A handler calls
// it before answering with a secret that is shown once (a new API token, an authorization code): the stored copy would
// outlive the secret and sit in the database for the whole retention of the keys. The cost is that a retry with the
// same key runs the request again instead of replaying the answer.
func withholdFromIdempotency(r *http.Request) {
	if flag, ok := r.Context().Value(withheldKey{}).(*bool); ok {
		*flag = true
	}
}

type recordingWriter struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (w *recordingWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *recordingWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	if w.body.Len()+len(b) <= maxIdempotentBody {
		w.body.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

// Idempotency replays the stored response of a mutation already processed with the same Idempotency-Key,
// so that requests queued offline and resent after a lost response are applied only once.
// It must run after authentication.
func Idempotency(repo *database.Repository) func(http.Handler) http.Handler {
	var locks idempotencyLocks
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("Idempotency-Key")
			if key == "" || len(key) > 100 || r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}
			userID := middleware.GetUserID(r.Context())
			// Reserve this user's key through lookup, mutation and response persistence.
			// A concurrent retry must look up the response only after its first writer finishes.
			release, err := locks.acquire(r.Context(), idempotencyRequestKey{userID: userID, key: key})
			if err != nil {
				return
			}
			defer release()

			stored, err := repo.GetIdempotentResponse(r.Context(), userID, key)
			if err != nil {
				slog.ErrorContext(r.Context(), "idempotency lookup failed", "component", "idempotency", "error", err)
				writeAPIError(w, http.StatusInternalServerError, apierror.New("idempotency.check_failed", "Idempotency check failed"))
				return
			}
			if stored != nil {
				if stored.Method != r.Method || stored.Path != r.URL.Path {
					writeAPIError(w, http.StatusUnprocessableEntity, apierror.New("idempotency.key_reused", "Idempotency-Key already used for another request"))
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Idempotent-Replay", "true")
				w.WriteHeader(stored.StatusCode)
				_, _ = w.Write(stored.Body)
				return
			}

			rec := &recordingWriter{ResponseWriter: w}
			withheld := new(bool)
			next.ServeHTTP(rec, r.WithContext(context.WithValue(r.Context(), withheldKey{}, withheld)))
			if rec.status >= 200 && rec.status < 300 && !*withheld {
				if err := repo.SaveIdempotentResponse(r.Context(), userID, key, database.StoredResponse{
					Method: r.Method, Path: r.URL.Path, StatusCode: rec.status, Body: rec.body.Bytes(),
				}); err != nil {
					slog.ErrorContext(r.Context(), "idempotency store failed", "component", "idempotency", "error", err)
				}
			}
		})
	}
}
