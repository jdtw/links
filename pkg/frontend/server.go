package frontend

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
	"jdtw.dev/links/pkg/client"
)

//go:embed static/*
var static embed.FS

const html = `
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
	<link rel="stylesheet" href="static/styles.css" />
  <title>Links</title>
</head>
<body>
<h1>➕ Add Link</h1>
<form id="link-form" method="POST">
  <table>
  <tr>
    <td><label>Link:</label></td>
    <td><input type="text" name="link"></td>
  </tr>
  <tr>
    <td><label>URI:</label></td>
    <td><input type="text" name="uri"></td>
  </tr>
  </table>
  <input type="submit">
</form>
<h1>🔗 Links</h1>
<table id="links">
  <tr><th>Link</th><th></th><th></th><th>URI</th></tr>
  {{range .}}
  <tr>
    <td>{{.Link}}</td>
    <td><button title="Edit" data-edit="{{.Link}}">🖋️️</button></td>
    <td><button title="Delete" data-remove="{{.Link}}">❌</button></td>
    <td><a id="{{.Link}}" href="{{.URI}}">{{.URI}}</a></td>
  </tr>
  {{end}}
</table>
<script src="/static/links.js"></script>
</body>
</html>
`

// NewHandler creates a new frontent handler with the given client.
func NewHandler(cli *client.Client) http.Handler {
	srv := &server{cli, chi.NewRouter()}
	srv.routes()
	return srv
}

type server struct {
	cli *client.Client
	*chi.Mux
}

func (s *server) routes() {
	s.Handle("/static/*", http.FileServer(http.FS(static)))
	s.Delete("/rm/{link}", s.removeLink())
	s.HandleFunc("/", s.addLink())
}

func (s *server) addLink() http.HandlerFunc {
	tmpl := template.Must(template.New("link").Parse(html))
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			link := r.FormValue("link")
			uri := r.FormValue("uri")
			if link == "" || uri == "" {
				http.Error(w, "missing link or URI", http.StatusBadRequest)
				return
			}
			if err := s.cli.Put(link, uri); err != nil {
				log.Printf("Put link failed: %v", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		m, err := s.cli.List()
		if err != nil {
			log.Printf("List links failed: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := tmpl.Execute(w, sortLinks(m)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (s *server) removeLink() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		link := chi.URLParam(r, "link")
		if err := s.cli.Delete(link); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Printf("Removed %s", link)
	}
}

type link struct {
	Link string
	URI  string
}

func sortLinks(m map[string]string) []*link {
	ls := make([]*link, 0, len(m))
	for k, v := range m {
		ls = append(ls, &link{k, v})
	}
	sort.SliceStable(ls, func(i, j int) bool {
		return strings.Compare(ls[i].Link, ls[j].Link) < 0
	})
	return ls
}
