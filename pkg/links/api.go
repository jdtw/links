package links

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"

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

		l := normalizeKey(chi.URLParam(r, "link"))
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

// validateLink reports whether a normalized key and link are acceptable to
// store, describing the problem if they are not. Shared by put() and
// bulkPut() so a bulk import enforces exactly the same rules as a single
// write.
func validateLink(key string, l *pb.Link) error {
	if key == qrKey {
		return fmt.Errorf("%q is a reserved link name", qrKey)
	}
	if l.GetUri() == "" {
		return errors.New("missing URI")
	}
	// Create a dummy URI with all template parameters replaced
	// with something innocuous so that we can try to parse it.
	dummy := replacement.ReplaceAllString(l.GetUri(), "links")
	url, err := url.Parse(dummy)
	if err != nil {
		return fmt.Errorf("URI %q failed to parse: %v", l.GetUri(), err)
	}
	if url.Scheme == "" {
		return fmt.Errorf("URI %q has no scheme", l.GetUri())
	}
	return nil
}

func (s *server) put() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rid := middleware.GetReqID(r.Context())

		l := normalizeKey(chi.URLParam(r, "link"))
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
		if err := validateLink(l, lpb); err != nil {
			badRequest(w, "%v", err)
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

// bulkPut creates or updates every link in the request body, which is a
// Links proto of the same shape that list() returns. Links already in the
// store that the body does not mention are left alone, so an import is
// additive rather than a replacement.
//
// Every entry is validated before anything is written: one malformed link
// fails the whole request rather than leaving a half-applied import.
func (s *server) bulkPut() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rid := middleware.GetReqID(r.Context())

		data, err := io.ReadAll(r.Body)
		if err != nil {
			internalError(w, err, rid)
			return
		}
		lpb := new(pb.Links)
		if err := protojson.Unmarshal(data, lpb); err != nil {
			badRequest(w, "failed to unmarshal body: %v", err)
			return
		}
		if len(lpb.GetLinks()) == 0 {
			badRequest(w, "no links in request body")
			return
		}

		// Normalize and validate everything before the first write. Keys
		// that collide only after normalization ("my-link" and "mylink")
		// would silently overwrite each other, so reject those too.
		normalized := make(map[string]*pb.Link, len(lpb.GetLinks()))
		sources := make(map[string]string, len(lpb.GetLinks()))
		var problems []string
		for k, l := range lpb.GetLinks() {
			key := normalizeKey(k)
			if err := validateLink(key, l); err != nil {
				problems = append(problems, fmt.Sprintf("%q: %v", k, err))
				continue
			}
			if prev, dup := sources[key]; dup {
				problems = append(problems, fmt.Sprintf("%q: collides with %q after normalization", k, prev))
				continue
			}
			sources[key] = k
			normalized[key] = l
		}
		if len(problems) > 0 {
			sort.Strings(problems)
			badRequest(w, "rejected %d of %d links:\n%s", len(problems), len(lpb.GetLinks()), strings.Join(problems, "\n"))
			return
		}

		var created, updated int
		for k, l := range normalized {
			wasCreated, err := s.store.Put(r.Context(), k, l)
			if err != nil {
				internalError(w, err, rid)
				return
			}
			if wasCreated {
				created++
			} else {
				updated++
			}
		}

		w.WriteHeader(http.StatusNoContent)
		log.Printf("[%s] %s imported %d links (%d created, %d updated)",
			rid, subject(r.Context()), len(normalized), created, updated)
	}
}

func (s *server) delete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rid := middleware.GetReqID(r.Context())

		l := normalizeKey(chi.URLParam(r, "link"))
		s.store.Delete(r.Context(), l)
		w.WriteHeader(http.StatusNoContent)
		log.Printf("[%s] %s deleted %q", rid, subject(r.Context()), l)
	}
}
