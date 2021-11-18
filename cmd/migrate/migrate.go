package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jdtw/links/pkg/client"
	"github.com/jdtw/links/pkg/token"
)

var (
	priv   = flag.String("priv", "", "Path to private key.")
	addr   = flag.String("addr", "https://jdtw.us", "Application URI.")
	dryRun = flag.Bool("dry_run", false, "If true, no changes are made.")
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
		log.Fatal(err)
	}
	signer, err := token.UnmarshalSigningKey(privContents)
	if err != nil {
		log.Fatal(err)
	}

	c := client.New(*addr, signer)
	links, err := c.List()
	if err != nil {
		log.Fatal(err)
	}

	for link, uri := range links {
		if !strings.ContainsRune(link, '-') {
			continue
		}
		newLink := strings.ReplaceAll(link, "-", "")
		fmt.Printf("%s\t%s\t%s\n", link, newLink, uri)
		if *dryRun {
			continue
		}
		if err := c.Put(newLink, uri); err != nil {
			log.Printf("Put(%s, %s) failed: %v", newLink, uri, err)
			continue
		}
		if err := c.Delete(link); err != nil {
			log.Printf("Delete(%s) failed: %v", link, err)
		}
	}
}
