package links

import (
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// KV is a key value store optionally backed by persistant storage
// in the form of a database directory, where key value pairs are
// stored as one file per entry.
type KV struct {
	rw  sync.RWMutex
	kv  map[string][]byte
	dir string
}

func (db *KV) keyPath(k string) (string, bool) {
	if db.dir == "" {
		return "", false
	}
	return filepath.Join(db.dir, hex.EncodeToString([]byte(k))), true
}

// NewMemKV returns an in-memory key value store.
func NewMemKV() *KV {
	kv, _ := NewKV("")
	return kv
}

// NewKV returns a key value store that stores updates to the given directory.
func NewKV(dir string) (*KV, error) {
	kv := make(map[string][]byte)
	if dir != "" {
		if err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				if path != dir {
					// All subdirectories are ignored.
					return fs.SkipDir
				}
				return nil
			}
			v, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			k, err := hex.DecodeString(filepath.Base(path))
			if err != nil {
				return nil
			}
			kv[string(k)] = v
			return nil
		}); err != nil {
			return nil, err
		}
	}
	return &KV{kv: kv, dir: dir}, nil
}

func (db *KV) Put(k string, v []byte) (bool, error) {
	db.rw.Lock()
	defer db.rw.Unlock()
	_, present := db.kv[k]
	if p, ok := db.keyPath(k); ok {
		if err := os.WriteFile(p, v, os.ModePerm); err != nil {
			return false, err
		}
	}
	db.kv[k] = v
	return !present, nil
}

func (m *KV) Get(k string) []byte {
	m.rw.RLock()
	defer m.rw.RUnlock()
	return m.kv[k]
}

// Delete removes an entry from the store and deletes the backing
// file.
func (db *KV) Delete(k string) error {
	db.rw.Lock()
	defer db.rw.Unlock()
	if _, ok := db.kv[k]; ok {
		if p, ok := db.keyPath(k); ok {
			if err := os.Remove(p); err != nil {
				return err
			}
		}
		delete(db.kv, k)
	}
	return nil
}

// Iterate iterates over all key value pairs, calling the supplied
// callback. It is not safe to refert to the value slice outside
// of the callback, unless it is copied.
func (m *KV) Iterate(visit func(k string, v []byte)) {
	m.rw.RLock()
	defer m.rw.RUnlock()
	for k, v := range m.kv {
		visit(k, v)
	}
}
