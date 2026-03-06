package gen

import (
	"fmt"
	"strings"
)

// Check validates connectivity and sanity of the parsed call graph.
func Check(pg *ParsedGraph) error {
	var errs []string

	// Sanity: basic structure
	if len(pg.Nodes) == 0 {
		errs = append(errs, "no nodes defined")
	}
	for id, n := range pg.Nodes {
		if n.Interface == "" {
			errs = append(errs, fmt.Sprintf("node %s: empty interface name", id))
		}
		if n.AvgRT < 0 {
			errs = append(errs, fmt.Sprintf("node %s: negative avg_rt", id))
		}
		if n.CPU <= 0 {
			errs = append(errs, fmt.Sprintf("node %s: cpu must be > 0", id))
		}
	}

	// Edges: all targets and non-USER sources must exist
	for _, e := range pg.Edges {
		if e.Source != "USER" {
			if _, ok := pg.Nodes[e.Source]; !ok {
				errs = append(errs, fmt.Sprintf("edge source %s not found", e.Source))
			}
		}
		if _, ok := pg.Nodes[e.Target]; !ok {
			errs = append(errs, fmt.Sprintf("edge target %s not found", e.Target))
		}
	}

	// Connectivity: all nodes reachable from entry
	reachable := reachableFrom(pg, pg.EntryNodeID)
	for id := range pg.Nodes {
		if !reachable[id] {
			errs = append(errs, fmt.Sprintf("node %s unreachable from entry", id))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("check failed:\n  %s", strings.Join(errs, "\n  "))
	}
	return nil
}

func reachableFrom(pg *ParsedGraph, start string) map[string]bool {
	reachable := map[string]bool{start: true}
	queue := []string{start}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, t := range pg.Downstream(cur) {
			if !reachable[t] {
				reachable[t] = true
				queue = append(queue, t)
			}
		}
	}
	return reachable
}
