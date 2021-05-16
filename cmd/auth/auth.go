package main

import (
	"crypto/rand"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jdtw/links/pkg/auth"
	"github.com/lestrrat-go/jwx/jwk"
)

var (
	new     = flag.Bool("new", false, "Create the keyset")
	keyset  = flag.String("keyset", "", "Path to the keyset")
	priv    = flag.String("priv", "", "Private key output location")
	subject = flag.String("subject", "", "Subject for the key")
)

func main() {
	flag.Parse()

	switch {
	case *subject == "":
		log.Fatal("missing 'subject' flag")
	case *keyset == "":
		log.Fatal("missing 'keyset' flag")
	}

	var ks jwk.Set
	if *new {
		ks = jwk.NewSet()
	} else {
		bs, err := os.ReadFile(*keyset)
		if err != nil {
			log.Fatalf("os.Readfile(%s) failed: %v", *keyset, err)
		}
		ks, err = jwk.Parse(bs)
		if err != nil {
			log.Fatal(err)
		}
	}

	pub, pkcs8, err := auth.NewKey(rand.Reader, *subject)
	if err != nil {
		log.Fatal(err)
	}

	ks.Add(pub)
	bs, err := json.Marshal(ks)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(*keyset, bs, os.ModePerm); err != nil {
		log.Fatalf("os.WriteFile(%s) failed: %v", *keyset, err)
	}

	pem := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: pkcs8,
	})
	if *priv == "" {
		fmt.Print(string(pem))
	} else {
		os.WriteFile(*priv, pem, os.ModePerm)
	}
}
