package links

import (
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	pb "github.com/jdtw/links/proto/links"
	"google.golang.org/protobuf/proto"
)

func (s *server) list() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

func (s *server) add() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

func (s *server) get() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

func (s *server) put() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *server) delete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := mux.Vars(r)["link"]
		s.deleteLinkEntry(l)
		w.WriteHeader(http.StatusNoContent)
	}
}
