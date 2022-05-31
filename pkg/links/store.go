package links

import pb "jdtw.dev/links/proto/links"

type Store interface {
	Get(k string) (*pb.LinkEntry, error)
	Put(k string, l *pb.Link) (bool, error)
	Delete(k string)
	Visit(visit func(string, *pb.LinkEntry))
}
