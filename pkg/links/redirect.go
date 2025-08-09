package links

import (
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
	qrcode "github.com/skip2/go-qrcode"
)

// Index is used for special handling for the root path; it is stored
// in the database as the "index" key.
const Index = ".index"

func (s *server) redirect() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rid := middleware.GetReqID(r.Context())

		// The first path segment is the key into our DB.
		// The remaining path segments are paths to be appended
		// or substituted in the redirect.
		path := r.URL.Path[1:]
		split := strings.Split(path, "/")
		key, paths := split[0], split[1:]
		if key == "" {
			key = Index
		}

		// If prefixed with the /qr/ path, show a QR instead of redirecting.
		qr := key == "qr"
		if qr {
			if len(paths) == 0 {
				key = Index
			} else {
				key, paths = paths[0], paths[1:]
			}
		}

		// Look up the key, and unmarshal the LinkEntry from the DB
		key = strings.ReplaceAll(key, "-", "")
		le, err := s.store.Get(r.Context(), key)
		if err != nil {
			internalError(w, err, rid)
			return
		}
		if le == nil {
			http.NotFound(w, r)
			return
		}

		// Get the URI and optionally perform substitutions on it.
		// For example, given paths ["bar", "foo"] and URI "example.com/{1}/{0}/baz",
		// we end up with "example.com/foo/bar/baz"
		uri, paths, err := subst(le, paths)
		if err != nil {
			badRequest(w, err.Error())
			return
		}

		loc, err := url.Parse(uri)
		if err != nil {
			internalError(w, err, rid)
			return
		}
		if len(paths) > 0 {
			loc.Path += "/" + strings.Join(paths, "/")
		}
		if r.URL.ForceQuery {
			loc.ForceQuery = true
		}
		if rq := r.URL.RawQuery; rq != "" {
			loc.RawQuery = rq
		}
		if qr {
			png, err := qrcode.Encode(loc.String(), qrcode.High, 256)
			if err != nil {
				internalError(w, err, rid)
				return
			}
			w.Header().Set("Content-Type", "image/png")
			w.Write(png)
			return
		}
		log.Printf("[%s] redirecting %s to %s", rid, r.URL, loc)
		http.Redirect(w, r, loc.String(), http.StatusFound)
	}
}
