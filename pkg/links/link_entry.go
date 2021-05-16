package links

import (
	"log"
	"net/url"
	"strings"

	pb "github.com/jdtw/links/proto/links"
	"google.golang.org/protobuf/proto"
)

const linkKeyPrefix = "lnk:"

// LinkKey returns a DB key prefixed with "lnk:"
func LinkKey(k string) string { return linkKeyPrefix + k }

// LinkEntry creates a DB entry value from a URI. Returns
// an error if the URI cannot be parsed.
func LinkEntry(uri string) ([]byte, error) {
	// Create a dummy API with all template parameters replaced
	// with something innocuous so that we can try to parse it.
	dummy := replacement.ReplaceAllString(uri, "links")
	if _, err := url.Parse(dummy); err != nil {
		return nil, err
	}
	le := linkEntry(uri)
	return proto.Marshal(le)
}

func linkEntry(uri string) *pb.LinkEntry {
	l := &pb.Link{
		Uri: uri,
	}
	return &pb.LinkEntry{
		Link:          l,
		RequiredPaths: requiredPaths(l),
	}
}

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

func (s *server) addLinks(ls *pb.Links) error {
	m := make(map[string][]byte, len(ls.Links))
	for k, l := range ls.Links {
		b, err := proto.Marshal(&pb.LinkEntry{
			Link:          l,
			RequiredPaths: requiredPaths(l),
		})
		if err != nil {
			return err
		}
		m[LinkKey(k)] = b
	}
	s.kv.Add(m)
	return nil
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
