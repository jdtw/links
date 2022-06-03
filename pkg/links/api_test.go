package links

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"jdtw.dev/links/pkg/tokentest"
	pb "jdtw.dev/links/proto/links"
	"jdtw.dev/token"
)

func TestPutRejectsInvalidRequests(t *testing.T) {
	tests := []io.Reader{
		nil,
		strings.NewReader("not-a-proto"),
		marshalLink(t, ""),
		marshalLink(t, "http://embedded\x00null"),
		marshalLink(t, "no-scheme"),
	}
	keyset, priv := tokentest.GenerateKey(t, "test")
	srv := NewHandler(NewMemStore(), keyset)

	for _, tc := range tests {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/api/links/foo", tc)
		signRequest(t, priv, req)
		srv.ServeHTTP(rr, req)
		if sc := rr.Result().StatusCode; sc != http.StatusBadRequest {
			t.Errorf("PUT %v returned %d, want 400", tc, sc)
		}
	}
}

func TestNilKeysetFailsClosed(t *testing.T) {
	routes := []struct {
		method string
		path   string
	}{{"GET", "/api/links"},
		{"PUT", "/api/links/foo"},
		{"GET", "/api/links/foo"},
		{"DELETE", "/api/links/foo"}}

	_, priv := tokentest.GenerateKey(t, "test")
	srv := NewHandler(NewMemStore(), nil)
	for _, r := range routes {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(r.method, r.path, nil)
		signRequest(t, priv, req)
		srv.ServeHTTP(rr, req)
		if code := rr.Result().StatusCode; code != http.StatusUnauthorized {
			t.Errorf("%v got code %d, want %d", r, code, http.StatusUnauthorized)
		}
	}
}

func TestUnsignedRequestFails(t *testing.T) {
	routes := []struct {
		method string
		path   string
	}{{"GET", "/api/links"},
		{"PUT", "/api/links/foo"},
		{"GET", "/api/links/foo"},
		{"DELETE", "/api/links/foo"}}

	keyset, _ := tokentest.GenerateKey(t, "test")
	srv := NewHandler(NewMemStore(), keyset)
	for _, r := range routes {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(r.method, r.path, nil)
		srv.ServeHTTP(rr, req)
		if code := rr.Result().StatusCode; code != http.StatusUnauthorized {
			t.Errorf("%v got code %d, want %d", r, code, http.StatusUnauthorized)
		}
	}
}

func TestUntrustedKeyFails(t *testing.T) {
	routes := []struct {
		method string
		path   string
	}{{"GET", "/api/links"},
		{"PUT", "/api/links/foo"},
		{"GET", "/api/links/foo"},
		{"DELETE", "/api/links/foo"}}

	keyset, _ := tokentest.GenerateKey(t, "test")
	_, priv := tokentest.GenerateKey(t, "evil")
	srv := NewHandler(NewMemStore(), keyset)
	for _, r := range routes {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(r.method, r.path, nil)
		signRequest(t, priv, req)
		srv.ServeHTTP(rr, req)
		if code := rr.Result().StatusCode; code != http.StatusUnauthorized {
			t.Errorf("%v got code %d, want %d", r, code, http.StatusUnauthorized)
		}
	}
}

func TestCRUD(t *testing.T) {
	keyset, priv := tokentest.GenerateKey(t, "test")
	srv := NewHandler(NewMemStore(), keyset)
	serveHTTP := func(method, path string, body io.Reader) *http.Response {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, body)
		signRequest(t, priv, req)
		srv.ServeHTTP(rr, req)
		return rr.Result()
	}

	// 1 ) Return an empty list in an empty DB
	func() {
		res := serveHTTP("GET", "/api/links", nil)
		if sc := res.StatusCode; sc != http.StatusOK {
			t.Errorf("GET /api/links failed: %d", sc)
			return
		}
		ls := new(pb.Links)
		unmarshal(t, res.Body, ls)
		if ls.Links != nil {
			t.Errorf("GET /api/links returned %v; want nil", ls.Links)
		}
	}()

	// 1.5 ) Return not found for a missing link
	func() {
		res := serveHTTP("GET", "/api/links/foo", nil)
		if sc := res.StatusCode; sc != http.StatusNotFound {
			t.Errorf("GET /api/links/foo returned %d; want %d", sc, http.StatusNotFound)
		}
	}()

	// 2 ) Add an item.
	var (
		path = "/api/links/foo"
		uri  = "http://example.com"
	)
	func() {
		body := marshalLink(t, uri)
		res := serveHTTP("PUT", path, body)
		if sc := res.StatusCode; sc != http.StatusCreated {
			t.Errorf("PUT %s returned %d, want 201", path, sc)
			return
		}
	}()

	// 3 ) Read the item back
	func() {
		res := serveHTTP("GET", path, nil)
		if sc := res.StatusCode; sc != http.StatusOK {
			t.Errorf("GET %s returned %d, want 200", path, sc)
			return
		}
		l := new(pb.Link)
		unmarshal(t, res.Body, l)
		if l.Uri != uri {
			t.Errorf("GET %s returned %v, want %q", path, l, uri)
		}
	}()

	// 4 ) List all items
	func() {
		res := serveHTTP("GET", "/api/links", nil)
		if sc := res.StatusCode; sc != http.StatusOK {
			t.Errorf("GET %s returned %d, want 200", path, sc)
			return
		}
		l := new(pb.Links)
		unmarshal(t, res.Body, l)
		if len(l.Links) != 1 || l.Links["foo"].GetUri() != uri {
			t.Errorf(`GET /api/links returned %v, want {"foo":%q}`, l, uri)
		}
	}()

	// 5 ) Update the item
	uri = "https://example.com/better/uri"
	func() {
		body := marshalLink(t, uri)
		res := serveHTTP("PUT", path, body)
		if sc := res.StatusCode; sc != http.StatusNoContent {
			t.Errorf("PUT %s returned %d, want 204", path, sc)
			return
		}
	}()

	// 6 ) Read it back again
	func() {
		res := serveHTTP("GET", path, nil)
		if sc := res.StatusCode; sc != http.StatusOK {
			t.Errorf("GET %s returned %d, want 200", path, sc)
			return
		}
		l := new(pb.Link)
		unmarshal(t, res.Body, l)
		if l.Uri != uri {
			t.Errorf("GET %s returned %v, want %q", path, l, uri)
		}
	}()

	// 7 ) Delete it
	func() {
		res := serveHTTP("DELETE", path, nil)
		if sc := res.StatusCode; sc != http.StatusNoContent {
			t.Errorf("DELETE %s returned %d, want 204", path, sc)
			return
		}
	}()

	// 8 ) Read it back again
	func() {
		res := serveHTTP("GET", path, nil)
		if sc := res.StatusCode; sc != http.StatusNotFound {
			t.Errorf("GET %s returned %d, want 404", path, sc)
		}
	}()
}

func marshalLink(t *testing.T, uri string) io.Reader {
	t.Helper()
	l := &pb.Link{
		Uri: uri,
	}
	return marshal(t, l)
}

func marshal(t *testing.T, m proto.Message) io.Reader {
	t.Helper()
	b, err := protojson.Marshal(m)
	if err != nil {
		t.Fatalf("protojson.Marshal(%v) failed: %v", m, err)
	}
	return bytes.NewReader(b)
}

func unmarshal(t *testing.T, r io.Reader, m proto.Message) {
	t.Helper()
	b, err := ioutil.ReadAll(r)
	if err != nil {
		t.Fatalf("ioutil.ReadAll failed: %v", err)
	}
	if err := protojson.Unmarshal(b, m); err != nil {
		t.Fatalf("protojson.Unmarshal failed: %v", err)
	}
}

// signRequest manually signs a request. We cannot use the AuthorizeRequest
// method provided by the token module because this is a httptest request,
// which looks like a server request, not a client one.
func signRequest(t *testing.T, s *token.SigningKey, r *http.Request) {
	t.Helper()
	opts := &token.SignOptions{
		Resource: fmt.Sprintf("%s %s%s", r.Method, r.Host, r.URL),
		Lifetime: time.Second * 10,
	}
	signed, _, err := s.Sign(opts)
	if err != nil {
		t.Fatal(err)
	}
	encoded := base64.URLEncoding.EncodeToString(signed)
	r.Header.Set("Authorization", token.Scheme+encoded)
}
