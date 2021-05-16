package links

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetPutDelete(t *testing.T) {
	mkv := NewMemKV()
	if created, _ := mkv.Put("foo", []byte("bar")); !created {
		t.Fatalf(`Put("foo") = false; want true`)
	}
	if got := mkv.Get("foo"); string(got) != "bar" {
		t.Fatalf(`Get("foo") = %q; want "bar"`, got)
	}
	if created, _ := mkv.Put("foo", []byte("baz")); created {
		t.Fatalf(`Put("foo") = true; want false`)
	}
	if got := mkv.Get("foo"); string(got) != "baz" {
		t.Fatalf(`Get("foo") = %q; want "baz"`, got)
	}
	mkv.Delete("foo")
	if got := mkv.Get("foo"); got != nil {
		t.Fatalf(`Get("foo") = %q; want ""`, got)
	}
}

func TestPersistantStorage(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "db")
	if err := os.Mkdir(dir, os.ModePerm); err != nil {
		t.Fatalf("os.Mkdir(%s) failed: %v", dir, err)
	}
	kv, err := NewKV(dir)
	if err != nil {
		t.Fatalf("NewKv(%s) failed: %v", dir, err)
	}
	kv.Put("foo", []byte("bar"))
	kv, err = NewKV(dir)
	if err != nil {
		t.Fatalf("NewKv(%s) failed: %v", dir, err)
	}
	if got := kv.Get("foo"); string(got) != "bar" {
		t.Fatalf("kv.Get(foo) = %s, want foo", got)
	}
	if err := kv.Delete("foo"); err != nil {
		t.Fatalf("kv.Delete(foo) failed: %v", err)
	}
	kv, err = NewKV(dir)
	if err != nil {
		t.Fatalf("NewKv(%s) failed: %v", dir, err)
	}
	if got := kv.Get("foo"); got != nil {
		t.Fatalf("kv.Get(foo) = %v, want nil", got)
	}
}

func TestDeleteNoOp(t *testing.T) {
	kv, err := NewKV(t.TempDir())
	if err != nil {
		t.Fatalf("NewKv(%s) failed: %v", t.TempDir(), err)
	}
	if err := kv.Delete("does-not-exist"); err != nil {
		t.Fatalf("kv.Delete(does-not-exist) failed: %v", err)
	}
}

func TestIterate(t *testing.T) {
	mkv := NewMemKV()
	items := map[string]string{
		"foo":    "bar",
		"baz":    "qux",
		"apples": "oranges",
		"":       "",
	}
	for k, v := range items {
		mkv.Put(k, []byte(v))
	}
	mkv.Iterate(func(k string, v []byte) {
		if want := items[k]; string(v) != want {
			t.Fatalf("got value %q for key %q; want value %q", v, k, want)
		}
		delete(items, k)
	})
	if len(items) > 0 {
		t.Fatalf("iteration incomplete, missed key/values %v", items)
	}
}
