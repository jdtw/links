package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"

	"google.golang.org/protobuf/encoding/protojson"
	"jdtw.dev/links/pkg/client"
	"jdtw.dev/links/pkg/frontend"
	"jdtw.dev/links/pkg/links"
	pb "jdtw.dev/links/proto/links"
	"jdtw.dev/token"
)

var (
	priv   = flag.String("priv", "", "Path to private key; can also be specified via the LINKS_PRIVATE_KEY environment variable.")
	addr   = flag.String("addr", "", "Appliction URI; can also be specified via the LINKS_ADDR environment variable")
	index  = flag.String("index", "", "Set the root redirect")
	add    = flag.String("add", "", "Add a redirect")
	link   = flag.String("link", "", "The redirect")
	get    = flag.String("get", "", "Get a redirect")
	rm     = flag.String("rm", "", "Remove a redirect")
	server = flag.Int("server", -1, "If not -1, starts starts a frontent HTTP server on the given port.")
	export = flag.String("export", "", "Write all links as a JSON Links proto to the given file, or '-' for stdout")
	imprt  = flag.String("import", "", "Bulk create or update links from a JSON Links proto file, or '-' for stdin")
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
	case *export != "":
		lpb, err := c.Export()
		if err != nil {
			log.Fatal(err)
		}
		// Indented so the backup is readable and reviewable. Note that
		// protojson does not promise byte-stable output, so don't expect
		// two exports of identical data to diff clean.
		data, err := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(lpb)
		if err != nil {
			log.Fatal(err)
		}
		if *export == "-" {
			os.Stdout.Write(data)
		} else if err := os.WriteFile(*export, data, 0600); err != nil {
			log.Fatal(err)
		}
		log.Printf("exported %d links", len(lpb.GetLinks()))
	case *imprt != "":
		var data []byte
		var err error
		if *imprt == "-" {
			data, err = io.ReadAll(os.Stdin)
		} else {
			data, err = os.ReadFile(*imprt)
		}
		if err != nil {
			log.Fatal(err)
		}
		lpb := &pb.Links{}
		if err := protojson.Unmarshal(data, lpb); err != nil {
			log.Fatalf("failed to parse %s: %v", *imprt, err)
		}
		if err := c.Import(lpb); err != nil {
			log.Fatal(err)
		}
		log.Printf("imported %d links", len(lpb.GetLinks()))
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
