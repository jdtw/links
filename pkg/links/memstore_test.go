package links

import (
	"testing"

	pb "jdtw.dev/links/proto/links"
)

func TestGetPutDelete(t *testing.T) {
	s := NewMemStore()
	if created, _ := s.Put("foo", &pb.Link{Uri: "bar"}); !created {
		t.Fatalf(`Put("foo") = false; want true`)
	}
	if got, _ := s.Get("foo"); got.Link.Uri != "bar" {
		t.Fatalf(`Get("foo") = %v; want "bar"`, got)
	}
	if created, _ := s.Put("foo", &pb.Link{Uri: "baz"}); created {
		t.Fatalf(`Put("foo") = true; want false`)
	}
	if got, _ := s.Get("foo"); got.Link.Uri != "baz" {
		t.Fatalf(`Get("foo") = %v; want "baz"`, got)
	}
	s.Delete("foo")
	if got, _ := s.Get("foo"); got != nil {
		t.Fatalf(`Get("foo") = %q; want ""`, got)
	}
}
