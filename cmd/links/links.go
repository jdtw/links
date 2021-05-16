package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/jdtw/links/pkg/links"
	"github.com/lestrrat-go/jwx/jwk"
)

var (
	port     = flag.Int("port", 9090, "Port")
	keyset   = flag.String("keyset", "", "Verification keyset")
	database = flag.String("database", "", "Database directory")
)

func main() {
	flag.Parse()
	log.SetPrefix("links: ")

	var ks jwk.Set
	if *keyset != "" {
		bs, err := ioutil.ReadFile(*keyset)
		if err != nil {
			log.Fatalf("ioutil.ReadFile(%s) failed: %v", *keyset, err)
		}
		ks = jwk.NewSet()
		if err := json.Unmarshal(bs, ks); err != nil {
			log.Fatalf("json.Unmarshal(%s) failed: %v", *keyset, err)
		}
	}

	if *database != "" {
		if err := os.MkdirAll(*database, os.ModePerm); err != nil {
			log.Fatalf("os.MkdirAll(%v) failed: %v", *database, err)
		}
	}
	kv, err := links.NewKV(*database)
	if err != nil {
		log.Fatalf("links.NewKV(%v) failed: %v", *database, err)
	}

	addr := fmt.Sprint(":", *port)
	log.Printf("listening on %q", addr)
	log.Fatal(http.ListenAndServe(addr, links.NewHandler(kv, ks)))
}
