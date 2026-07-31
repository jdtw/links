package links

import (
	"context"
	"os"
	"testing"

	pb "jdtw.dev/links/proto/links"
)

// newTestPostgresStore connects to the database at DATABASE_URL, skipping
// the test if it isn't set. Run via ./docker_test.sh or CI, both of which
// export it against a real Postgres instance with the links.sql schema
// already applied.
func newTestPostgresStore(t *testing.T) *PostgresStore {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set; skipping PostgresStore tests")
	}
	s, err := NewPostgresStore(context.Background(), dbURL)
	if err != nil {
		t.Fatalf("NewPostgresStore failed: %v", err)
	}
	t.Cleanup(s.Close)
	return s
}

func TestPostgresPutReportsCreatedVsUpdated(t *testing.T) {
	s := newTestPostgresStore(t)
	ctx := context.Background()
	const key = "pgv5migrationtest_createdvsupdated"
	t.Cleanup(func() { s.Delete(ctx, key) })

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
}

func TestPostgresGetMissingKeyReturnsNil(t *testing.T) {
	s := newTestPostgresStore(t)
	ctx := context.Background()

	le, err := s.Get(ctx, "pgv5migrationtest_doesnotexist")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if le != nil {
		t.Errorf("Get(missing) = %v, want nil", le)
	}
}

func TestPostgresDelete(t *testing.T) {
	s := newTestPostgresStore(t)
	ctx := context.Background()
	const key = "pgv5migrationtest_delete"

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

func TestPostgresVisit(t *testing.T) {
	s := newTestPostgresStore(t)
	ctx := context.Background()
	const key = "pgv5migrationtest_visit"
	const uri = "http://example.com/visit"
	t.Cleanup(func() { s.Delete(ctx, key) })

	if _, err := s.Put(ctx, key, &pb.Link{Uri: uri}); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	found := false
	err := s.Visit(ctx, func(k string, le *pb.LinkEntry) {
		if k == key {
			found = true
			if le.Link.GetUri() != uri {
				t.Errorf("Visit(%s) URI = %q, want %q", key, le.Link.GetUri(), uri)
			}
		}
	})
	if err != nil {
		t.Fatalf("Visit failed: %v", err)
	}
	if !found {
		t.Errorf("Visit did not observe key %q", key)
	}
}
