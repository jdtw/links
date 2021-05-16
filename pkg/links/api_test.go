package links

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	pb "github.com/jdtw/links/proto/links"
	"google.golang.org/protobuf/proto"
)

func TestCRUD(t *testing.T) {
	kv := NewMemKV()
	srv := NewHandler(kv, nil)
	serveHTTP := func(method, path string, body io.Reader) *http.Response {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, body)
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

	// 9 ) Batch add/read
	links := &pb.Links{
		Links: map[string]*pb.Link{
			"foo": {Uri: "https://example.com"},
			"bar": {Uri: "https://example.com/baz"},
		},
	}
	func() {
		res := serveHTTP("POST", "/api/links", marshal(t, links))
		if sc := res.StatusCode; sc != http.StatusNoContent {
			t.Errorf("POST /api/links returned %d, want 204", sc)
			return
		}
		for k, want := range links.Links {
			res = serveHTTP("GET", "/api/links/"+k, nil)
			if sc := res.StatusCode; sc != http.StatusOK {
				t.Errorf("GET %s returned %d, want 200", path, sc)
				return
			}
			got := new(pb.Link)
			unmarshal(t, res.Body, got)
			if got.Uri != want.Uri {
				t.Errorf("GET /api/links/%s returned %v, want %q", k, got, want)
			}
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
	b, err := proto.Marshal(m)
	if err != nil {
		t.Fatalf("proto.Marshal(%v) failed: %v", m, err)
	}
	return bytes.NewReader(b)
}

func unmarshal(t *testing.T, r io.Reader, m proto.Message) {
	t.Helper()
	b, err := ioutil.ReadAll(r)
	if err != nil {
		t.Fatalf("ioutil.ReadAll failed: %v", err)
	}
	if err := proto.Unmarshal(b, m); err != nil {
		t.Fatalf("proto.Unmarshal failed: %v", err)
	}
}
