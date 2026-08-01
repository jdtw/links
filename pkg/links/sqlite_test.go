package links

import (
	"context"
	"path/filepath"
	"testing"

	pb "jdtw.dev/links/proto/links"
)

// newTestSQLiteStore opens a store backed by a file in the test's temp
// directory. These need no external service, so they always run.
func newTestSQLiteStore(t *testing.T) *SQLiteStore {
	t.Helper()
	s, err := NewSQLiteStore(context.Background(), filepath.Join(t.TempDir(), "links.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestSQLitePutReportsCreatedVsUpdated(t *testing.T) {
	s := newTestSQLiteStore(t)
	ctx := context.Background()
	const key = "createdvsupdated"

	created, err := s.Put(ctx, key, &pb.Link{Uri: "http://example.com/first"})
	if err != nil {
		t.Fatalf("Put (insert) failed: %v", err)
	}
	if !created {
		t.Errorf("Put (insert) reported created=false, want true")
	}

	created, err = s.Put(ctx, key, &pb.Link{Uri: "http://example.com/second"})
	if err != nil {
		t.Fatalf("Put (update) failed: %v", err)
	}
	if created {
		t.Errorf("Put (update) reported created=true, want false")
	}

	le, err := s.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got, want := le.Link.GetUri(), "http://example.com/second"; got != want {
		t.Errorf("Get(%s) URI = %q, want %q", key, got, want)
	}
}

func TestSQLiteGetMissingKeyReturnsNil(t *testing.T) {
	s := newTestSQLiteStore(t)

	le, err := s.Get(context.Background(), "doesnotexist")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if le != nil {
		t.Errorf("Get(missing) = %v, want nil", le)
	}
}

func TestSQLiteDelete(t *testing.T) {
	s := newTestSQLiteStore(t)
	ctx := context.Background()
	const key = "delete"

	if _, err := s.Put(ctx, key, &pb.Link{Uri: "http://example.com"}); err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	if err := s.Delete(ctx, key); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	le, err := s.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get after delete failed: %v", err)
	}
	if le != nil {
		t.Errorf("Get after delete = %v, want nil", le)
	}
}

// Deleting a key that was never present should be a no-op, matching the
// in-memory store's behavior.
func TestSQLiteDeleteMissingKeyIsNoOp(t *testing.T) {
	s := newTestSQLiteStore(t)
	if err := s.Delete(context.Background(), "neverexisted"); err != nil {
		t.Errorf("Delete(missing) failed: %v", err)
	}
}

func TestSQLiteVisit(t *testing.T) {
	s := newTestSQLiteStore(t)
	ctx := context.Background()

	want := map[string]string{
		"one":   "http://example.com/one",
		"two":   "http://example.com/two",
		"three": "http://example.com/three",
	}
	for k, uri := range want {
		if _, err := s.Put(ctx, k, &pb.Link{Uri: uri}); err != nil {
			t.Fatalf("Put(%s) failed: %v", k, err)
		}
	}

	got := map[string]string{}
	if err := s.Visit(ctx, func(k string, le *pb.LinkEntry) {
		got[k] = le.Link.GetUri()
	}); err != nil {
		t.Fatalf("Visit failed: %v", err)
	}

	if len(got) != len(want) {
		t.Errorf("Visit saw %d entries, want %d", len(got), len(want))
	}
	for k, wantURI := range want {
		if got[k] != wantURI {
			t.Errorf("Visit(%s) URI = %q, want %q", k, got[k], wantURI)
		}
	}
}

// Put must persist the computed RequiredPaths so that {n} substitution keeps
// working after a restart.
func TestSQLitePutPersistsRequiredPaths(t *testing.T) {
	s := newTestSQLiteStore(t)
	ctx := context.Background()
	const key = "subst"

	l := &pb.Link{Uri: "http://example.com/{1}/{0}"}
	if _, err := s.Put(ctx, key, l); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	le, err := s.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got, want := le.RequiredPaths, requiredPaths(l); got != want {
		t.Errorf("RequiredPaths = %d, want %d", got, want)
	}
	if le.RequiredPaths != 2 {
		t.Errorf("RequiredPaths = %d, want 2 for %q", le.RequiredPaths, l.Uri)
	}
}

// The database file must survive being closed and reopened -- this is the
// whole point of putting it on a volume.
func TestSQLitePersistsAcrossReopen(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "links.db")

	first, err := NewSQLiteStore(ctx, path)
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	if _, err := first.Put(ctx, "persisted", &pb.Link{Uri: "http://example.com/persisted"}); err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	second, err := NewSQLiteStore(ctx, path)
	if err != nil {
		t.Fatalf("reopening store failed: %v", err)
	}
	t.Cleanup(func() { second.Close() })

	le, err := second.Get(ctx, "persisted")
	if err != nil {
		t.Fatalf("Get after reopen failed: %v", err)
	}
	if le == nil {
		t.Fatal("Get after reopen = nil, want the entry written before close")
	}
	if got, want := le.Link.GetUri(), "http://example.com/persisted"; got != want {
		t.Errorf("URI after reopen = %q, want %q", got, want)
	}
}
