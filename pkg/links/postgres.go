package links

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
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
	db *pgxpool.Pool
}

var _ Store = &PostgresStore{}

func (s *PostgresStore) Close() {
	if s.db != nil {
		s.db.Close()
	}
}

func NewPostgresStore(ctx context.Context, source string) (*PostgresStore, error) {
	cfg, err := pgxpool.ParseConfig(source)
	if err != nil {
		return nil, err
	}
	s := &PostgresStore{}
	s.db, err = pgxpool.ConnectConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("pgxpool.ConnectConfig failed: %w", err)
	}

	if err := s.db.Ping(ctx); err != nil {
		return nil, fmt.Errorf("db.Ping failed: %v", err)
	}
	return s, nil
}

func (s *PostgresStore) Get(ctx context.Context, key string) (*pb.LinkEntry, error) {
	var link string
	var segments int
	if err := s.db.QueryRow(ctx, get, key).Scan(&link, &segments); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &pb.LinkEntry{
		Link:          &pb.Link{Uri: link},
		RequiredPaths: int32(segments),
	}, nil
}

func (s *PostgresStore) Put(ctx context.Context, key string, l *pb.Link) (bool, error) {
	_, err := s.db.Exec(ctx, put, key, l.Uri, requiredPaths(l))
	// Always returns true, since there's no easy way to differentiate
	// created (true) vs updated.
	return true, err
}

func (s *PostgresStore) Delete(ctx context.Context, key string) error {
	_, err := s.db.Exec(ctx, del, key)
	return err
}

func (s *PostgresStore) Visit(ctx context.Context, visit func(string, *pb.LinkEntry)) error {
	rows, err := s.db.Query(ctx, list)
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
