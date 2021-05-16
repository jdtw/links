package links

import "testing"

func TestGetPutDelete(t *testing.T) {
	mkv := NewMemKV()
	if !mkv.Put("foo", []byte("bar")) {
		t.Fatalf(`Put("foo") = false; want true`)
	}
	if got := mkv.Get("foo"); string(got) != "bar" {
		t.Fatalf(`Get("foo") = %q; want "bar"`, got)
	}
	if mkv.Put("foo", []byte("baz")) {
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
