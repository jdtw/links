package frontend

import (
	"net/http"
	"net/url"
)

// sameOrigin rejects requests whose Origin header (or, failing that,
// Referer) doesn't match the request's own Host. This server has no
// auth of its own -- it signs whatever it's told with a held key -- so
// without this check, a form or fetch() triggered from any other page
// loaded on the same network could add or remove links on the caller's
// behalf. Browsers set Origin on cross-origin requests and don't let
// page script override it, so a forged request can't fake a match.
func sameOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		src := r.Header.Get("Origin")
		if src == "" {
			src = r.Header.Get("Referer")
		}
		u, err := url.Parse(src)
		if src == "" || err != nil || u.Host != r.Host {
			http.Error(w, "cross-origin request rejected", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
