package client

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"testing"

	"jdtw.dev/links/pkg/links"
	"jdtw.dev/links/pkg/tokentest"
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
	ks, signer := tokentest.GenerateKey(t, "test")
	store := links.NewMemStore()
	var wg sync.WaitGroup
	wg.Add(1)
	addr := fmt.Sprintf("localhost:%d", getFreePort(t))
	s := &http.Server{Addr: addr, Handler: links.NewHandler(store, ks, 0)}
	go func() {
		defer wg.Done()
		s.ListenAndServe()
	}()
	ctx := context.Background()
	t.Cleanup(func() {
		s.Shutdown(ctx)
		wg.Wait()
	})

	c := New("http://"+addr, signer)
	if _, err := c.Get("foo"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get(foo) returned %v; want err %v", err, ErrNotFound)
	}
	if err := c.Put("foo", "http://bar"); err != nil {
		t.Fatalf("client.Put(foo, http://bar) failed: %v", err)
	}
	{
		got, err := c.Get("foo")
		if err != nil {
			t.Fatalf("client.Get(foo) failed: %v", err)
		}
		if got != "http://bar" {
			t.Fatalf("client.Get(foo) = %v, want http://bar", got)
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
		if got["foo"] != "http://bar" {
			t.Fatalf("client.List[foo] = %v, want bar", got["foo"])
		}
	}
	{
		if err := c.Delete("foo"); err != nil {
			t.Fatalf("client.Delete(foo) failed: %v", err)
		}
		if _, err := c.Get("foo"); !errors.Is(err, ErrNotFound) {
			t.Fatal("expected link foo to be deleted")
		}
	}
}
