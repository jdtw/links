package links

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"jdtw.dev/links/pkg/token"
)

type server struct {
	kv *KV
	ks *token.VerificationKeyset
	*mux.Router
}

func (s *server) routes() {
	// REST API
	a := s.PathPrefix("/api").Subrouter()
	// Get all links as a Links proto.
	a.HandleFunc("/links", logged(s.authenticated(s.list()))).Methods("GET")
	// Get a speficic link.
	a.HandleFunc("/links/{link}", logged(s.authenticated(s.get()))).Methods("GET")
	// Create or update a link.
	a.HandleFunc("/links/{link}", logged(s.authenticated(s.put()))).Methods("PUT")
	// Remove a link.
	a.HandleFunc("/links/{link}", logged(s.authenticated(s.delete()))).Methods("DELETE")

	// Application
	s.PathPrefix("/").HandlerFunc(logged(s.redirect())).Methods("GET")
}

// NewHandler sets up routes based on the given key value store.
func NewHandler(kv *KV, ks *token.VerificationKeyset) http.Handler {
	srv := &server{kv, ks, mux.NewRouter()}
	srv.routes()
	return srv
}

func internalError(w http.ResponseWriter, err error) {
	log.Printf("internal error: %v", err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func badRequest(w http.ResponseWriter, format string, a ...interface{}) {
	http.Error(w, fmt.Sprintf(format, a...), http.StatusBadRequest)
}

// logged logs the HTTP request, respecting the X-Forwarded-For header to support
// running behind a proxy.
func logged(hf http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		remote := strings.Join(r.Header["X-Forwarded-For"], " ")
		if remote == "" {
			remote = r.RemoteAddr
		}
		log.Printf("%s %s %s %s %s", remote, r.Method, r.Host, r.URL, r.UserAgent())
		hf(w, r)
	}
}
