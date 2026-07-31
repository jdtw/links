package frontend

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"jdtw.dev/links/pkg/client"
	"jdtw.dev/links/pkg/links"
	"jdtw.dev/links/pkg/tokentest"
)

// newTestServer wires a real links backend to a frontend handler, the same
// way cmd/client does, so CSRF checks can be exercised end to end.
func newTestServer(t *testing.T) http.Handler {
	t.Helper()
	keyset, priv := tokentest.GenerateKey(t, "test")
	backend := httptest.NewServer(links.NewHandler(links.NewMemStore(), keyset, 0))
	t.Cleanup(backend.Close)
	return NewHandler(client.New(backend.URL, priv))
}

func postForm(link, uri string) (*http.Request, *httptest.ResponseRecorder) {
	form := url.Values{"link": {link}, "uri": {uri}}
	req := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req, httptest.NewRecorder()
}

func TestAddLinkRejectsCrossOrigin(t *testing.T) {
	srv := newTestServer(t)

	tests := []struct {
		name    string
		origin  string
		referer string
	}{
		{name: "missing Origin and Referer"},
		{name: "mismatched Origin", origin: "http://evil.example"},
		{name: "mismatched Referer", referer: "http://evil.example/attack"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, rr := postForm("foo", "http://example.com")
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			if tc.referer != "" {
				req.Header.Set("Referer", tc.referer)
			}
			srv.ServeHTTP(rr, req)
			if sc := rr.Result().StatusCode; sc != http.StatusForbidden {
				t.Errorf("got status %d, want %d", sc, http.StatusForbidden)
			}
		})
	}
}

func TestAddLinkAllowsSameOrigin(t *testing.T) {
	srv := newTestServer(t)

	req, rr := postForm("foo", "http://example.com")
	req.Header.Set("Origin", "http://example.com")
	srv.ServeHTTP(rr, req)
	if sc := rr.Result().StatusCode; sc != http.StatusOK {
		t.Fatalf("Origin match: got status %d, want %d", sc, http.StatusOK)
	}

	// Referer fallback works too, when Origin is absent.
	req, rr = postForm("bar", "http://example.com")
	req.Header.Set("Referer", "http://example.com/")
	srv.ServeHTTP(rr, req)
	if sc := rr.Result().StatusCode; sc != http.StatusOK {
		t.Fatalf("Referer fallback: got status %d, want %d", sc, http.StatusOK)
	}
}

func TestGetIndexIsUnaffectedByCSRFCheck(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if sc := rr.Result().StatusCode; sc != http.StatusOK {
		t.Fatalf("got status %d, want %d", sc, http.StatusOK)
	}
}

func TestRemoveLinkRejectsCrossOrigin(t *testing.T) {
	srv := newTestServer(t)

	// Add the link with a valid same-origin request first.
	req, rr := postForm("foo", "http://example.com")
	req.Header.Set("Origin", "http://example.com")
	srv.ServeHTTP(rr, req)
	if sc := rr.Result().StatusCode; sc != http.StatusOK {
		t.Fatalf("setup PUT: got status %d, want %d", sc, http.StatusOK)
	}

	req = httptest.NewRequest("DELETE", "/rm/foo", nil)
	req.Header.Set("Origin", "http://evil.example")
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if sc := rr.Result().StatusCode; sc != http.StatusForbidden {
		t.Fatalf("got status %d, want %d", sc, http.StatusForbidden)
	}
}

func TestRemoveLinkAllowsSameOrigin(t *testing.T) {
	srv := newTestServer(t)

	req, rr := postForm("foo", "http://example.com")
	req.Header.Set("Origin", "http://example.com")
	srv.ServeHTTP(rr, req)
	if sc := rr.Result().StatusCode; sc != http.StatusOK {
		t.Fatalf("setup PUT: got status %d, want %d", sc, http.StatusOK)
	}

	req = httptest.NewRequest("DELETE", "/rm/foo", nil)
	req.Header.Set("Origin", "http://example.com")
	rr = httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if sc := rr.Result().StatusCode; sc != http.StatusOK {
		t.Fatalf("got status %d, want %d", sc, http.StatusOK)
	}
}
