package links

import (
	"net/http"
	"net/http/httptest"
	"testing"

	pb "github.com/jdtw/links/proto/links"
	"google.golang.org/protobuf/proto"
)

func TestRedirect(t *testing.T) {
	tests := []struct {
		key      string
		value    string
		rawValue []byte
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
		key:      "invalid",
		rawValue: []byte("garbage"),
		get:      "/invalid",
		wantCode: http.StatusInternalServerError,
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

		val := tc.rawValue
		if val == nil {
			le := linkEntry(tc.value)
			leBytes, err := proto.Marshal(le)
			if err != nil {
				t.Errorf("proto.Marshal(%+v) failed: %v", le, err)
				continue
			}
			val = leBytes
		}
		kv := NewMemKV()
		kv.Put(LinkKey(tc.key), val)

		req, err := http.NewRequest("GET", tc.get, nil)
		if err != nil {
			t.Errorf("NewRequest(%v) failed: %v", tc.get, err)
			continue
		}

		rr := httptest.NewRecorder()
		srv := NewHandler(kv, nil)
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

func linkEntry(uri string) *pb.LinkEntry {
	l := &pb.Link{
		Uri: uri,
	}
	return &pb.LinkEntry{
		Link:          l,
		RequiredPaths: requiredPaths(l),
	}
}
