package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"

	"callgraph-framework/gen"
)

func main() {
	out := flag.String("o", "", "output directory for generated benchmark")
	flag.Parse()
	if flag.NArg() < 1 {
		log.Fatal("usage: gen <callgraph.json> [-o output-dir]")
	}
	callgraphPath := flag.Arg(0)
	if *out == "" {
		*out = filepath.Dir(callgraphPath)
	}
	benchmarkName := filepath.Base(*out)
	if err := gen.Generate(callgraphPath, *out, benchmarkName); err != nil {
		log.Fatalf("generate: %v", err)
	}
	fmt.Printf("Generated benchmark in %s\n", *out)
}
