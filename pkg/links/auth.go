package links

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/jdtw/links/pkg/auth"
)

func unauthorized(w http.ResponseWriter, err error) {
	log.Printf("unauthorized: %v", err)
	http.Error(w, err.Error(), http.StatusUnauthorized)
}

func (s *server) authenticated(f http.HandlerFunc) http.HandlerFunc {
	// TODO(jdtw): We should fail closed.
	if s.ks == nil {
		return f
	}
	return func(w http.ResponseWriter, r *http.Request) {
		token, err := extractToken(r)
		if err != nil {
			unauthorized(w, err)
			return
		}
		if _, err := auth.VerifyJWT(s.ks, token, auth.ServerAudience(r)); err != nil {
			unauthorized(w, err)
			return
		}
		f(w, r)
	}
}

func extractToken(r *http.Request) ([]byte, error) {
	authz := r.Header["Authorization"]
	if len(authz) != 1 {
		return nil, fmt.Errorf("expected 1 authorization header, got %d", len(authz))
	}
	const bearer = "Bearer "
	if !strings.HasPrefix(authz[0], bearer) {
		return nil, fmt.Errorf("authorization header missing %q prefix", bearer)
	}
	return []byte(strings.TrimPrefix(authz[0], bearer)), nil
}
