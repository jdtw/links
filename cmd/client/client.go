package main

import (
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/jdtw/links/pkg/client"
	"github.com/jdtw/links/pkg/links"
)

var (
	priv  = flag.String("priv", "", "Path to private key; can also be specified via the LINKS_PRIVATE_KEY environment variable.")
	addr  = flag.String("addr", "", "Appliction URI; can also be specified via the LINKS_ADDR environment variable")
	index = flag.String("index", "", "Set the root redirect")
	add   = flag.String("add", "", "Add a redirect")
	link  = flag.String("link", "", "The redirect")
	get   = flag.String("get", "", "Get a redirect")
	rm    = flag.String("rm", "", "Remove a redirect")
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

	var pkcs8 []byte
	if *priv != "" {
		bs, err := os.ReadFile(*priv)
		if err != nil {
			log.Fatal(err)
		}
		block, _ := pem.Decode(bs)
		if block.Type != "PRIVATE KEY" {
			log.Fatalf("unexpected PEM block type: %s", block.Type)
		}
		pkcs8 = block.Bytes
	}

	c := client.New(*addr, pkcs8)
	switch {
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
