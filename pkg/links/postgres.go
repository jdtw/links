package links

import (
	"database/sql"
	"errors"
	"fmt"
	"log"

	_ "github.com/jackc/pgx/v4/stdlib"
	pb "jdtw.dev/links/proto/links"
)

type PostgresStore struct {
	db   *sql.DB
	get  *sql.Stmt
	put  *sql.Stmt
	del  *sql.Stmt
	list *sql.Stmt
}

var _ Store = &PostgresStore{}

func (s *PostgresStore) Close() {
	if s.list != nil {
		s.list.Close()
	}
	if s.get != nil {
		s.get.Close()
	}
	if s.del != nil {
		s.del.Close()
	}
	if s.put != nil {
		s.put.Close()
	}
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

	s.get, err = s.db.Prepare("SELECT link, segments FROM test WHERE path = $1")
	if err != nil {
		s.Close()
		return nil, fmt.Errorf("failed to prepare SELECT WHERE: %w", err)
	}

	s.put, err = s.db.Prepare(`INSERT INTO test (path, link, segments)
		VALUES ($1, $2, $3)
		ON CONFLICT (path)
		DO UPDATE SET link = $2, segments = $3`)
	if err != nil {
		s.Close()
		return nil, fmt.Errorf("failed to prepare INSERT: %w", err)
	}

	s.del, err = s.db.Prepare("DELETE FROM test WHERE path = $1")
	if err != nil {
		s.Close()
		return nil, fmt.Errorf("failed to prepare DELETE: %w", err)
	}

	s.list, err = s.db.Prepare("SELECT * FROM test")
	if err != nil {
		s.Close()
		return nil, fmt.Errorf("failed to prepare SELECT: %w", err)
	}

	return s, nil
}

func (s *PostgresStore) Get(k string) (*pb.LinkEntry, error) {
	var link string
	var segments int
	if err := s.get.QueryRow(k).Scan(&link, &segments); err != nil {
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

func (s *PostgresStore) Put(k string, l *pb.Link) (bool, error) {
	_, err := s.put.Exec(k, l.Uri, requiredPaths(l))
	// TODO: can I differentiate between insert and update?
	// Probably if I use a transaction...
	return false, err
}

func (s *PostgresStore) Delete(k string) error {
	_, err := s.del.Exec(k)
	return err
}

func (s *PostgresStore) Visit(visit func(string, *pb.LinkEntry)) error {
	rows, err := s.list.Query()
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var path string
		var link string
		var segments int
		if err := rows.Scan(&path, &link, &segments); err != nil {
			log.Fatal(err)
		}
		le := &pb.LinkEntry{
			Link:          &pb.Link{Uri: link},
			RequiredPaths: int32(segments),
		}
		visit(path, le)
	}
	return rows.Err()
}
