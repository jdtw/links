package links

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"jdtw.dev/links/pkg/token"
)

type authHandler func(http.ResponseWriter, *http.Request, string)

func (s *server) authenticated(f authHandler) http.HandlerFunc {
	if s.ks == nil {
		return func(w http.ResponseWriter, r *http.Request) {
			f(w, r, "")
		}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		subject, err := s.ks.AuthorizeRequest(r, token.CheckExpiry(time.Now()))
		if err != nil {
			log.Printf("request %+v unauthorized: %v", r, err)
			http.Error(w, fmt.Sprintf("unauthorized: %v", err), http.StatusUnauthorized)
			return
		}
		f(w, r, subject)
	}
}
