package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"jdtw.dev/links/pkg/links"
	"jdtw.dev/token"
)

var (
	port     = flag.Int("port", 9090, "Port")
	keyset   = flag.String("keyset", "", "Verification keyset")
	database = flag.String("database", "", "Database directory")
)

func main() {
	flag.Parse()
	log.SetPrefix("links: ")

	if *keyset == "" {
		*keyset = os.Getenv("LINKS_KEYSET")
	}
	if *keyset == "" {
		log.Fatal("missing 'keyset' flag")
	}

	ksContents, err := os.ReadFile(*keyset)
	if err != nil {
		log.Fatalf("os.ReadFile(%s) failed: %v", *keyset, err)
	}
	ks, err := token.UnmarshalKeyset(ksContents)
	if err != nil {
		log.Fatalf("token.UnmarshalKeyset(%s) failed: %v", *keyset, err)
	}

	var store links.Store
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		log.Printf("Connecting to postgres database...")
		pgStore, err := links.NewPostgresStore(dbURL)
		if err != nil {
			log.Fatalf("links.NewPostgresStore failed: %v", err)
		}
		store = pgStore
		defer pgStore.Close()
	} else {
		if *database != "" {
			if err := os.MkdirAll(*database, os.ModePerm); err != nil {
				log.Fatalf("os.MkdirAll(%v) failed: %v", *database, err)
			}
		}
		kv, err := links.NewKV(*database)
		if err != nil {
			log.Fatalf("links.NewKV(%v) failed: %v", *database, err)
		}
		store = links.NewKVStore(kv)
	}

	addr := fmt.Sprint(":", *port)
	log.Printf("listening on %q", addr)
	log.Fatal(http.ListenAndServe(addr, links.NewHandler(store, ks)))
}
