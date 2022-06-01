package links

import (
	"strings"

	"google.golang.org/protobuf/proto"
	pb "jdtw.dev/links/proto/links"
)

type KVStore struct {
	kv *KV
}

var _ Store = &KVStore{}

func NewKVStore(kv *KV) *KVStore {
	return &KVStore{kv}
}

const linkKeyPrefix = "lnk:"

// LinkKey returns a DB key prefixed with "lnk:"
func LinkKey(k string) string { return linkKeyPrefix + k }

func (s *KVStore) Get(k string) (*pb.LinkEntry, error) {
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

func (s *KVStore) Put(k string, l *pb.Link) (bool, error) {
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

func (s *KVStore) Delete(k string) error {
	s.kv.Delete(LinkKey(k))
	return nil
}

func (s *KVStore) Visit(visit func(string, *pb.LinkEntry)) error {
	s.kv.Iterate(func(k string, v []byte) {
		if !strings.HasPrefix(k, linkKeyPrefix) {
			return
		}
		k = strings.TrimPrefix(k, linkKeyPrefix)
		lepb := new(pb.LinkEntry)
		if err := proto.Unmarshal(v, lepb); err != nil {
			return
		}
		visit(k, lepb)
	})
	return nil
}
