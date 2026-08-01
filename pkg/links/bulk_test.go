package links

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"jdtw.dev/links/pkg/tokentest"
	pb "jdtw.dev/links/proto/links"
	"jdtw.dev/token"
)

// postLinks bulk-imports the given path -> URI pairs and returns the response.
func postLinks(t *testing.T, srv http.Handler, priv *token.SigningKey, links map[string]string) *http.Response {
	t.Helper()
	lpb := &pb.Links{Links: make(map[string]*pb.Link, len(links))}
	for k, uri := range links {
		lpb.Links[k] = &pb.Link{Uri: uri}
	}
	return postBody(t, srv, priv, marshal(t, lpb))
}

func postBody(t *testing.T, srv http.Handler, priv *token.SigningKey, body io.Reader) *http.Response {
	t.Helper()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/links", body)
	signRequest(t, priv, req)
	srv.ServeHTTP(rr, req)
	return rr.Result()
}

func TestBulkPutImportsLinks(t *testing.T) {
	keyset, priv := tokentest.GenerateKey(t, "test")
	store := NewMemStore()
	srv := NewHandler(store, keyset, 0)

	want := map[string]string{
		"rfc":   "https://datatracker.ietf.org/doc/html/rfc{0}",
		"gh":    "https://github.com/jdtw/{0}",
		"plain": "https://example.com/plain",
		Index:   "https://example.com",
	}
	if sc := postLinks(t, srv, priv, want).StatusCode; sc != http.StatusNoContent {
		t.Fatalf("POST returned %d, want 204", sc)
	}

	for k, wantURI := range want {
		le, err := store.Get(context.Background(), k)
		if err != nil {
			t.Fatalf("Get(%s) failed: %v", k, err)
		}
		if le == nil {
			t.Errorf("Get(%s) = nil, want the imported link", k)
			continue
		}
		if le.Link.GetUri() != wantURI {
			t.Errorf("Get(%s) = %q, want %q", k, le.Link.GetUri(), wantURI)
		}
	}
}

// A bulk import must compute RequiredPaths the same way a single PUT does,
// or {n} substitution silently breaks on imported links.
func TestBulkPutComputesRequiredPaths(t *testing.T) {
	keyset, priv := tokentest.GenerateKey(t, "test")
	store := NewMemStore()
	srv := NewHandler(store, keyset, 0)

	const uri = "https://example.com/{1}/{0}"
	if sc := postLinks(t, srv, priv, map[string]string{"swap": uri}).StatusCode; sc != http.StatusNoContent {
		t.Fatalf("POST returned %d, want 204", sc)
	}

	le, err := store.Get(context.Background(), "swap")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got, want := le.RequiredPaths, requiredPaths(&pb.Link{Uri: uri}); got != want {
		t.Errorf("RequiredPaths = %d, want %d", got, want)
	}
	if le.RequiredPaths != 2 {
		t.Errorf("RequiredPaths = %d, want 2", le.RequiredPaths)
	}
}

func TestBulkPutIsAdditive(t *testing.T) {
	keyset, priv := tokentest.GenerateKey(t, "test")
	store := NewMemStore()
	srv := NewHandler(store, keyset, 0)
	ctx := context.Background()

	if _, err := store.Put(ctx, "existing", &pb.Link{Uri: "https://example.com/existing"}); err != nil {
		t.Fatalf("seeding failed: %v", err)
	}

	if sc := postLinks(t, srv, priv, map[string]string{"new": "https://example.com/new"}).StatusCode; sc != http.StatusNoContent {
		t.Fatalf("POST returned %d, want 204", sc)
	}

	// The link the import did not mention must survive.
	le, err := store.Get(ctx, "existing")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if le == nil {
		t.Error("import removed a link it did not mention; want additive behavior")
	}
}

func TestBulkPutOverwritesExisting(t *testing.T) {
	keyset, priv := tokentest.GenerateKey(t, "test")
	store := NewMemStore()
	srv := NewHandler(store, keyset, 0)
	ctx := context.Background()

	if _, err := store.Put(ctx, "dup", &pb.Link{Uri: "https://example.com/old"}); err != nil {
		t.Fatalf("seeding failed: %v", err)
	}
	if sc := postLinks(t, srv, priv, map[string]string{"dup": "https://example.com/new"}).StatusCode; sc != http.StatusNoContent {
		t.Fatalf("POST returned %d, want 204", sc)
	}

	le, err := store.Get(ctx, "dup")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got, want := le.Link.GetUri(), "https://example.com/new"; got != want {
		t.Errorf("URI = %q, want %q", got, want)
	}
}

func TestBulkPutRejectsInvalidRequests(t *testing.T) {
	tests := []struct {
		name string
		body io.Reader
	}{
		{"nil body", nil},
		{"not a proto", strings.NewReader("not-a-proto")},
		{"empty links", bytes.NewReader([]byte(`{"links":{}}`))},
		{"missing uri", bytes.NewReader([]byte(`{"links":{"foo":{"uri":""}}}`))},
		{"no scheme", bytes.NewReader([]byte(`{"links":{"foo":{"uri":"no-scheme"}}}`))},
		{"reserved qr key", bytes.NewReader([]byte(`{"links":{"qr":{"uri":"https://example.com"}}}`))},
		{"normalization collision", bytes.NewReader([]byte(`{"links":{"my-link":{"uri":"https://example.com/a"},"mylink":{"uri":"https://example.com/b"}}}`))},
	}

	keyset, priv := tokentest.GenerateKey(t, "test")
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := NewHandler(NewMemStore(), keyset, 0)
			if sc := postBody(t, srv, priv, tc.body).StatusCode; sc != http.StatusBadRequest {
				t.Errorf("POST returned %d, want 400", sc)
			}
		})
	}
}

// One bad link must not leave a partially applied import behind.
func TestBulkPutIsAtomicOnValidationFailure(t *testing.T) {
	keyset, priv := tokentest.GenerateKey(t, "test")
	store := NewMemStore()
	srv := NewHandler(store, keyset, 0)

	body := bytes.NewReader([]byte(`{"links":{
		"good":{"uri":"https://example.com/good"},
		"bad":{"uri":"no-scheme"}
	}}`))
	if sc := postBody(t, srv, priv, body).StatusCode; sc != http.StatusBadRequest {
		t.Fatalf("POST returned %d, want 400", sc)
	}

	le, err := store.Get(context.Background(), "good")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if le != nil {
		t.Error("a rejected import wrote the valid link anyway; want nothing written")
	}
}

func TestBulkPutRequiresAuth(t *testing.T) {
	_, priv := tokentest.GenerateKey(t, "test")
	srv := NewHandler(NewMemStore(), nil, 0)
	if sc := postLinks(t, srv, priv, map[string]string{"foo": "https://example.com"}).StatusCode; sc != http.StatusUnauthorized {
		t.Errorf("POST with nil keyset returned %d, want 401", sc)
	}
}

// Export then import must reproduce the original set exactly -- this is the
// property the Postgres -> SQLite migration relies on.
func TestExportImportRoundTrip(t *testing.T) {
	keyset, priv := tokentest.GenerateKey(t, "test")
	ctx := context.Background()

	source := NewMemStore()
	want := map[string]string{
		"rfc":   "https://datatracker.ietf.org/doc/html/rfc{0}",
		"swap":  "https://example.com/{1}/{0}",
		"plain": "https://example.com/plain",
		Index:   "https://example.com",
	}
	for k, uri := range want {
		if _, err := source.Put(ctx, k, &pb.Link{Uri: uri}); err != nil {
			t.Fatalf("seeding %s failed: %v", k, err)
		}
	}

	// Export from the source server.
	exportSrv := NewHandler(source, keyset, 0)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/links", nil)
	signRequest(t, priv, req)
	exportSrv.ServeHTTP(rr, req)
	if sc := rr.Result().StatusCode; sc != http.StatusOK {
		t.Fatalf("GET /api/links returned %d, want 200", sc)
	}
	exported, err := io.ReadAll(rr.Result().Body)
	if err != nil {
		t.Fatalf("reading export failed: %v", err)
	}

	// Import into a fresh, empty store -- the SQLite side of the migration.
	dest := NewMemStore()
	destSrv := NewHandler(dest, keyset, 0)
	if sc := postBody(t, destSrv, priv, bytes.NewReader(exported)).StatusCode; sc != http.StatusNoContent {
		t.Fatalf("POST returned %d, want 204", sc)
	}

	got := map[string]string{}
	if err := dest.Visit(ctx, func(k string, le *pb.LinkEntry) {
		got[k] = le.Link.GetUri()
	}); err != nil {
		t.Fatalf("Visit failed: %v", err)
	}

	if len(got) != len(want) {
		t.Errorf("round trip produced %d links, want %d", len(got), len(want))
	}
	for k, wantURI := range want {
		if got[k] != wantURI {
			t.Errorf("round trip [%s] = %q, want %q", k, got[k], wantURI)
		}
		srcLE, err := source.Get(ctx, k)
		if err != nil {
			t.Fatalf("source.Get(%s) failed: %v", k, err)
		}
		dstLE, err := dest.Get(ctx, k)
		if err != nil {
			t.Fatalf("dest.Get(%s) failed: %v", k, err)
		}
		if srcLE.RequiredPaths != dstLE.RequiredPaths {
			t.Errorf("round trip [%s] RequiredPaths = %d, want %d", k, dstLE.RequiredPaths, srcLE.RequiredPaths)
		}
	}
}
