package gen

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

const defaultAvgRT = 1.0
const busyLoopScale = 320

type CallGraph struct {
	Nodes []ServiceNode `json:"nodes"`
	Edges []Edge        `json:"edges"`
}

type ServiceNode struct {
	ID             string      `json:"id"`
	Interfaces     []Interface `json:"interfaces"`
	CPU            int         `json:"cpu"`
	SidecarCPU     int         `json:"sidecar_cpu"`
	OverCommitment float64     `json:"over_commitment"`
}

type Interface struct {
	Name     string  `json:"name"`
	AvgRT    float64 `json:"avg_rt"`
	SLO      *int    `json:"slo"`
	Priority *int    `json:"priority"`
}

type Node struct {
	ID             string
	Microservice   string
	Interface      string
	AvgRT          float64
	CPU            int
	SidecarCPU     int
	OverCommitment float64
	SLO            *int
	Priority       *int
}

type Edge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	API    string `json:"api"`
}

// EdgeVisible is true if this edge may be used when the request's entry API is apiName.
// Empty API on an edge means all APIs (legacy).
func EdgeVisible(e Edge, apiName string) bool {
	if e.Source == "USER" {
		return true
	}
	if e.API == "" {
		return true
	}
	return e.API == apiName
}

type ParsedGraph struct {
	Nodes        map[string]*Node
	Edges        []Edge
	EntryNodeIDs []string
	Services     map[string][]*Node
}

func ParseCallGraph(path string) (*ParsedGraph, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cg CallGraph
	if err := json.Unmarshal(data, &cg); err != nil {
		return nil, err
	}
	return buildParsedGraph(&cg)
}

func buildParsedGraph(cg *CallGraph) (*ParsedGraph, error) {
	pg := &ParsedGraph{
		Nodes:    make(map[string]*Node),
		Services: make(map[string][]*Node),
	}
	for i := range cg.Nodes {
		svc := &cg.Nodes[i]
		cpu := svc.CPU
		if cpu == 0 {
			cpu = 1
		}
		sidecarCPU := svc.SidecarCPU
		if sidecarCPU == 0 {
			sidecarCPU = 1
		}
		if svc.ID == "USER" {
			continue
		}
		for _, iface := range svc.Interfaces {
			avgRT := iface.AvgRT
			if avgRT == 0 {
				avgRT = defaultAvgRT
			}
			node := &Node{
				ID:             svc.ID + ":" + iface.Name,
				Microservice:   svc.ID,
				Interface:      iface.Name,
				AvgRT:          avgRT,
				CPU:            cpu,
				SidecarCPU:     sidecarCPU,
				OverCommitment: svc.OverCommitment,
				SLO:            iface.SLO,
				Priority:       iface.Priority,
			}
			pg.Nodes[node.ID] = node
			pg.Services[svc.ID] = append(pg.Services[svc.ID], node)
		}
	}
	for _, e := range cg.Edges {
		edge := e
		if e.Source == "USER" {
			tn, ok := pg.Nodes[e.Target]
			if !ok {
				return nil, fmt.Errorf("entry node %s not found", e.Target)
			}
			if edge.API == "" {
				edge.API = tn.Interface
			} else if edge.API != tn.Interface {
				return nil, fmt.Errorf("USER edge to %s: api %q must match entry interface %q", e.Target, edge.API, tn.Interface)
			}
			pg.EntryNodeIDs = append(pg.EntryNodeIDs, e.Target)
		}
		pg.Edges = append(pg.Edges, edge)
	}
	if len(pg.EntryNodeIDs) == 0 {
		return nil, fmt.Errorf("no USER entry point found")
	}
	// All USER targets must belong to the same service (frontend)
	entrySvc := pg.Nodes[pg.EntryNodeIDs[0]].Microservice
	for _, id := range pg.EntryNodeIDs {
		if pg.Nodes[id].Microservice != entrySvc {
			return nil, fmt.Errorf("all USER entry points must target same service; %s vs %s", pg.EntryNodeIDs[0], id)
		}
	}
	return pg, nil
}

// FullRPCName returns the gRPC full method name for sidecar config (without leading slash).
// Format: benchmark.{Microservice}/{ProtoMethodName}
func (n *Node) FullRPCName() string {
	return "benchmark." + n.Microservice + "/" + n.ProtoMethodName()
}

func (n *Node) BusyLoopRepeats() int {
	repeats := int(n.AvgRT * busyLoopScale)
	if repeats < 1 {
		repeats = 1
	}
	return repeats
}

func (n *Node) ProtoMethodName() string {
	s := strings.ReplaceAll(n.Interface, "-", "_")
	s = regexp.MustCompile(`[^a-zA-Z0-9_]`).ReplaceAllString(s, "_")
	if s == "" {
		return "Call"
	}
	if s[0] >= '0' && s[0] <= '9' {
		s = "M" + s
	}
	return s
}

func (n *Node) GoMethodName() string {
	s := n.ProtoMethodName()
	if s == "" {
		return "Call"
	}
	leadingUnderscore := strings.HasPrefix(s, "_")
	s = strings.TrimPrefix(s, "_")
	parts := strings.Split(s, "_")
	capParts := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			continue
		}
		runes := []rune(p)
		runes[0] = unicode.ToUpper(runes[0])
		for i := 1; i < len(runes); i++ {
			if unicode.IsDigit(runes[i-1]) && unicode.IsLower(runes[i]) {
				runes[i] = unicode.ToUpper(runes[i])
			}
		}
		capParts = append(capParts, string(runes))
	}
	merged := make([]string, 0, len(capParts))
	mergedFromSingle := make([]bool, 0, len(capParts))
	for i := 0; i < len(capParts); i++ {
		p := capParts[i]
		if len(p) == 1 && !unicode.IsDigit(rune(p[0])) && i+1 < len(capParts) {
			merged = append(merged, p+capParts[i+1])
			mergedFromSingle = append(mergedFromSingle, true)
			i++
		} else {
			merged = append(merged, p)
			mergedFromSingle = append(mergedFromSingle, false)
		}
	}
	var b strings.Builder
	if leadingUnderscore {
		b.WriteByte('X')
	}
	for i, p := range merged {
		if i > 0 {
			if mergedFromSingle[i] {
				b.WriteByte('_')
			} else if len(p) > 0 && unicode.IsDigit(rune(p[0])) {
				b.WriteByte('_')
			}
		}
		b.WriteString(p)
	}
	return b.String()
}

func (pg *ParsedGraph) Downstream(nodeID string) []string {
	var targets []string
	for _, e := range pg.Edges {
		if e.Source == nodeID {
			targets = append(targets, e.Target)
		}
	}
	return targets
}

// DownstreamForAPI returns targets of edges from nodeID visible for the given entry API name.
func (pg *ParsedGraph) DownstreamForAPI(nodeID string, apiName string) []string {
	var targets []string
	for _, e := range pg.Edges {
		if e.Source != nodeID {
			continue
		}
		if EdgeVisible(e, apiName) {
			targets = append(targets, e.Target)
		}
	}
	return targets
}

// ReachableFromWithAPI is BFS from entryID following only edges visible for that entry's API.
func (pg *ParsedGraph) ReachableFromWithAPI(entryID string) map[string]bool {
	apiName := pg.Nodes[entryID].Interface
	reachable := map[string]bool{entryID: true}
	queue := []string{entryID}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, t := range pg.DownstreamForAPI(cur, apiName) {
			if !reachable[t] {
				reachable[t] = true
				queue = append(queue, t)
			}
		}
	}
	return reachable
}

// APIsReachingNode returns sorted entry API names whose virtual graph includes nodeID.
func (pg *ParsedGraph) APIsReachingNode(nodeID string) []string {
	var out []string
	seen := make(map[string]bool)
	for _, eid := range pg.EntryNodeIDs {
		iface := pg.Nodes[eid].Interface
		if seen[iface] {
			continue
		}
		if pg.ReachableFromWithAPI(eid)[nodeID] {
			seen[iface] = true
			out = append(out, iface)
		}
	}
	sort.Strings(out)
	return out
}

func (pg *ParsedGraph) EntryMicroservice() string {
	return pg.Nodes[pg.EntryNodeIDs[0]].Microservice
}

// EntryInterfaces returns all entry nodes (APIs) that USER connects to.
func (pg *ParsedGraph) EntryInterfaces() []*Node {
	out := make([]*Node, 0, len(pg.EntryNodeIDs))
	for _, id := range pg.EntryNodeIDs {
		out = append(out, pg.Nodes[id])
	}
	return out
}

func (pg *ParsedGraph) CPUForService(svcName string) int {
	if nodes, ok := pg.Services[svcName]; ok && len(nodes) > 0 {
		return nodes[0].CPU
	}
	return 1
}

func (pg *ParsedGraph) SidecarCPUForService(svcName string) int {
	if nodes, ok := pg.Services[svcName]; ok && len(nodes) > 0 {
		return nodes[0].SidecarCPU
	}
	return 1
}

func (pg *ParsedGraph) OverCommitmentForService(svcName string) float64 {
	if nodes, ok := pg.Services[svcName]; ok && len(nodes) > 0 {
		return nodes[0].OverCommitment
	}
	return 0
}

// UserEntryCount returns the number of APIs (entry interfaces USER connects to).
func (pg *ParsedGraph) UserEntryCount() int {
	if len(pg.EntryNodeIDs) < 1 {
		return 1
	}
	return len(pg.EntryNodeIDs)
}
