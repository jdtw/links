package links

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"jdtw.dev/token"
	"jdtw.dev/token/nonce"
)

type server struct {
	store Store
	ks    *token.VerificationKeyset
	nv    nonce.Verifier
	skew  time.Duration
	*chi.Mux
}

func (s *server) routes() {
	s.Use(middleware.RequestID)
	s.Use(middleware.Logger)
	// REST API
	s.Route("/api", func(r chi.Router) {
		r.Use(s.authenticated())
		// Get all links as a Links proto.
		r.Get("/links", s.list())
		// Get a speficic link.
		r.Get("/links/{link}", s.get())
		// Create or update a link.
		r.Put("/links/{link}", s.put())
		// Remove a link.
		r.Delete("/links/{link}", s.delete())
	})

	// Application
	s.Get("/*", s.redirect())
}

// NewHandler sets up routes based on the given key value store.
func NewHandler(store Store, ks *token.VerificationKeyset, skew time.Duration) http.Handler {
	srv := &server{store, ks, nonce.NewMapVerifier(time.Minute), skew, chi.NewRouter()}
	srv.routes()
	return srv
}

type contextKey struct {
	name string
}

func (k *contextKey) String() string {
	return "jdtw.dev/links " + k.name
}

func internalError(w http.ResponseWriter, err error, rid string) {
	log.Printf("[%s] internal error: %v", rid, err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func badRequest(w http.ResponseWriter, format string, a ...interface{}) {
	http.Error(w, fmt.Sprintf(format, a...), http.StatusBadRequest)
}
