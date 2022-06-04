package links

import (
	"context"
	"testing"

	pb "jdtw.dev/links/proto/links"
)

func TestGetPutDelete(t *testing.T) {
	ctx := context.Background()
	s := NewMemStore()
	if created, _ := s.Put(ctx, "foo", &pb.Link{Uri: "bar"}); !created {
		t.Fatalf(`Put("foo") = false; want true`)
	}
	if got, _ := s.Get(ctx, "foo"); got.Link.Uri != "bar" {
		t.Fatalf(`Get("foo") = %v; want "bar"`, got)
	}
	if created, _ := s.Put(ctx, "foo", &pb.Link{Uri: "baz"}); created {
		t.Fatalf(`Put("foo") = true; want false`)
	}
	if got, _ := s.Get(ctx, "foo"); got.Link.Uri != "baz" {
		t.Fatalf(`Get("foo") = %v; want "baz"`, got)
	}
	s.Delete(ctx, "foo")
	if got, _ := s.Get(ctx, "foo"); got != nil {
		t.Fatalf(`Get("foo") = %q; want ""`, got)
	}
}
