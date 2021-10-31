package links

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jdtw/links/pkg/token"
)

func unauthorized(w http.ResponseWriter, err error) {
	log.Printf("unauthorized: %v", err)
	http.Error(w, err.Error(), http.StatusUnauthorized)
}

type AuthHandler func(http.ResponseWriter, *http.Request, string)

func (s *server) authenticated(f AuthHandler) http.HandlerFunc {
	if s.ks == nil {
		return func(w http.ResponseWriter, r *http.Request) {
			f(w, r, "")
		}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		t, err := extractToken(r)
		if err != nil {
			unauthorized(w, err)
			return
		}
		sub, err := s.ks.Verify(t, token.CheckRequestResource(r), token.CheckExpiry(time.Now()))
		if err != nil {
			unauthorized(w, err)
			return
		}
		f(w, r, sub)
	}
}

func extractToken(r *http.Request) (string, error) {
	authz := r.Header["Authorization"]
	if len(authz) != 1 {
		return "", fmt.Errorf("expected 1 authorization header, got %d", len(authz))
	}
	const bearer = "Bearer "
	if !strings.HasPrefix(authz[0], bearer) {
		return "", fmt.Errorf("authorization header missing %q prefix", bearer)
	}
	return strings.TrimPrefix(authz[0], bearer), nil
}
