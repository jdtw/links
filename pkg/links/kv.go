package links

import "sync"

// KV is an interface for a key value store
type KV interface {
	// Put returns true if created for the first time
	Add(map[string][]byte)
	Put(string, []byte) bool
	Get(string) []byte
	Delete(string)
	Iterate(func(string, []byte))
}

// NewMemKV returns an in-memory key value store.
func NewMemKV() KV {
	return &memKV{kv: make(map[string][]byte)}
}

type memKV struct {
	rw sync.RWMutex
	kv map[string][]byte
}

func (m *memKV) Add(kv map[string][]byte) {
	m.rw.Lock()
	defer m.rw.Unlock()
	for k, v := range kv {
		m.kv[k] = v
	}
}

func (m *memKV) Put(k string, v []byte) bool {
	m.rw.Lock()
	defer m.rw.Unlock()
	created := m.kv[k] == nil
	m.kv[k] = v
	return created
}

func (m *memKV) Get(k string) []byte {
	m.rw.RLock()
	defer m.rw.RUnlock()
	return m.kv[k]
}

func (m *memKV) Delete(k string) {
	m.rw.Lock()
	defer m.rw.Unlock()
	delete(m.kv, k)
}

func (m *memKV) Iterate(visit func(k string, v []byte)) {
	m.rw.RLock()
	defer m.rw.RUnlock()
	for k, v := range m.kv {
		visit(k, v)
	}
}
