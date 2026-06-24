package gen

import (
	"fmt"
	"sort"
	"strings"
)

const weightEpsilon = 1e-6

var knownFeatures = map[string]struct{}{
	"queueing_delay_export": {},
}

func checkFeatures(pg *ParsedGraph, errs *[]string) {
	for name := range pg.Features {
		if _, ok := knownFeatures[name]; !ok {
			*errs = append(*errs, fmt.Sprintf("unknown feature %q", name))
		}
	}
}

func checkEdgeWeights(pg *ParsedGraph, errs *[]string) {
	for _, e := range pg.Edges {
		if e.Source == "USER" && e.Weight != nil {
			*errs = append(*errs, "weight on USER edge is not allowed")
			break
		}
	}
	entryAPIs := make([]string, 0, len(pg.EntryNodeIDs))
	for _, eid := range pg.EntryNodeIDs {
		entryAPIs = append(entryAPIs, pg.Nodes[eid].Interface)
	}
	for nodeID := range pg.Nodes {
		for _, apiName := range entryAPIs {
			edges := pg.OutgoingEdgesForAPI(nodeID, apiName)
			if len(edges) == 0 {
				continue
			}
			nWeighted := 0
			for _, e := range edges {
				if e.Weight != nil {
					nWeighted++
				}
			}
			if nWeighted == 0 {
				continue
			}
			if nWeighted != len(edges) {
				*errs = append(*errs, fmt.Sprintf("%s (api %q): mix of weighted and unweighted outgoing edges", nodeID, apiName))
				continue
			}
			var sum float64
			for _, e := range edges {
				w := *e.Weight
				if w <= 0 {
					*errs = append(*errs, fmt.Sprintf("edge %s→%s: weight must be > 0", e.Source, e.Target))
				}
				sum += w
			}
			if sum < 1-weightEpsilon || sum > 1+weightEpsilon {
				*errs = append(*errs, fmt.Sprintf("%s (api %q): weights sum to %g, want 1", nodeID, apiName, sum))
			}
		}
	}
}

func sortedEdgeTargets(edges []Edge) []string {
	t := make([]string, len(edges))
	for i, e := range edges {
		t[i] = e.Target
	}
	sort.Strings(t)
	return t
}

func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// checkFanOutAndParallel enforces parallel vs weighted vs sequential trichotomy and cross-API rules for parallel.
func checkFanOutAndParallel(pg *ParsedGraph, errs *[]string) {
	entryAPIs := make([]string, 0, len(pg.EntryNodeIDs))
	seen := make(map[string]bool)
	for _, eid := range pg.EntryNodeIDs {
		iface := pg.Nodes[eid].Interface
		if !seen[iface] {
			seen[iface] = true
			entryAPIs = append(entryAPIs, iface)
		}
	}

	for _, e := range pg.Edges {
		if e.Parallel && e.Weight != nil {
			*errs = append(*errs, fmt.Sprintf("edge %s→%s: parallel and weight are mutually exclusive", e.Source, e.Target))
		}
	}

	for nodeID := range pg.Nodes {
		for _, apiName := range entryAPIs {
			edges := pg.OutgoingEdgesForAPI(nodeID, apiName)
			if len(edges) < 2 {
				continue
			}
			nWeighted := 0
			nParallel := 0
			for _, e := range edges {
				if e.Weight != nil {
					nWeighted++
				}
				if e.Parallel {
					nParallel++
				}
			}
			if nWeighted == len(edges) {
				if nParallel > 0 {
					*errs = append(*errs, fmt.Sprintf("%s (api %q): parallel not allowed on weighted fan-out edges", nodeID, apiName))
				}
				continue
			}
			if nWeighted > 0 {
				continue
			}
			if nParallel > 0 && nParallel < len(edges) {
				*errs = append(*errs, fmt.Sprintf("%s (api %q): parallel must be true on every edge in the fan-out group or false/omitted on all", nodeID, apiName))
			}
		}
	}

	for nodeID := range pg.Nodes {
		apis := pg.APIsReachingNode(nodeID)
		var ref []string
		hasParallel := false
		for _, api := range apis {
			edges := pg.OutgoingEdgesForAPI(nodeID, api)
			if !IsParallelFanoutGroup(edges) {
				continue
			}
			st := sortedEdgeTargets(edges)
			if !hasParallel {
				ref = st
				hasParallel = true
				continue
			}
			if !stringSliceEqual(ref, st) {
				*errs = append(*errs, fmt.Sprintf("%s: parallel fan-out targets differ across entry APIs (sidecar mapping is one row per method)", nodeID))
				break
			}
		}
		if !hasParallel {
			continue
		}
		for _, api := range apis {
			edges := pg.OutgoingEdgesForAPI(nodeID, api)
			if IsParallelFanoutGroup(edges) {
				continue
			}
			*errs = append(*errs, fmt.Sprintf("%s (api %q): parallel fan-out from this node for another entry API requires the same parallel fan-out here", nodeID, api))
		}
	}
}

// Check validates connectivity and sanity of the parsed call graph.
func Check(pg *ParsedGraph) error {
	var errs []string

	// Sanity: basic structure
	if len(pg.Nodes) == 0 {
		errs = append(errs, "no nodes defined")
	}
	switch pg.LoadBalancingPolicy {
	case "weighted_round_robin", "round_robin", "least_request":
	default:
		errs = append(errs, fmt.Sprintf("load_balancing_policy %q: want weighted_round_robin, round_robin, or least_request", pg.LoadBalancingPolicy))
	}
	if pg.DagorQueuingThreshMs <= 0 {
		errs = append(errs, fmt.Sprintf("dagor.queuing_thresh_ms must be > 0, got %g", pg.DagorQueuingThreshMs))
	}
	if pg.DagorAlpha <= 0 {
		errs = append(errs, fmt.Sprintf("dagor.alpha must be > 0, got %g", pg.DagorAlpha))
	}
	if pg.DagorBeta <= 0 {
		errs = append(errs, fmt.Sprintf("dagor.beta must be > 0, got %g", pg.DagorBeta))
	}
	ocChecked := make(map[string]bool)
	for id, n := range pg.Nodes {
		if n.Interface == "" {
			errs = append(errs, fmt.Sprintf("node %s: empty interface name", id))
		}
		modes := 0
		if n.Bimodal {
			modes++
		}
		if n.Exponential {
			modes++
		}
		if !n.Bimodal && !n.Exponential {
			modes++
		}
		if modes != 1 {
			errs = append(errs, fmt.Sprintf("node %s: exactly one service-time mode required", id))
		}
		if n.Bimodal {
			if n.BimodalRT0 <= 0 || n.BimodalRT1 <= 0 {
				errs = append(errs, fmt.Sprintf("node %s: bimodal rt must be > 0", id))
			}
			p0, p1 := n.BimodalProb0, n.BimodalProb1
			if p0 <= weightEpsilon || p0 >= 1-weightEpsilon || p1 <= weightEpsilon || p1 >= 1-weightEpsilon {
				errs = append(errs, fmt.Sprintf("node %s: bimodal prob must be in (0,1)", id))
			}
			sum := p0 + p1
			if sum < 1-weightEpsilon || sum > 1+weightEpsilon {
				errs = append(errs, fmt.Sprintf("node %s: bimodal prob sum is %g, want 1", id, sum))
			}
		} else if n.Exponential {
			if n.ExponentialMean <= 0 {
				errs = append(errs, fmt.Sprintf("node %s: exponential mean must be > 0", id))
			}
		} else if n.AvgRT < 0 {
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

	svcChecked := make(map[string]bool)
	for _, n := range pg.Nodes {
		if svcChecked[n.Microservice] {
			continue
		}
		svcChecked[n.Microservice] = true
		if n.Replicas < 1 {
			errs = append(errs, fmt.Sprintf("service %s: replicas must be >= 1", n.Microservice))
		}
		if PerReplicaCPU(n.CPU, n.Replicas) <= 0 {
			errs = append(errs, fmt.Sprintf("service %s: cpu=%g too small for replicas=%d", n.Microservice, n.CPU, n.Replicas))
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

	if pg.EffectiveConnectionPoolSize() < 1 {
		errs = append(errs, "effective connection_pool_size must be >= 1")
	}

	checkFeatures(pg, &errs)
	checkEdgeWeights(pg, &errs)
	checkFanOutAndParallel(pg, &errs)

	if len(errs) > 0 {
		return fmt.Errorf("check failed:\n  %s", strings.Join(errs, "\n  "))
	}
	return nil
}
