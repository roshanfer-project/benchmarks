package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"

	"callgraph-framework/gen"
	"callgraph-framework/viz"
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
	pg, err := gen.ParseCallGraph(callgraphPath)
	if err != nil {
		log.Fatalf("parse: %v", err)
	}
	if err := gen.Generate(callgraphPath, *out, benchmarkName); err != nil {
		log.Fatalf("generate: %v", err)
	}
	callgraphOut := filepath.Join(*out, "callgraph.json")
	pdfOut := filepath.Join(*out, "callgraph.pdf")
	if err := viz.Visualize(callgraphOut, pdfOut); err != nil {
		log.Fatalf("viz: %v", err)
	}
	if err := gen.WriteModeComparison(pg, *out); err != nil {
		log.Fatalf("mode comparison: %v", err)
	}
	fmt.Printf("Generated benchmark in %s (callgraph.pdf, mode-comparison.md, mode-comparison.csv)\n", *out)
}
