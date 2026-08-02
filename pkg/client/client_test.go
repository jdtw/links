package client

import (
	"errors"
	"net/http/httptest"
	"testing"

	"jdtw.dev/links/pkg/links"
	"jdtw.dev/links/pkg/tokentest"
)

func TestClient(t *testing.T) {
	ks, signer := tokentest.GenerateKey(t, "test")
	store := links.NewMemStore()

	// httptest.NewServer binds before it returns, so the client below cannot
	// race the listener. The previous version picked a port, closed it, then
	// started the server in a goroutine and immediately dialed -- which meant
	// the request could arrive before the server was listening (and left a
	// window for another process to take the port). That raced on loaded CI
	// runners.
	s := httptest.NewServer(links.NewHandler(store, ks, 0))
	t.Cleanup(s.Close)

	c := New(s.URL, signer)
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
	// Test that Put strips whitespace.
	if err := c.Put(" whitespace ", "http://bar"); err != nil {
		t.Fatalf("client.Put(whitespace, http://bar) failed: %v", err)
	}
	{
		got, err := c.Get("whitespace")
		if err != nil {
			t.Fatalf("client.Get(whitespace) failed: %v", err)
		}
		if got != "http://bar" {
			t.Fatalf("client.Get(whitespace) = %v, want http://bar", got)
		}
	}

}
