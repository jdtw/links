package links

import (
	"sync"

	pb "jdtw.dev/links/proto/links"
)

// MemStore is an in-memory store of links.
type MemStore struct {
	entries map[string]*pb.LinkEntry
	sync.RWMutex
}

var _ Store = &MemStore{}

func NewMemStore() *MemStore {
	return &MemStore{make(map[string]*pb.LinkEntry), sync.RWMutex{}}
}

func (s *MemStore) Get(k string) (*pb.LinkEntry, error) {
	s.RLock()
	defer s.RUnlock()
	return s.entries[k], nil
}

func (s *MemStore) Put(k string, l *pb.Link) (bool, error) {
	le := &pb.LinkEntry{
		Link:          l,
		RequiredPaths: requiredPaths(l),
	}
	s.Lock()
	defer s.Unlock()
	_, present := s.entries[k]
	s.entries[k] = le
	return !present, nil
}

func (s *MemStore) Delete(k string) error {
	s.Lock()
	defer s.Unlock()
	delete(s.entries, k)
	return nil
}

func (s *MemStore) Visit(visit func(string, *pb.LinkEntry)) error {
	s.RLock()
	defer s.RUnlock()
	for k, v := range s.entries {
		visit(k, v)
	}
	return nil
}
