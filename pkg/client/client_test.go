package client

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"testing"

	"github.com/jdtw/links/pkg/auth"
	"github.com/jdtw/links/pkg/links"
	"github.com/lestrrat-go/jwx/jwk"
)

func getFreePort(t *testing.T) int {
	t.Helper()
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("net.ResolveTCPAddr failed: %v", err)
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		t.Fatalf("net.ListenTCP failed: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func TestClient(t *testing.T) {
	pub, priv, err := auth.NewKey(rand.Reader, "test")
	if err != nil {
		t.Fatalf("auth.NewKey failed: %v", err)
	}
	ks := jwk.NewSet()
	ks.Add(pub)
	kv := links.NewMemKV()
	var wg sync.WaitGroup
	wg.Add(1)
	addr := fmt.Sprintf("localhost:%d", getFreePort(t))
	s := &http.Server{Addr: addr, Handler: links.NewHandler(kv, ks)}
	go func() {
		defer wg.Done()
		s.ListenAndServe()
	}()
	ctx := context.Background()
	t.Cleanup(func() {
		s.Shutdown(ctx)
		wg.Wait()
	})

	c := New("http://"+addr, priv)
	if _, err := c.Get("foo"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get(foo) returned %v; want err %v", err, ErrNotFound)
	}
	if err := c.Put("foo", "bar"); err != nil {
		t.Fatalf("client.Put(foo, bar) failed: %v", err)
	}
	{
		got, err := c.Get("foo")
		if err != nil {
			t.Fatalf("client.Get(foo) failed: %v", err)
		}
		if got != "bar" {
			t.Fatalf("client.Get(foo) = %v, want bar", got)
		}
	}
	{
		got, err := c.List()
		if err != nil {
			t.Fatalf("client.List failed: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("client.List len = %d, want 1", len(got))
		}
		if got["foo"] != "bar" {
			t.Fatalf("client.List[foo] = %v, want bar", got["foo"])
		}
	}
}
