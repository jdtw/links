package main

import (
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/jdtw/links/pkg/client"
)

var (
	priv = flag.String("priv", "", "Path to private key")
	addr = flag.String("addr", "https://jdtw.us", "Appliction URI")
	add  = flag.String("add", "", "Add a redirect")
	to   = flag.String("to", "", "The redirect")
	get  = flag.String("get", "", "Get a redirect")
	rm   = flag.String("rm", "", "Remove a redirect")
)

func main() {
	flag.Parse()

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
	case *add != "":
		if *to == "" {
			log.Fatal("missing 'to' flag")
		}
		if err := c.Put(*add, *to); err != nil {
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
