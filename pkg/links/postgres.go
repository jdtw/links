package links

import (
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/jackc/pgx/v4/stdlib"
	pb "jdtw.dev/links/proto/links"
)

const (
	get = "select link, segments from links where path=$1"
	put = `insert into links (path, link, segments) values ($1, $2, $3)
         on conflict (path) do update set link=excluded.link, segments=excluded.segments`
	del  = "delete from links where path=$1"
	list = "select * from links"
)

type PostgresStore struct {
	db *sql.DB
}

var _ Store = &PostgresStore{}

func (s *PostgresStore) Close() {
	if s.db != nil {
		s.db.Close()
	}
}

func NewPostgresStore(source string) (*PostgresStore, error) {
	s := &PostgresStore{}
	var err error
	s.db, err = sql.Open("pgx", source)
	if err != nil {
		return nil, fmt.Errorf("sql.Open failed: %w", err)
	}

	if err := s.db.Ping(); err != nil {
		return nil, fmt.Errorf("db.Ping failed: %v", err)
	}
	return s, nil
}

func (s *PostgresStore) Get(key string) (*pb.LinkEntry, error) {
	var link string
	var segments int
	if err := s.db.QueryRow(get, key).Scan(&link, &segments); err != nil {
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

func (s *PostgresStore) Put(key string, l *pb.Link) (bool, error) {
	_, err := s.db.Exec(put, key, l.Uri, requiredPaths(l))
	// Always returns true, since there's no easy way to differentiate
	// created (true) vs updated.
	return true, err
}

func (s *PostgresStore) Delete(key string) error {
	_, err := s.db.Exec(del, key)
	return err
}

func (s *PostgresStore) Visit(visit func(string, *pb.LinkEntry)) error {
	rows, err := s.db.Query(list)
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
		le := &pb.LinkEntry{
			Link:          &pb.Link{Uri: link},
			RequiredPaths: int32(segments),
		}
		visit(path, le)
	}
	return rows.Err()
}
