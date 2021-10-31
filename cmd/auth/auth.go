package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jdtw/links/pkg/token"
	pb "github.com/jdtw/links/proto/token"
	"google.golang.org/protobuf/proto"
)

var (
	new     = flag.Bool("new", false, "Create the keyset")
	keyset  = flag.String("keyset", "", "Path to the keyset")
	priv    = flag.String("priv", "", "Private key output location")
	subject = flag.String("subject", "", "Subject for the key")
	dump    = flag.Bool("dump", false, "If true, prints the keyset and/or key as JSON")
)

func main() {
	flag.Parse()

	if *keyset == "" {
		*keyset = os.Getenv("LINKS_KEYSET")
	}

	if *dump && *keyset != "" {
		bs, err := os.ReadFile(*keyset)
		if err != nil {
			log.Fatal(err)
		}
		kspb := &pb.VerificationKeyset{}
		if err := proto.Unmarshal(bs, kspb); err != nil {
			log.Fatal(err)
		}
		dumped, err := json.Marshal(kspb)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s\n", dumped)
		return
	}

	switch {
	case *keyset == "":
		log.Fatal("missing 'keyset' flag")
	case *subject == "":
		log.Fatal("missing 'subject' flag")
	case *priv == "":
		log.Fatal("missing 'priv' flag")
	}

	var ks *token.VerificationKeyset
	if *new {
		ks = token.NewVerificationKeyset()
	} else {
		bs, err := os.ReadFile(*keyset)
		if err != nil {
			log.Fatalf("os.Readfile(%s) failed: %v", *keyset, err)
		}
		ks, err = token.UnmarshalVerificationKeyset(bs)
		if err != nil {
			log.Fatal(err)
		}
	}

	signer, err := token.NewSigningKey()
	if err != nil {
		log.Fatal(err)
	}

	if err := ks.AddKey(signer.ID(), *subject, signer.Public()); err != nil {
		log.Fatal(err)
	}

	bs, err := ks.Marshal()
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(*keyset, bs, os.ModePerm); err != nil {
		log.Fatalf("os.WriteFile(%s) failed: %v", *keyset, err)
	}

	bs, err = signer.Marshal()
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(*priv, bs, os.ModePerm); err != nil {
		log.Fatalf("os.WriteFile(%s) failed: %v", *priv, err)
	}
}
