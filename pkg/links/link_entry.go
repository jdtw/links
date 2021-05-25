package links

import (
	"log"
	"strings"

	pb "github.com/jdtw/links/proto/links"
	"google.golang.org/protobuf/proto"
)

const linkKeyPrefix = "lnk:"

// LinkKey returns a DB key prefixed with "lnk:"
func LinkKey(k string) string { return linkKeyPrefix + k }

func (s *server) getLinkEntry(k string) (*pb.LinkEntry, error) {
	data := s.kv.Get(LinkKey(k))
	if data == nil {
		return nil, nil
	}
	lepb := new(pb.LinkEntry)
	if err := proto.Unmarshal(data, lepb); err != nil {
		return nil, err
	}
	return lepb, nil
}

func (s *server) putLinkEntry(k string, l *pb.Link) (bool, error) {
	le := &pb.LinkEntry{
		Link:          l,
		RequiredPaths: requiredPaths(l),
	}
	data, err := proto.Marshal(le)
	if err != nil {
		return false, err
	}
	return s.kv.Put(LinkKey(k), data)
}

func (s *server) deleteLinkEntry(k string) {
	s.kv.Delete(LinkKey(k))
}

func (s *server) visitLinkEntries(visit func(string, *pb.LinkEntry)) {
	s.kv.Iterate(func(k string, v []byte) {
		if !strings.HasPrefix(k, linkKeyPrefix) {
			return
		}
		k = strings.TrimPrefix(k, linkKeyPrefix)
		lepb := new(pb.LinkEntry)
		if err := proto.Unmarshal(v, lepb); err != nil {
			log.Printf("failed to unmarshal link entry key=%v, v=%v: %v", k, v, err)
			return
		}
		visit(k, lepb)
	})
}
