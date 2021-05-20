package frontend

import (
	"html/template"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jdtw/links/pkg/client"
)

const form = `{{if .Success}}
<h1>Link added!</h1>
{{else}}
<h1>Add Link</h1>
<form method="POST">
  <label>Link:</label><br />
  <input type="text" name="link"><br />
  <label>URI:</label><br />
  <input type="text" name="uri"><br />
  <input type="submit">
</form>
{{end}}
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
	tmpl := template.Must(template.New("link").Parse(form))
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			tmpl.Execute(w, nil)
			return
		}

		link := r.FormValue("link")
		uri := r.FormValue("uri")

		if err := s.cli.Put(link, uri); err != nil {
			log.Printf("internal error: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		tmpl.Execute(w, struct{ Success bool }{true})
	}
}
