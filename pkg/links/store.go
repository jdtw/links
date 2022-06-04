package links

import (
	"context"

	pb "jdtw.dev/links/proto/links"
)

type Store interface {
	Get(ctx context.Context, k string) (*pb.LinkEntry, error)
	Put(ctx context.Context, k string, l *pb.Link) (bool, error)
	Delete(ctx context.Context, k string) error
	Visit(ctx context.Context, visit func(string, *pb.LinkEntry)) error
}
