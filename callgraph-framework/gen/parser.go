package gen

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode"
)

const defaultAvgRT = 1.0
const busyLoopScale = 320

type CallGraph struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

type Node struct {
	ID           string  `json:"id"`
	Microservice string  `json:"microservice"`
	Interface    string  `json:"interface"`
	AvgRT        float64 `json:"avg_rt"`
	CPU          int     `json:"cpu"`
}

type Edge struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

type ParsedGraph struct {
	Nodes       map[string]*Node
	Edges       []Edge
	EntryNodeID string
	Services    map[string][]*Node
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
		n := &cg.Nodes[i]
		if n.AvgRT == 0 {
			n.AvgRT = defaultAvgRT
		}
		if n.CPU == 0 {
			n.CPU = 1
		}
		if n.ID == "USER" {
			continue
		}
		pg.Nodes[n.ID] = n
		pg.Services[n.Microservice] = append(pg.Services[n.Microservice], n)
	}
	for _, e := range cg.Edges {
		if e.Source == "USER" {
			if pg.EntryNodeID != "" {
				return nil, fmt.Errorf("multiple USER entry points")
			}
			pg.EntryNodeID = e.Target
		}
		pg.Edges = append(pg.Edges, e)
	}
	if pg.EntryNodeID == "" {
		return nil, fmt.Errorf("no USER entry point found")
	}
	if _, ok := pg.Nodes[pg.EntryNodeID]; !ok {
		return nil, fmt.Errorf("entry node %s not found", pg.EntryNodeID)
	}
	return pg, nil
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

func (pg *ParsedGraph) EntryMicroservice() string {
	return pg.Nodes[pg.EntryNodeID].Microservice
}

func (pg *ParsedGraph) CPUForService(svcName string) int {
	if nodes, ok := pg.Services[svcName]; ok && len(nodes) > 0 {
		return nodes[0].CPU
	}
	return 1
}
