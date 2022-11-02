package links

import (
	"log"
	"net/http"
	"net/url"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

// Index is used for special handling for the root path; it is stored
// in the database as the "index" key.
const Index = ".index"

func (s *server) redirect() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		qr := false
		if key == "qr" {
			qr = true
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
			internalError(w, err)
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
			internalError(w, err)
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
			writeQR(w, loc)
			return
		}
		log.Printf("redirecting %s to %s", r.URL, loc)
		http.Redirect(w, r, loc.String(), http.StatusFound)
	}
}

func writeQR(w http.ResponseWriter, u *url.URL) {
	png, err := qrcode.Encode(u.String(), qrcode.High, 256)
	if err != nil {
		log.Printf("qrcode.Encode(%s) failed: %v", u, err)
		internalError(w, err)
	}
	w.Header().Set("Content-Type", "image/png")
	w.Write(png)
}
