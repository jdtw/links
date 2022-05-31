package links

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
)

type authHandler func(http.ResponseWriter, *http.Request, string)

func (s *server) authenticated(f authHandler) http.HandlerFunc {
	if s.ks == nil {
		return func(w http.ResponseWriter, r *http.Request) {
			f(w, r, "")
		}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		subject, _, err := s.ks.AuthorizeRequest(r, s.nv)
		if err != nil {
			if dump, dumpErr := httputil.DumpRequest(r, false); dumpErr == nil {
				log.Printf("request unauthorized: %v\n%s", err, dump)
			}
			http.Error(w, fmt.Sprintf("unauthorized: %v", err), http.StatusUnauthorized)
			return
		}
		f(w, r, subject)
	}
}
