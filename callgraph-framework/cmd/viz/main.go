package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"

	"callgraph-framework/viz"
)

func main() {
	out := flag.String("o", "", "output PDF path")
	flag.Parse()
	if flag.NArg() < 1 {
		log.Fatal("usage: viz <callgraph.json> [-o callgraph.pdf]")
	}
	callgraphPath := flag.Arg(0)
	if *out == "" {
		*out = filepath.Join(filepath.Dir(callgraphPath), "callgraph.pdf")
	}
	if err := viz.Visualize(callgraphPath, *out); err != nil {
		log.Fatalf("viz: %v", err)
	}
	fmt.Printf("Generated %s\n", *out)
}
