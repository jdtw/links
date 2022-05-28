package frontend

import (
	"html/template"
	"log"
	"net/http"
	"sort"
	"strings"

	"github.com/gorilla/mux"
	"jdtw.dev/links/pkg/client"
)

const html = `
<h1>Add Link</h1>
<form method="POST">
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
<h1>Links</h1>
<table>
  <tr><th>Link</th><th>URI</th></tr>
  {{range .}}
  <tr><td>{{.Link}}</td><td><a href="{{.URI}}">{{.URI}}</a></td></tr>
  {{end}}
</table>
`

// NewHandler creates a new frontent handler with the given client.
func NewHandler(cli *client.Client) http.Handler {
	srv := &server{cli, mux.NewRouter()}
	srv.routes()
	return srv
}

type server struct {
	cli *client.Client
	*mux.Router
}

func (s *server) routes() {
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
		tmpl.Execute(w, sortLinks(m))
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
