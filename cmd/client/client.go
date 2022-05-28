package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"

	"jdtw.dev/links/pkg/client"
	"jdtw.dev/links/pkg/frontend"
	"jdtw.dev/links/pkg/keybase"
	"jdtw.dev/links/pkg/links"
	"jdtw.dev/links/pkg/token"
)

var (
	priv       = flag.String("priv", "", "Path to private key; can also be specified via the LINKS_PRIVATE_KEY environment variable.")
	addr       = flag.String("addr", "", "Appliction URI; can also be specified via the LINKS_ADDR environment variable")
	index      = flag.String("index", "", "Set the root redirect")
	add        = flag.String("add", "", "Add a redirect")
	link       = flag.String("link", "", "The redirect")
	get        = flag.String("get", "", "Get a redirect")
	rm         = flag.String("rm", "", "Remove a redirect")
	server     = flag.Int("server", -1, "If not -1, starts starts a frontent HTTP server on the given port.")
	keybaseLoc = flag.String("keybase_loc", "", "If not empty, starts a keybase chat bot using the given keybase binary location.")
)

func main() {
	flag.Parse()

	if *addr == "" {
		*addr = os.Getenv("LINKS_ADDR")
	}
	if *addr == "" {
		log.Fatal("missing 'addr' flag.")
	}
	if *priv == "" {
		*priv = os.Getenv("LINKS_PRIVATE_KEY")
	}
	if *priv == "" {
		log.Fatal("missing 'priv' flag.")
	}

	privContents, err := os.ReadFile(*priv)
	if err != nil {
		log.Fatalf("ReadFile(%s) failed: %v", *priv, err)
	}
	signer, err := token.UnmarshalSigningKey(privContents)
	if err != nil {
		log.Fatalf("UnmarshalSigningKey failed: %v", err)
	}

	c := client.New(*addr, signer)
	switch {
	case *server != -1:
		addr := fmt.Sprint(":", *server)
		log.Printf("listening on %q", addr)
		log.Fatal(http.ListenAndServe(addr, frontend.NewHandler(c)))
	case *keybaseLoc != "":
		log.Fatal(keybase.ChatBot(*keybaseLoc, c))
	case *index != "":
		if err := c.Put(links.Index, *index); err != nil {
			log.Fatal(err)
		}
	case *add != "":
		if *link == "" {
			log.Fatal("missing 'link' flag")
		}
		if err := c.Put(*add, *link); err != nil {
			log.Fatal(err)
		}
	case *get != "":
		redir, err := c.Get(*get)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(redir)
	case *rm != "":
		if err := c.Delete(*rm); err != nil {
			log.Fatal(err)
		}
	default:
		l, err := c.List()
		if err != nil {
			log.Fatal(err)
		}
		keys := make([]string, 0, len(l))
		for k := range l {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Printf("%s\t%s\n", k, l[k])
		}
	}
}
