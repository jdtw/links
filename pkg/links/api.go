package links

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"
	pb "github.com/jdtw/links/proto/links"
	"google.golang.org/protobuf/proto"
)

func (s *server) list() authHandler {
	return func(w http.ResponseWriter, r *http.Request, sub string) {
		lpb := &pb.Links{
			Links: make(map[string]*pb.Link),
		}
		s.visitLinkEntries(func(k string, v *pb.LinkEntry) {
			lpb.Links[k] = v.Link
		})
		data, err := proto.Marshal(lpb)
		if err != nil {
			internalError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/x-protobuf")
		w.Write(data)
	}
}

func (s *server) get() authHandler {
	return func(w http.ResponseWriter, r *http.Request, sub string) {
		l := mux.Vars(r)["link"]
		lepb, err := s.getLinkEntry(l)
		if err != nil {
			internalError(w, err)
			return
		}
		if lepb == nil {
			http.NotFound(w, r)
			return
		}
		data, err := proto.Marshal(lepb.Link)
		if err != nil {
			internalError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/x-protobuf")
		w.Write(data)
	}
}

func (s *server) put() authHandler {
	return func(w http.ResponseWriter, r *http.Request, sub string) {
		l := mux.Vars(r)["link"]
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			internalError(w, err)
			return
		}
		lpb := new(pb.Link)
		if err := proto.Unmarshal(data, lpb); err != nil {
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
		created, err := s.putLinkEntry(l, lpb)
		if err != nil {
			internalError(w, err)
			return
		}
		if created {
			w.WriteHeader(http.StatusCreated)
			log.Printf("%s added %q -> %q", sub, l, lpb.Uri)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		log.Printf("%s updated %q -> %q", sub, l, lpb.Uri)
	}
}

func (s *server) delete() authHandler {
	return func(w http.ResponseWriter, r *http.Request, sub string) {
		l := mux.Vars(r)["link"]
		s.deleteLinkEntry(l)
		w.WriteHeader(http.StatusNoContent)
		log.Printf("%s deleted %q", sub, l)
	}
}
