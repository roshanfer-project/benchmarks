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
	ocChecked := make(map[string]bool)
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
		if !ocChecked[n.Microservice] {
			ocChecked[n.Microservice] = true
			if n.OverCommitment < 0 || n.OverCommitment > 1 {
				errs = append(errs, fmt.Sprintf("service %s: over_commitment must be between 0 and 1", n.Microservice))
			}
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

	entryIface := make(map[string]bool)
	for _, entryID := range pg.EntryNodeIDs {
		entryIface[pg.Nodes[entryID].Interface] = true
	}
	for _, e := range pg.Edges {
		if e.Source == "USER" {
			continue
		}
		if e.API != "" && !entryIface[e.API] {
			errs = append(errs, fmt.Sprintf("edge %s→%s: unknown api %q (not an entry interface)", e.Source, e.Target, e.API))
		}
	}

	// Connectivity: all nodes reachable from at least one entry (per-api slices)
	reachable := make(map[string]bool)
	for _, entryID := range pg.EntryNodeIDs {
		for id := range pg.ReachableFromWithAPI(entryID) {
			reachable[id] = true
		}
	}
	for id := range pg.Nodes {
		if !reachable[id] {
			errs = append(errs, fmt.Sprintf("node %s unreachable from entry", id))
		}
	}

	// Entry interfaces (APIs) must have slo and priority for sidecar mode
	for _, entryID := range pg.EntryNodeIDs {
		n := pg.Nodes[entryID]
		if n.SLO == nil || n.Priority == nil {
			errs = append(errs, fmt.Sprintf("entry interface %s must have slo and priority (required for sidecar mode)", entryID))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("check failed:\n  %s", strings.Join(errs, "\n  "))
	}
	return nil
}
