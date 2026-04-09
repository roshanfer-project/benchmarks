package gen

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

// EntryGrpcK8s is the pod/app name for the gRPC entry (rajomon mode).
func EntryGrpcK8s(pg *ParsedGraph) string {
	return k8sName(pg.EntryMicroservice()) + "-grpc"
}

// RajomonCallgraphForEntry builds msgraph callgraph keys (entry MS aliased to EntryGrpcK8s).
func RajomonCallgraphForEntry(pg *ParsedGraph, entryID string) map[string][]string {
	entryMS := pg.EntryMicroservice()
	ek := EntryGrpcK8s(pg)
	alias := func(ms string) string {
		if ms == entryMS {
			return ek
		}
		return ms
	}
	apiName := pg.Nodes[entryID].Interface
	reach := pg.ReachableFromWithAPI(entryID)
	keySet := make(map[string]bool)
	for nid := range reach {
		keySet[alias(pg.Nodes[nid].Microservice)] = true
	}
	var keys []string
	for k := range keySet {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make(map[string][]string)
	for _, K := range keys {
		seen := make(map[string]bool)
		var down []string
		for nid := range reach {
			n := pg.Nodes[nid]
			if alias(n.Microservice) != K {
				continue
			}
			for _, tid := range pg.DownstreamForAPI(nid, apiName) {
				if !reach[tid] {
					continue
				}
				dk := alias(pg.Nodes[tid].Microservice)
				if !seen[dk] {
					seen[dk] = true
					down = append(down, dk)
				}
			}
		}
		sort.Strings(down)
		out[K] = down
	}
	return out
}

func formatRajomonApplicationYAML(cg map[string][]string, entrypoint, iface string) string {
	var keys []string
	for k := range cg {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString("- callgraph:\n")
	for _, k := range keys {
		downs := cg[k]
		if len(downs) == 0 {
			b.WriteString(fmt.Sprintf("    %s: []\n", k))
			continue
		}
		b.WriteString(fmt.Sprintf("    %s:\n", k))
		for _, d := range downs {
			b.WriteString(fmt.Sprintf("    - %s\n", d))
		}
	}
	b.WriteString(fmt.Sprintf("  entrypoint: %s\n", entrypoint))
	b.WriteString(fmt.Sprintf("  interface: %s\n", iface))
	return b.String()
}

func buildMsgraphYAML(pg *ParsedGraph) string {
	ek := EntryGrpcK8s(pg)
	var apps []string
	for _, eid := range pg.EntryNodeIDs {
		cg := RajomonCallgraphForEntry(pg, eid)
		iface := pg.Nodes[eid].Interface
		apps = append(apps, formatRajomonApplicationYAML(cg, ek, iface))
	}
	return "applications:\n" + strings.Join(apps, "")
}

const rajomonConfigGoTmpl = `package rajomoninit

import (
	"{{.Module}}/utils"
	"os"
	"strings"
	"time"

	"github.com/pennsail/rajomon"
	"gopkg.in/yaml.v2"
)

var rajomonOptions = map[string]interface{}{
	"initprice":       int64(0),
	"postPrice":       "",
	"postDelay":       "",
	"rateLimiting":    true,
	"loadShedding":    true,
	"pinpointQueuing": true,
	"debug":           false,
	"priceUpdateRate": 3235 * time.Microsecond,
	"priceStrategy":   "expdecay",
	"latencyThreshold": 1839 * time.Microsecond,
	"priceStep":        int64(319),
	"priceAggregation": "maximal",
	"recordPrice":      false,
}

var rajomonOptionsEnd = map[string]interface{}{
	"rateLimiting":     true,
	"loadShedding":     false,
	"pinpointQueuing":  false,
	"debug":            false,
	"priceUpdateRate":  10 * time.Millisecond,
	"priceStrategy":    "step",
	"latencyThreshold": 100 * time.Microsecond,
	"priceAggregation": "maximal",
	"tokensLeft":       int64(0),
	"initprice":        int64(0),
	"tokenUpdateStep":  int64(14),
	"tokenUpdateRate":  42573 * time.Microsecond,
	"tokenRefillDist":  "poisson",
	"tokenStrategy":    "uniform",
}

type Application struct {
	CallGraph map[string][]string ` + "`yaml:\"callgraph\"`" + `
	Interface string              ` + "`yaml:\"interface\"`" + `
}

type Node struct {
	Name       string   ` + "`yaml:\"name\"`" + `
	Value      string   ` + "`yaml:\"value\"`" + `
	URL        string   ` + "`yaml:\"URL\"`" + `
	Rajomon    []Config ` + "`yaml:\"rajomon\"`" + `
	Downstream []string ` + "`yaml:\"downstream\"`" + `
	Server     []Config ` + "`yaml:\"server\"`" + `
	ID         string   ` + "`yaml:\"id\"`" + `
}

type Config struct {
	Name  string ` + "`yaml:\"name\"`" + `
	Value string ` + "`yaml:\"value\"`" + `
}

var log = utils.GetLogger("rajomon-init")

const yamlFile = "../rajomon_init/msgraph.yaml"

const entryGrpcRajomonName = "{{.EntryGrpcK8s}}"

func SwapKeys(applications []Application) map[string]map[string][]string {
	downstreamMappings := make(map[string]map[string][]string)
	for _, app := range applications {
		callGraph := app.CallGraph
		interfaceName := app.Interface
		for upstream, downstreams := range callGraph {
			for _, downstream := range downstreams {
				if downstreamMappings[upstream] == nil {
					downstreamMappings[upstream] = make(map[string][]string)
				}
				if downstreamMappings[upstream][interfaceName] == nil {
					downstreamMappings[upstream][interfaceName] = make([]string, 0)
				}
				downstreamMappings[upstream][interfaceName] = append(downstreamMappings[upstream][interfaceName], downstream)
			}
		}
	}
	return downstreamMappings
}

func getCallGraph() map[string]map[string][]string {
	if _, err := os.Stat(yamlFile); os.IsNotExist(err) {
		panic("Call graph YAML file does not exist: " + yamlFile)
	}
	data, err := os.ReadFile(yamlFile)
	if err != nil {
		log.Error("[getCallGraph]", "err", err.Error())
	}
	var cfg struct {
		Applications []Application ` + "`yaml:\"applications\"`" + `
		Nodes        []Node        ` + "`yaml:\"nodes\"`" + `
	}
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		log.Error("[getCallGraph]", "err", err.Error())
	}
	return SwapKeys(cfg.Applications)
}

func loadEnvVars(accepted map[string]interface{}) map[string]interface{} {
	var value string
	value = utils.GetEnvVar("priceUpdateRate", true)
	accepted["priceUpdateRate"] = time.Duration(int64(utils.ParseFloatString(value))) * time.Microsecond
	value = utils.GetEnvVar("latencyThreshold", true)
	accepted["latencyThreshold"] = time.Duration(int64(utils.ParseFloatString(value))) * time.Microsecond
	value = utils.GetEnvVar("tokenUpdateRate", true)
	accepted["tokenUpdateRate"] = time.Duration(int64(utils.ParseFloatString(value))) * time.Microsecond
	value = utils.GetEnvVar("priceStep", true)
	accepted["priceStep"] = int64(utils.ParseFloatString(value))
	value = utils.GetEnvVar("tokenUpdateStep", true)
	accepted["tokenUpdateStep"] = int64(utils.ParseFloatString(value))
	return accepted
}

func GetPriceTable(name string, enduser bool) *rajomon.PriceTable {
	var callgraph map[string][]string
	var options map[string]interface{}
	if !enduser {
		callgraph = getCallGraph()[name]
		if utils.GetEnvVar("RAJOMON_"+strings.ToUpper(name)+"_DEBUG", false) == "true" {
			log.Debug("[GetPriceTable] turning on rajomon debug", "service", name)
			rajomonOptions["debug"] = true
		}
		options = rajomonOptions
	} else {
		callgraph = getCallGraph()[entryGrpcRajomonName]
		if utils.GetEnvVar("RAJOMON_CLIENT_DEBUG", false) == "true" {
			rajomonOptionsEnd["debug"] = true
		}
		options = rajomonOptionsEnd
	}
	log.Debug("[GetPriceTable]", "callgraph:", callgraph)
	if utils.GetEnvVar("RAJOMON_DEBUG", false) == "true" {
		options["debug"] = true
	}
	options = loadEnvVars(options)
	log.Info("[GetPriceTable]", "options:", options)
	return rajomon.NewRajomon(name, callgraph, options)
}
`

// GenerateRajomon writes rajomon_init/msgraph.yaml and rajomon_init/rajomon-config.go.
func GenerateRajomon(pg *ParsedGraph, module string, outDir string) error {
	dir := filepath.Join(outDir, "rajomon_init")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "msgraph.yaml"), []byte(buildMsgraphYAML(pg)), 0644); err != nil {
		return err
	}
	t, err := template.New("rajomoncfg").Parse(rajomonConfigGoTmpl)
	if err != nil {
		return err
	}
	var b strings.Builder
	if err := t.Execute(&b, map[string]string{
		"Module":       module,
		"EntryGrpcK8s": EntryGrpcK8s(pg),
	}); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "rajomon-config.go"), []byte(b.String()), 0644)
}
