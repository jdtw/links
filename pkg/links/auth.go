package links

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"

	"github.com/go-chi/chi/v5/middleware"
)

var subjectCtxKey = &contextKey{"Subject"}

func (s *server) authenticated() func(next http.Handler) http.Handler {
	if s.ks == nil {
		log.Printf("server missing keyset!")
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "server missing keyset", http.StatusUnauthorized)
			})
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rid := middleware.GetReqID(r.Context())
			subject, _, err := s.ks.AuthorizeRequest(r, s.skew, s.nv)
			if err != nil {
				if dump, dumpErr := httputil.DumpRequest(r, false); dumpErr == nil {
					log.Printf("[%s] request unauthorized: %v\n%s", rid, err, dump)
				}
				http.Error(w, fmt.Sprintf("unauthorized: %v", err), http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), subjectCtxKey, subject)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func subject(ctx context.Context) string {
	if user, ok := ctx.Value(subjectCtxKey).(string); ok {
		return user
	}
	return ""
}
