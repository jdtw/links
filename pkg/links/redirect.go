package links

import (
	"log"
	"net/http"
	"net/url"
	"strings"
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

		// Look up the key, and unmarshal the LinkEntry from the DB
		le, err := s.getLinkEntry(key)
		if err != nil {
			internalError(w, err)
			return
		}
		// If that wasn't found and the key contains a hypen, try again
		// with the hyphen(s) removed.
		if le == nil && strings.ContainsRune(key, '-') {
			key = strings.ReplaceAll(key, "-", "")
			le, err = s.getLinkEntry(key)
			if err != nil {
				internalError(w, err)
				return
			}
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
		log.Printf("redirecting %s to %s", r.URL, loc)
		http.Redirect(w, r, loc.String(), http.StatusFound)
	}
}
