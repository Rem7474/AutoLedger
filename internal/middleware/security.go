package middleware

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

// DefaultCSP suits the embedded single-page app: everything comes from the same origin, styles may be inline (Vue
// binds style attributes), images may be data or blob URLs (receipt previews), and PDFs are previewed in a frame
// from a blob URL.
const DefaultCSP = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data: blob:; font-src 'self' data:; connect-src 'self'; " +
	"frame-src 'self' blob:; object-src 'self' blob:; " +
	"base-uri 'self'; form-action 'self'; frame-ancestors 'none'"

// SecurityHeaders sets the response headers that limit what a browser does with the pages and API answers.
// csp is the Content-Security-Policy value; empty leaves that header out. HSTS is sent only on HTTPS requests.
func SecurityHeaders(csp string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")
			h.Set("Cross-Origin-Opener-Policy", "same-origin")
			if csp != "" {
				h.Set("Content-Security-Policy", csp)
			}
			if IsHTTPS(r) {
				h.Set("Strict-Transport-Security", "max-age=31536000")
			}
			next.ServeHTTP(w, r)
		})
	}
}

// originOf reduces a URL to scheme://host, lower case, or "" when it is not one.
func originOf(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return strings.ToLower(u.Scheme + "://" + u.Host)
}

// OriginCheck rejects browser requests that change state from a page served by another site. The session lives in
// cookies, which a browser attaches to any request, so SameSite=Lax alone still lets a sibling subdomain forge
// them. Browsers always send Origin on such requests; it must be one of allowed or the host the request addressed.
// Requests without Origin (scripts, curl) and requests carrying an Authorization header (not sent automatically)
// are not affected.
func OriginCheck(allowed []string) func(http.Handler) http.Handler {
	allowedSet := map[string]bool{}
	for _, a := range allowed {
		if o := originOf(a); o != "" {
			allowedSet[o] = true
		}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
				next.ServeHTTP(w, r)
				return
			}
			origin := r.Header.Get("Origin")
			if origin == "" || r.Header.Get("Authorization") != "" || isOpenEndpoint(r) {
				next.ServeHTTP(w, r)
				return
			}
			scheme := "http"
			if IsHTTPS(r) {
				scheme = "https"
			}
			self := strings.ToLower(scheme + "://" + r.Host)
			if got := originOf(origin); got != "" && (allowedSet[got] || got == self) {
				next.ServeHTTP(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Request origin not allowed", "code": "request.origin_denied"})
		})
	}
}

// BodyLimit caps request bodies so a client cannot make the server buffer an arbitrarily large JSON document.
// Multipart uploads are left to their handler, which applies its own, larger limit.
func BodyLimit(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil && !strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "multipart/") {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}
