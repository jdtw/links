package main

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"jdtw.dev/links/pkg/links"
	"jdtw.dev/token"
)

var (
	ephemeral = flag.Bool("ephemeral", false, "If true, don't connect to DATABASE_URL and use in-memory storage")
)

func main() {
	flag.Parse()
	log.SetPrefix("links: ")

	port := 8080
	if env := os.Getenv("PORT"); env != "" {
		parsed, err := strconv.Atoi(env)
		if err != nil {
			log.Fatalf("failed to parse PORT %q", env)
		}
		port = parsed
	}

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
		ctx := context.Background()
		pgStore, err := links.NewPostgresStore(ctx, os.Getenv("DATABASE_URL"))
		if err != nil {
			log.Fatalf("links.NewPostgresStore failed: %v", err)
		}
		store = pgStore
		defer pgStore.Close()
	}

	skew := time.Duration(0)
	if val := os.Getenv("SKEW"); val != "" {
		d, err := time.ParseDuration(val)
		if err != nil {
			log.Fatalf("failed to parse %q as a time.Duration: %v", val, err)
		}
		skew = d
		log.Printf("Allowing auth skew of %s", skew)
	}

	addr := fmt.Sprint(":", port)
	log.Printf("listening on %q", addr)
	log.Fatal(http.ListenAndServe(addr, links.NewHandler(store, keyset, skew)))
}
