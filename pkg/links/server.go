package links

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/lestrrat-go/jwx/jwk"
)

type server struct {
	kv *KV
	ks jwk.Set
	*mux.Router
}

func (s *server) routes() {
	// REST API
	a := s.PathPrefix("/api").Subrouter()
	// Get all links as a Links proto.
	a.HandleFunc("/links", s.authenticated(s.list())).Methods("GET")
	// Get a speficic link.
	a.HandleFunc("/links/{link}", s.authenticated(s.get())).Methods("GET")
	// Create or update a link.
	a.HandleFunc("/links/{link}", s.authenticated(s.put())).Methods("PUT")
	// Remove a link.
	a.HandleFunc("/links/{link}", s.authenticated(s.delete())).Methods("DELETE")

	// Application
	s.PathPrefix("/").HandlerFunc(s.redirect())
}

// NewHandler sets up routes based on the given key value store.
func NewHandler(kv *KV, ks jwk.Set) http.Handler {
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
