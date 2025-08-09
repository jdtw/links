package links

import (
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"google.golang.org/protobuf/encoding/protojson"
	pb "jdtw.dev/links/proto/links"
)

func (s *server) list() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rid := middleware.GetReqID(r.Context())

		lpb := &pb.Links{
			Links: make(map[string]*pb.Link),
		}
		s.store.Visit(r.Context(), func(k string, v *pb.LinkEntry) {
			lpb.Links[k] = v.Link
		})
		data, err := protojson.Marshal(lpb)
		if err != nil {
			internalError(w, err, rid)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}
}

func (s *server) get() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rid := middleware.GetReqID(r.Context())

		l := chi.URLParam(r, "link")
		lepb, err := s.store.Get(r.Context(), l)
		if err != nil {
			internalError(w, err, rid)
			return
		}
		if lepb == nil {
			http.NotFound(w, r)
			return
		}
		data, err := protojson.Marshal(lepb.Link)
		if err != nil {
			internalError(w, err, rid)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}
}

func (s *server) put() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rid := middleware.GetReqID(r.Context())

		l := chi.URLParam(r, "link")
		data, err := io.ReadAll(r.Body)
		if err != nil {
			internalError(w, err, rid)
			return
		}
		lpb := new(pb.Link)
		if err := protojson.Unmarshal(data, lpb); err != nil {
			badRequest(w, "failed to unmarshal body: %v", err)
			return
		}
		if lpb.Uri == "" {
			badRequest(w, "missing URI")
			return
		}
		// Create a dummy URI with all template parameters replaced
		// with something innocuous so that we can try to parse it.
		dummy := replacement.ReplaceAllString(lpb.Uri, "links")
		url, err := url.Parse(dummy)
		if err != nil {
			badRequest(w, "URI %q failed to parse: %v", lpb.Uri, err)
			return
		}
		if url.Scheme == "" {
			badRequest(w, "URI %q has no scheme", lpb.Uri)
			return
		}
		created, err := s.store.Put(r.Context(), l, lpb)
		if err != nil {
			internalError(w, err, rid)
			return
		}

		sub := subject(r.Context())
		if created {
			w.WriteHeader(http.StatusCreated)
			log.Printf("[%s] %s added %q -> %q", rid, sub, l, lpb.Uri)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		log.Printf("[%s] %s updated %q -> %q", rid, sub, l, lpb.Uri)
	}
}

func (s *server) delete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rid := middleware.GetReqID(r.Context())

		l := chi.URLParam(r, "link")
		s.store.Delete(r.Context(), l)
		w.WriteHeader(http.StatusNoContent)
		log.Printf("[%s] %s deleted %q", rid, subject(r.Context()), l)
	}
}
