package links

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	pb "jdtw.dev/links/proto/links"
	_ "modernc.org/sqlite"
)

const (
	// sqliteSchema is applied on open so that a fresh database file (for
	// example, a newly provisioned volume) is usable without any manual
	// setup.
	sqliteSchema = `create table if not exists links (
  path text primary key,
  link text not null,
  segments integer not null
)`

	sqliteGet    = "select link, segments from links where path=?"
	sqliteExists = "select 1 from links where path=?"
	sqlitePut    = `insert into links (path, link, segments) values (?, ?, ?)
         on conflict (path) do update set link=excluded.link, segments=excluded.segments`
	sqliteDel  = "delete from links where path=?"
	sqliteList = "select path, link, segments from links"
)

// SQLiteStore is a Store backed by a local SQLite database file. The link
// table is small enough that a file on a mounted volume serves it fine, at
// the cost of pinning the app to a single machine in a single region.
type SQLiteStore struct {
	db *sql.DB
}

var _ Store = &SQLiteStore{}

func (s *SQLiteStore) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

// NewSQLiteStore opens (creating if necessary) the SQLite database at path
// and applies the schema. WAL mode keeps redirect reads from blocking on the
// occasional write, and busy_timeout absorbs the brief contention that WAL
// still allows between concurrent writers.
func NewSQLiteStore(ctx context.Context, path string) (*SQLiteStore, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open failed: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("db.Ping failed: %w", err)
	}

	if _, err := db.ExecContext(ctx, sqliteSchema); err != nil {
		db.Close()
		return nil, fmt.Errorf("applying schema failed: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Get(ctx context.Context, key string) (*pb.LinkEntry, error) {
	var link string
	var segments int
	if err := s.db.QueryRowContext(ctx, sqliteGet, key).Scan(&link, &segments); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &pb.LinkEntry{
		Link:          &pb.Link{Uri: link},
		RequiredPaths: int32(segments),
	}, nil
}

// Put upserts the link and reports whether it was created rather than
// updated. SQLite cannot report that from the upsert itself, so the existence
// check and the write share a transaction to keep the answer accurate under
// concurrent writers.
func (s *SQLiteStore) Put(ctx context.Context, key string, l *pb.Link) (bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	var exists int
	err = tx.QueryRowContext(ctx, sqliteExists, key).Scan(&exists)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	created := errors.Is(err, sql.ErrNoRows)

	if _, err := tx.ExecContext(ctx, sqlitePut, key, l.Uri, requiredPaths(l)); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return created, nil
}

func (s *SQLiteStore) Delete(ctx context.Context, key string) error {
	_, err := s.db.ExecContext(ctx, sqliteDel, key)
	return err
}

func (s *SQLiteStore) Visit(ctx context.Context, visit func(string, *pb.LinkEntry)) error {
	rows, err := s.db.QueryContext(ctx, sqliteList)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var path string
		var link string
		var segments int
		if err := rows.Scan(&path, &link, &segments); err != nil {
			return err
		}
		visit(path, &pb.LinkEntry{
			Link:          &pb.Link{Uri: link},
			RequiredPaths: int32(segments),
		})
	}
	return rows.Err()
}
