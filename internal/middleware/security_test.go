package middleware

import (
	"github.com/teslacost/teslacost/internal/config"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var ok = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })

func TestSecurityHeadersOnlyPinsHTTPSOnHTTPS(t *testing.T) {
	trusted, _ := ParseTrustedProxies(config.DefaultTrustedProxies)
	h := ClientIP(trusted)(SecurityHeaders(DefaultCSP)(ok))

	plain := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.5:1"
	h.ServeHTTP(plain, req)
	for _, name := range []string{"X-Content-Type-Options", "X-Frame-Options", "Referrer-Policy", "Content-Security-Policy", "Permissions-Policy"} {
		if plain.Header().Get(name) == "" {
			t.Errorf("%s missing", name)
		}
	}
	if plain.Header().Get("Strict-Transport-Security") != "" {
		t.Error("HSTS must not be sent over plain HTTP: it would pin a site that cannot serve HTTPS")
	}

	viaProxy := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "172.18.0.2:1"
	req.Header.Set("X-Forwarded-Proto", "https")
	h.ServeHTTP(viaProxy, req)
	if viaProxy.Header().Get("Strict-Transport-Security") == "" {
		t.Error("HSTS expected behind a trusted TLS-terminating proxy")
	}
}

func TestSecurityHeadersWithoutCSP(t *testing.T) {
	rec := httptest.NewRecorder()
	SecurityHeaders("")(ok).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Header().Get("Content-Security-Policy") != "" {
		t.Error("an empty policy must leave the header out")
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("the other headers stay")
	}
}

func TestOriginCheck(t *testing.T) {
	h := OriginCheck([]string{"https://teslacost.example.org"})(ok)
	do := func(method, host, origin string, headers map[string]string) int {
		req := httptest.NewRequest(method, "http://"+host+"/api/x", nil)
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}

	tests := []struct {
		name    string
		method  string
		host    string
		origin  string
		headers map[string]string
		want    int
	}{
		{"same site, configured origin", "POST", "teslacost.example.org", "https://teslacost.example.org", nil, 204},
		{"another site", "POST", "teslacost.example.org", "https://evil.example.net", nil, 403},
		{"sibling subdomain", "DELETE", "teslacost.example.org", "https://blog.example.org", nil, 403},
		{"opaque origin", "PUT", "teslacost.example.org", "null", nil, 403},
		{"same host as the request", "POST", "192.168.1.20:8080", "http://192.168.1.20:8080", nil, 204},
		{"scheme differs", "POST", "192.168.1.20:8080", "https://192.168.1.20:8080", nil, 403},
		{"no origin (script or curl)", "POST", "teslacost.example.org", "", nil, 204},
		{"safe method is never checked", "GET", "teslacost.example.org", "https://evil.example.net", nil, 204},
		{"bearer token is not ambient", "POST", "teslacost.example.org", "https://evil.example.net", map[string]string{"Authorization": "Bearer x"}, 204},
		{"case of the host does not matter", "POST", "teslacost.example.org", "https://TeslaCost.Example.org", nil, 204},
	}
	for _, tt := range tests {
		if got := do(tt.method, tt.host, tt.origin, tt.headers); got != tt.want {
			t.Errorf("%s: got %d, want %d", tt.name, got, tt.want)
		}
	}
}

func TestBodyLimit(t *testing.T) {
	read := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.Copy(io.Discard, r.Body); err != nil {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	h := BodyLimit(10, 50)(read)

	small := httptest.NewRecorder()
	h.ServeHTTP(small, httptest.NewRequest(http.MethodPost, "/", strings.NewReader("12345")))
	if small.Code != http.StatusNoContent {
		t.Errorf("small body: %d", small.Code)
	}
	big := httptest.NewRecorder()
	h.ServeHTTP(big, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("x", 100))))
	if big.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("oversized JSON body: %d", big.Code)
	}
	upload := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("x", 40)))
	upload.Header.Set("Content-Type", "multipart/form-data; boundary=x")
	up := httptest.NewRecorder()
	h.ServeHTTP(up, upload)
	if up.Code != http.StatusNoContent {
		t.Errorf("a multipart body within the upload cap must pass, got %d", up.Code)
	}
	// The Content-Type is client-chosen: declaring multipart must not lift the cap.
	huge := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("x", 100)))
	huge.Header.Set("Content-Type", "Multipart/Form-Data; boundary=x")
	hugeRec := httptest.NewRecorder()
	h.ServeHTTP(hugeRec, huge)
	if hugeRec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("an oversized body declared as multipart must still be refused, got %d", hugeRec.Code)
	}
}

func TestOpenEndpointsAnswerAnyOriginWhileTheRestKeepsTheOriginCheck(t *testing.T) {
	h := OpenEndpoints("/mcp")(OriginCheck([]string{"https://ledger.example"})(ok))
	do := func(method, path string, headers map[string]string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "https://ledger.example"+path, nil)
		req.Header.Set("Origin", "https://claude.ai")
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	if rec := do("POST", "/mcp", nil); rec.Code != http.StatusNoContent || rec.Header().Get("Access-Control-Allow-Origin") != "*" ||
		!strings.Contains(rec.Header().Get("Access-Control-Expose-Headers"), "WWW-Authenticate") {
		t.Errorf("a foreign origin reaches an open endpoint, which must expose its challenge: %d %v", rec.Code, rec.Header())
	}
	pre := do("OPTIONS", "/mcp", map[string]string{"Access-Control-Request-Method": "POST", "Access-Control-Request-Headers": "authorization,content-type"})
	if pre.Code != http.StatusNoContent || !strings.Contains(pre.Header().Get("Access-Control-Allow-Headers"), "Authorization") ||
		!strings.Contains(pre.Header().Get("Access-Control-Allow-Methods"), "POST") {
		t.Errorf("preflight: %d %v", pre.Code, pre.Header())
	}
	if rec := do("POST", "/api/auth/login", nil); rec.Code != http.StatusForbidden || rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("any other route keeps the origin check and gets no CORS grant: %d %v", rec.Code, rec.Header())
	}
	if rec := do("POST", "/mcp/other", nil); rec.Code != http.StatusForbidden {
		t.Errorf("only the exact paths are open: %d", rec.Code)
	}
}
