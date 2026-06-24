package gen

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

const busyLoopScale = 320
const defaultConnectionPoolSize = 200

type DagorConfig struct {
	QueuingThreshMs *float64 `json:"queuing_thresh_ms,omitempty"`
	Alpha           *float64 `json:"alpha,omitempty"`
	Beta            *float64 `json:"beta,omitempty"`
}

type CallGraph struct {
	LoadBalancingPolicy string        `json:"load_balancing_policy,omitempty"`
	Dagor               *DagorConfig  `json:"dagor,omitempty"`
	Nodes               []ServiceNode `json:"nodes"`
	Edges               []Edge        `json:"edges"`
}

type ServiceNode struct {
	ID                 string      `json:"id"`
	Interfaces         []Interface `json:"interfaces"`
	CPU                float64     `json:"cpu"`
	Replicas           int         `json:"replicas"`
	SidecarCPU         int         `json:"sidecar_cpu"`
	OverCommitment     float64     `json:"over_commitment"`
	ConnectionPoolSize int         `json:"connection_pool_size,omitempty"`
}

// BimodalSpec is two-component service time: BusyLoop duration is rt[i] with probability prob[i].
type BimodalSpec struct {
	RT   []float64 `json:"rt"`
	Prob []float64 `json:"prob"`
}

// ExponentialSpec is exponential service time with mean in the same units as avg_rt.
type ExponentialSpec struct {
	Mean float64 `json:"mean"`
}

type Interface struct {
	Name        string           `json:"name"`
	AvgRT       *float64         `json:"avg_rt,omitempty"`
	Bimodal     *BimodalSpec     `json:"bimodal,omitempty"`
	Exponential *ExponentialSpec `json:"exponential,omitempty"`
	SLO         *int             `json:"slo"`
	Priority    *int             `json:"priority"`
}

type Node struct {
	ID              string
	Microservice    string
	Interface       string
	AvgRT           float64
	Bimodal         bool
	BimodalP0       float64
	BimodalR0       int
	BimodalR1       int
	BimodalRT0      float64
	BimodalRT1      float64
	BimodalProb0    float64
	BimodalProb1    float64
	Exponential     bool
	ExponentialMean float64
	CPU             float64
	Replicas        int
	SidecarCPU      int
	OverCommitment  float64
	SLO             *int
	Priority        *int
}

type Edge struct {
	Source   string   `json:"source"`
	Target   string   `json:"target"`
	API      string   `json:"api"`
	Weight   *float64 `json:"weight,omitempty"`
	Parallel bool     `json:"parallel,omitempty"`
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
	Nodes                map[string]*Node
	Edges                []Edge
	EntryNodeIDs         []string
	Services             map[string][]*Node
	ConnectionPoolSize   int // 0: use defaultConnectionPoolSize
	LoadBalancingPolicy  string
	DagorQueuingThreshMs float64
	DagorAlpha           float64
	DagorBeta            float64
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

func serviceTimeModes(iface Interface) int {
	n := 0
	if iface.AvgRT != nil {
		n++
	}
	if iface.Bimodal != nil {
		n++
	}
	if iface.Exponential != nil {
		n++
	}
	return n
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
		replicas := svc.Replicas
		if replicas == 0 {
			replicas = 1
		}
		sidecarCPU := svc.SidecarCPU
		if sidecarCPU == 0 {
			sidecarCPU = 1
		}
		if svc.ID == "USER" {
			continue
		}
		for _, iface := range svc.Interfaces {
			id := svc.ID + ":" + iface.Name
			switch serviceTimeModes(iface) {
			case 0:
				return nil, fmt.Errorf("node %s: must set exactly one of avg_rt, bimodal, exponential", id)
			case 1:
			default:
				return nil, fmt.Errorf("node %s: set only one of avg_rt, bimodal, exponential", id)
			}
			node := &Node{
				ID:             id,
				Microservice:   svc.ID,
				Interface:      iface.Name,
				CPU:            cpu,
				Replicas:       replicas,
				SidecarCPU:     sidecarCPU,
				OverCommitment: svc.OverCommitment,
				SLO:            iface.SLO,
				Priority:       iface.Priority,
			}
			if iface.Bimodal != nil {
				b := iface.Bimodal
				node.Bimodal = true
				if len(b.RT) != 2 || len(b.Prob) != 2 {
					return nil, fmt.Errorf("node %s: bimodal requires rt and prob of length 2", id)
				}
				node.BimodalP0 = b.Prob[0]
				node.BimodalRT0, node.BimodalRT1 = b.RT[0], b.RT[1]
				node.BimodalProb0, node.BimodalProb1 = b.Prob[0], b.Prob[1]
				node.BimodalR0 = repeatsFromServiceTime(b.RT[0])
				node.BimodalR1 = repeatsFromServiceTime(b.RT[1])
			} else if iface.Exponential != nil {
				node.Exponential = true
				node.ExponentialMean = iface.Exponential.Mean
			} else {
				node.AvgRT = *iface.AvgRT
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
	for i := range cg.Nodes {
		svc := &cg.Nodes[i]
		if svc.ID == "USER" || svc.ID == entrySvc {
			continue
		}
		if svc.ConnectionPoolSize != 0 {
			return nil, fmt.Errorf("connection_pool_size is only valid on entry service %q, not %q", entrySvc, svc.ID)
		}
	}
	for i := range cg.Nodes {
		if cg.Nodes[i].ID != entrySvc {
			continue
		}
		p := cg.Nodes[i].ConnectionPoolSize
		if p < 0 {
			return nil, fmt.Errorf("service %s: connection_pool_size must be >= 1", entrySvc)
		}
		if p > 0 {
			pg.ConnectionPoolSize = p
		}
		break
	}
	policy := strings.TrimSpace(cg.LoadBalancingPolicy)
	if policy == "" {
		policy = "least_request"
	}
	pg.LoadBalancingPolicy = policy
	pg.DagorQueuingThreshMs = 2
	pg.DagorAlpha = 0.45
	pg.DagorBeta = 0.01
	if cg.Dagor != nil {
		if cg.Dagor.QueuingThreshMs != nil {
			pg.DagorQueuingThreshMs = *cg.Dagor.QueuingThreshMs
		}
		if cg.Dagor.Alpha != nil {
			pg.DagorAlpha = *cg.Dagor.Alpha
		}
		if cg.Dagor.Beta != nil {
			pg.DagorBeta = *cg.Dagor.Beta
		}
	}
	return pg, nil
}

// EffectiveConnectionPoolSize returns connection_pool_size from the entry service, or the default if unset.
func (pg *ParsedGraph) EffectiveConnectionPoolSize() int {
	if pg.ConnectionPoolSize > 0 {
		return pg.ConnectionPoolSize
	}
	return defaultConnectionPoolSize
}

// FullRPCName returns the gRPC full method name for sidecar config (without leading slash).
// Format: benchmark.{Microservice}/{ProtoMethodName}
func (n *Node) FullRPCName() string {
	return "benchmark." + n.Microservice + "/" + n.ProtoMethodName()
}

func repeatsFromServiceTime(rt float64) int {
	repeats := int(rt * busyLoopScale)
	if repeats < 1 {
		repeats = 1
	}
	return repeats
}

// BusyLoopRepeats is only valid for fixed avg_rt nodes.
func (n *Node) BusyLoopRepeats() int {
	if n.Bimodal || n.Exponential {
		panic("BusyLoopRepeats called on random service-time node")
	}
	return repeatsFromServiceTime(n.AvgRT)
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

// OutgoingEdgesForAPI returns edges from nodeID visible for the given entry API (JSON order).
func (pg *ParsedGraph) OutgoingEdgesForAPI(nodeID string, apiName string) []Edge {
	var out []Edge
	for _, e := range pg.Edges {
		if e.Source != nodeID {
			continue
		}
		if EdgeVisible(e, apiName) {
			out = append(out, e)
		}
	}
	return out
}

// IsParallelFanoutGroup is true when len >= 2, all edges unweighted, and every edge has Parallel set.
func IsParallelFanoutGroup(edges []Edge) bool {
	if len(edges) < 2 {
		return false
	}
	for _, e := range edges {
		if e.Weight != nil || !e.Parallel {
			return false
		}
	}
	return true
}

// NodeUsesParallelFanout is true if some entry API sees a parallel fan-out from nodeID.
func (pg *ParsedGraph) NodeUsesParallelFanout(nodeID string) bool {
	for _, api := range pg.APIsReachingNode(nodeID) {
		if IsParallelFanoutGroup(pg.OutgoingEdgesForAPI(nodeID, api)) {
			return true
		}
	}
	return false
}

// IsWeightedFanoutGroup is true when len >= 2 and every edge has a weight.
func IsWeightedFanoutGroup(edges []Edge) bool {
	if len(edges) < 2 {
		return false
	}
	for _, e := range edges {
		if e.Weight == nil {
			return false
		}
	}
	return true
}

// NodeUsesWeightedFanout is true if some entry API sees a weighted fan-out from nodeID.
func (pg *ParsedGraph) NodeUsesWeightedFanout(nodeID string) bool {
	for _, api := range pg.APIsReachingNode(nodeID) {
		if IsWeightedFanoutGroup(pg.OutgoingEdgesForAPI(nodeID, api)) {
			return true
		}
	}
	return false
}

// DownstreamForAPI returns targets of edges from nodeID visible for the given entry API name.
func (pg *ParsedGraph) DownstreamForAPI(nodeID string, apiName string) []string {
	edges := pg.OutgoingEdgesForAPI(nodeID, apiName)
	out := make([]string, len(edges))
	for i, e := range edges {
		out[i] = e.Target
	}
	return out
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

func (pg *ParsedGraph) CPUForService(svcName string) float64 {
	if nodes, ok := pg.Services[svcName]; ok && len(nodes) > 0 {
		return nodes[0].CPU
	}
	return 1
}

func (pg *ParsedGraph) ReplicasForService(svcName string) int {
	if nodes, ok := pg.Services[svcName]; ok && len(nodes) > 0 {
		return nodes[0].Replicas
	}
	return 1
}

// PerReplicaCPU returns cpu/replicas for plain-lb and dagor-lb pod resources.
func PerReplicaCPU(cpu float64, replicas int) float64 {
	return cpu / float64(replicas)
}

func FormatK8sCPU(f float64) string {
	return strconv.FormatFloat(f, 'g', -1, 64)
}

func GOMAXPROCSForPerReplicaCPU(f float64) int {
	return int(math.Ceil(f))
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
