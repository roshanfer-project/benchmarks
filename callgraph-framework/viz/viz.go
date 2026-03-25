package viz

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"callgraph-framework/gen"
)

func dotID(s string) string {
	s = strings.ReplaceAll(s, ":", "_")
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, ".", "_")
	return s
}

func dotLabel(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}

func reachableEdgesFrom(pg *gen.ParsedGraph, entryID string) []gen.Edge {
	apiName := pg.Nodes[entryID].Interface
	reach := pg.ReachableFromWithAPI(entryID)
	var edges []gen.Edge
	for _, e := range pg.Edges {
		if e.Source == "USER" {
			continue
		}
		if !reach[e.Source] || !reach[e.Target] {
			continue
		}
		if !gen.EdgeVisible(e, apiName) {
			continue
		}
		edges = append(edges, e)
	}
	return edges
}

func nodeLabel(n *gen.Node, isEntry bool) string {
	parts := []string{n.Interface, fmt.Sprintf("rt:%.2g", n.AvgRT)}
	if n.SLO != nil {
		parts = append(parts, fmt.Sprintf("slo:%d", *n.SLO))
	}
	if isEntry && n.Priority != nil {
		parts = append(parts, fmt.Sprintf("prio:%d", *n.Priority))
	}
	return strings.Join(parts, "\n")
}

func Visualize(callgraphPath string, outPath string) error {
	pg, err := gen.ParseCallGraph(callgraphPath)
	if err != nil {
		return fmt.Errorf("parse callgraph: %w", err)
	}

	var b strings.Builder
	b.WriteString("digraph G {\n")
	b.WriteString("  rankdir=LR;\n")
	// splines=true (not curved): edge labels sit on the visible path more reliably
	b.WriteString("  splines=true;\n")
	b.WriteString("  nodesep=0.5;\n")
	b.WriteString("  ranksep=0.8;\n")
	b.WriteString("  fontname=\"Helvetica\";\n")
	b.WriteString("  node [shape=box, style=\"rounded,filled\", fontname=\"Helvetica\", fontsize=10, margin=0.2, fillcolor=\"#f8f9fa\", color=\"#495057\"];\n")
	b.WriteString("  edge [fontname=\"Helvetica\", fontsize=9, color=\"#6c757d\"];\n")

	b.WriteString("  USER [label=" + dotLabel("USER") + ", shape=ellipse, fillcolor=\"#e9ecef\", penwidth=2];\n")

	entryNodes := make(map[string]bool)
	for _, e := range pg.Edges {
		if e.Source == "USER" {
			entryNodes[e.Target] = true
		}
	}

	svcNames := make([]string, 0, len(pg.Services))
	for name := range pg.Services {
		svcNames = append(svcNames, name)
	}
	sort.Strings(svcNames)

	for _, svc := range svcNames {
		nodes := pg.Services[svc]
		cpu := pg.CPUForService(svc)
		oc := pg.OverCommitmentForService(svc)
		clusterLabel := fmt.Sprintf("%s\ncpu:%d\nover_commit:%g", svc, cpu, oc)
		b.WriteString("  subgraph cluster_" + dotID(svc) + " {\n")
		b.WriteString("    label=" + dotLabel(clusterLabel) + ";\n")
		b.WriteString("    style=rounded;\n")
		b.WriteString("    fillcolor=\"#f1f3f5\";\n")
		b.WriteString("    penwidth=1;\n")
		b.WriteString("    fontname=\"Helvetica\";\n")
		b.WriteString("    fontsize=10;\n")
		for _, n := range nodes {
			attr := ""
			if entryNodes[n.ID] {
				attr = ", fillcolor=\"#dee2e6\""
			}
			b.WriteString("    " + dotID(n.ID) + " [label=" + dotLabel(nodeLabel(n, entryNodes[n.ID])) + attr + "];\n")
		}
		b.WriteString("  }\n")
	}

	// Color edges by API: compute reachable edges per entry, assign colors
	apiColors := []string{"#2e86ab", "#a23b72", "#f18f01", "#c73e1d", "#3b1f2b", "#95c623"}
	edgeToAPI := make(map[string]string) // "src->tgt" -> api (first API that reaches it)
	for _, entryID := range pg.EntryNodeIDs {
		api := pg.Nodes[entryID].Interface
		// Include USER -> entry edge (reachableEdgesFrom only goes downstream)
		userEdgeKey := "USER->" + entryID
		if _, ok := edgeToAPI[userEdgeKey]; !ok {
			edgeToAPI[userEdgeKey] = api
		}
		reachable := reachableEdgesFrom(pg, entryID)
		for _, e := range reachable {
			key := e.Source + "->" + e.Target
			if _, ok := edgeToAPI[key]; !ok {
				edgeToAPI[key] = api
			}
		}
	}
	apiToColor := make(map[string]string)
	for i, entryID := range pg.EntryNodeIDs {
		api := pg.Nodes[entryID].Interface
		apiToColor[api] = apiColors[i%len(apiColors)]
	}
	for _, e := range pg.Edges {
		key := e.Source + "->" + e.Target
		colorKey := ""
		if e.Source == "USER" {
			colorKey = pg.Nodes[e.Target].Interface
		} else if e.API != "" {
			colorKey = e.API
		} else if api, ok := edgeToAPI[key]; ok {
			colorKey = api
		}
		var parts []string
		var edgeHex string
		if colorKey != "" {
			if c, ok := apiToColor[colorKey]; ok {
				parts = append(parts, fmt.Sprintf("color=%q", c))
				edgeHex = c
			}
		}
		if e.Weight != nil {
			lbl := fmt.Sprintf("p=%.2f", *e.Weight)
			parts = append(parts, "label="+dotLabel(lbl))
			parts = append(parts, "fontsize=10")
			if edgeHex != "" {
				parts = append(parts, fmt.Sprintf("fontcolor=%q", edgeHex))
			}
		}
		attr := ""
		if len(parts) > 0 {
			attr = " [" + strings.Join(parts, ", ") + "]"
		}
		b.WriteString("  " + dotID(e.Source) + " -> " + dotID(e.Target) + attr + ";\n")
	}
	b.WriteString("}\n")

	tmpDir := filepath.Dir(outPath)
	tmpFile, err := os.CreateTemp(tmpDir, "callgraph-*.dot")
	if err != nil {
		return fmt.Errorf("create temp dot: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.WriteString(b.String()); err != nil {
		tmpFile.Close()
		return fmt.Errorf("write dot: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close dot: %w", err)
	}

	cmd := exec.Command("dot", "-Tpdf", "-o", outPath, tmpPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("dot failed (install graphviz: apt install graphviz / brew install graphviz): %w\n%s", err, out)
	}
	return nil
}
