package links

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	pb "jdtw.dev/links/proto/links"
)

func TestRedirect(t *testing.T) {
	tests := []struct {
		key      string
		value    string
		get      string
		wantCode int
		wantLoc  string
	}{{
		key:      Index,
		value:    "https://example.com",
		get:      "/",
		wantCode: http.StatusFound,
		wantLoc:  "https://example.com",
	}, {
		key:      "foo",
		value:    "https://example.com",
		get:      "/foo",
		wantCode: http.StatusFound,
		wantLoc:  "https://example.com",
	}, {
		get:      "/notfound",
		wantCode: http.StatusNotFound,
	}, {
		key:      "replacements",
		value:    "https://example.com/{1}/{0}/quack",
		get:      "/replacements/bar/baz/qux",
		wantCode: http.StatusFound,
		wantLoc:  "https://example.com/baz/bar/quack/qux",
	}, {
		key:      "badreq",
		value:    "https://example.com/{0}/quack",
		get:      "/badreq",
		wantCode: http.StatusBadRequest,
	}, {
		key:      "query",
		value:    "https://example.com",
		get:      "/query/path?e=mc2",
		wantCode: http.StatusFound,
		wantLoc:  "https://example.com/path?e=mc2",
	}, {
		key:      "querytempl",
		value:    "https://example.com?q=foo",
		get:      "/querytempl",
		wantCode: http.StatusFound,
		wantLoc:  "https://example.com?q=foo",
	}, {
		key:      "override",
		value:    "https://example.com?q=default",
		get:      "/override?q=override",
		wantCode: http.StatusFound,
		wantLoc:  "https://example.com?q=override",
	}, {
		key:      "force",
		value:    "https://example.com",
		get:      "/force?",
		wantCode: http.StatusFound,
		wantLoc:  "https://example.com?",
	}, {
		key:      "foobar",
		value:    "https://example.com",
		get:      "/foo-bar",
		wantCode: http.StatusFound,
		wantLoc:  "https://example.com",
	}, {
		key:      "abc",
		value:    "https://example.com/abc",
		get:      "/a-b-c-----",
		wantCode: http.StatusFound,
		wantLoc:  "https://example.com/abc",
	}}

	for _, tc := range tests {
		t.Logf("test %q", tc.key)

		s := NewMemStore()
		s.Put(context.Background(), tc.key, &pb.Link{Uri: tc.value})

		req, err := http.NewRequest("GET", tc.get, nil)
		if err != nil {
			t.Errorf("NewRequest(%v) failed: %v", tc.get, err)
			continue
		}

		rr := httptest.NewRecorder()
		srv := NewHandler(s, nil)
		srv.ServeHTTP(rr, req)
		res := rr.Result()

		if res.StatusCode != tc.wantCode {
			t.Errorf("result %+v: got code %v, want %v", res, res.StatusCode, tc.wantCode)
			continue
		}
		if tc.wantLoc == "" {
			continue
		}
		if loc := res.Header["Location"]; len(loc) != 1 || loc[0] != tc.wantLoc {
			t.Errorf("got location %v, want %q", loc, tc.wantLoc)
		}
	}
}

func TestQR(t *testing.T) {
	tests := []struct {
		key   string
		value string
		get   string
	}{{
		key:   Index,
		value: "https://example.com",
		get:   "/qr",
	}, {
		key:   "foo",
		value: "https://example.com",
		get:   "/qr/foo",
	}}

	for _, tc := range tests {
		t.Logf("test %q", tc.key)

		s := NewMemStore()
		s.Put(context.Background(), tc.key, &pb.Link{Uri: tc.value})

		req, err := http.NewRequest("GET", tc.get, nil)
		if err != nil {
			t.Errorf("NewRequest(%v) failed: %v", tc.get, err)
			continue
		}

		rr := httptest.NewRecorder()
		srv := NewHandler(s, nil)
		srv.ServeHTTP(rr, req)
		res := rr.Result()

		if res.StatusCode != http.StatusOK {
			t.Errorf("result %+v: got code %v, want OK", res, res.StatusCode)
			continue
		}
		if ct := res.Header["Content-Type"]; len(ct) != 1 || ct[0] != "image/png" {
			t.Errorf("got Content-Type %v, want image/png", ct)
		}
	}
}
