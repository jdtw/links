package links

import (
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	pb "github.com/jdtw/links/proto/links"
	"google.golang.org/protobuf/proto"
)

func (s *server) list() AuthHandler {
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

func (s *server) add() AuthHandler {
	return func(w http.ResponseWriter, r *http.Request, sub string) {
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			internalError(w, err)
			return
		}
		ls := new(pb.Links)
		if err := proto.Unmarshal(data, ls); err != nil {
			internalError(w, err)
			return
		}
		if err := s.addLinks(ls); err != nil {
			internalError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *server) get() AuthHandler {
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

func (s *server) put() AuthHandler {
	return func(w http.ResponseWriter, r *http.Request, sub string) {
		l := mux.Vars(r)["link"]
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			internalError(w, err)
			return
		}
		lpb := new(pb.Link)
		if err := proto.Unmarshal(data, lpb); err != nil {
			internalError(w, err)
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

func (s *server) delete() AuthHandler {
	return func(w http.ResponseWriter, r *http.Request, sub string) {
		l := mux.Vars(r)["link"]
		s.deleteLinkEntry(l)
		w.WriteHeader(http.StatusNoContent)
		log.Printf("%s deleted %q", sub, l)
	}
}
