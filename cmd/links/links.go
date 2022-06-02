package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"jdtw.dev/links/pkg/links"
	"jdtw.dev/token"
)

var (
	port      = flag.Int("port", 9090, "Port")
	ephemeral = flag.Bool("ephemeral", false, "If true, don't connect to DATABASE_URL and use in-memory storage")
)

func main() {
	flag.Parse()
	log.SetPrefix("links: ")

	encoded := os.Getenv("LINKS_KEYSET")
	if encoded == "" {
		log.Fatal("LINKS_KEYSET environment variable must be set")
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		log.Fatalf("base64 decoding keyset failed: %v", err)
	}
	keyset, err := token.UnmarshalKeyset(decoded)
	if err != nil {
		log.Fatalf("token.UnmarshalKeyset failed: %v", err)
	}
	log.Printf("loaded keyset:\n%s", keyset)

	var store links.Store
	if *ephemeral {
		log.Printf("Running in ephemeral mode!")
		store = links.NewMemStore()
	} else {
		pgStore, err := links.NewPostgresStore(os.Getenv("DATABASE_URL"))
		if err != nil {
			log.Fatalf("links.NewPostgresStore failed: %v", err)
		}
		store = pgStore
		defer pgStore.Close()
	}

	addr := fmt.Sprint(":", *port)
	log.Printf("listening on %q", addr)
	log.Fatal(http.ListenAndServe(addr, links.NewHandler(store, keyset)))
}
