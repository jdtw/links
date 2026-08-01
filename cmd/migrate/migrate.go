// migrate copies every link from the Postgres store named by DATABASE_URL
// into a SQLite database file, then verifies the copy. It is idempotent:
// rerunning it overwrites rows that already exist rather than duplicating
// them, and it never modifies the source database.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"jdtw.dev/links/pkg/links"
	pb "jdtw.dev/links/proto/links"
)

func main() {
	sqlitePath := flag.String("sqlite", os.Getenv("SQLITE_PATH"), "Destination SQLite database file (defaults to $SQLITE_PATH)")
	flag.Parse()
	log.SetPrefix("migrate: ")

	if *sqlitePath == "" {
		log.Fatal("no destination: pass -sqlite or set SQLITE_PATH")
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL must be set to the source Postgres database")
	}

	ctx := context.Background()

	src, err := links.NewPostgresStore(ctx, dbURL)
	if err != nil {
		log.Fatalf("connecting to source Postgres failed: %v", err)
	}
	defer src.Close()
	log.Print("connected to source Postgres")

	dst, err := links.NewSQLiteStore(ctx, *sqlitePath)
	if err != nil {
		log.Fatalf("opening destination SQLite database failed: %v", err)
	}
	defer dst.Close()
	log.Printf("opened destination %s", *sqlitePath)

	n, err := links.ImportFrom(ctx, dst, src)
	if err != nil {
		log.Fatalf("import failed: %v", err)
	}
	log.Printf("copied %d entries", n)

	if err := verify(ctx, src, dst); err != nil {
		log.Fatalf("verification failed: %v", err)
	}
	log.Printf("verified %d entries match the source", n)
}

// verify re-reads every source entry and confirms the destination holds an
// identical URI and RequiredPaths for the same key.
func verify(ctx context.Context, src, dst links.Store) error {
	var checked int
	var mismatches []string

	if err := src.Visit(ctx, func(k string, want *pb.LinkEntry) {
		checked++
		got, err := dst.Get(ctx, k)
		if err != nil {
			mismatches = append(mismatches, k+": read error: "+err.Error())
			return
		}
		if got == nil {
			mismatches = append(mismatches, k+": missing from destination")
			return
		}
		if got.Link.GetUri() != want.Link.GetUri() {
			mismatches = append(mismatches, k+": URI mismatch")
			return
		}
		if got.RequiredPaths != want.RequiredPaths {
			mismatches = append(mismatches, k+": RequiredPaths mismatch")
		}
	}); err != nil {
		return err
	}

	for _, m := range mismatches {
		log.Printf("MISMATCH %s", m)
	}
	if len(mismatches) > 0 {
		return fmt.Errorf("%d of %d entries did not match", len(mismatches), checked)
	}
	return nil
}
