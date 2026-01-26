package rajomoninit

import (
	"hotel/utils"
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
	//"pinpointThroughput": false,
	//"pinpointLatency":    false,
	"debug": false,
	//"lazyResponse":       false,
	"priceUpdateRate": 3235 * time.Microsecond,
	//"guidePrice":       int64(-1),
	"priceStrategy":    "expdecay",
	"latencyThreshold": 1839 * time.Microsecond,
	"priceStep":        int64(319),
	"priceAggregation": "maximal",
	"recordPrice":      false,
}

var rajomonOptionsEnd = map[string]interface{}{
	"rateLimiting":    true,
	"loadShedding":    false,
	"pinpointQueuing": false,
	//"pinpointThroughput": false,
	//"pinpointLatency":    false,
	"debug": false,
	//"lazyResponse":       false,
	"priceUpdateRate": 10 * time.Millisecond,
	//"guidePrice":       int64(-1),
	"priceStrategy":    "step",
	"latencyThreshold": 100 * time.Microsecond,
	//"priceStep":        int64(5),
	"priceAggregation": "maximal",
	"tokensLeft":       int64(0),
	"initprice":        int64(0),
	"tokenUpdateStep":  int64(14),
	"tokenUpdateRate":  42573 * time.Microsecond,
	"tokenRefillDist":  "poisson",
	"tokenStrategy":    "uniform",
	//"clientTimeOut":    60 * time.Millisecond,
}

type Application struct {
	CallGraph map[string][]string `yaml:"callgraph"`
	Interface string              `yaml:"interface"`
}

type Node struct {
	Name       string   `yaml:"name"`
	Value      string   `yaml:"value"`
	URL        string   `yaml:"URL"`
	Rajomon    []Config `yaml:"rajomon"`
	Downstream []string `yaml:"downstream"`
	Server     []Config `yaml:"server"`
	ID         string   `yaml:"id"`
}

type Config struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

var log = utils.GetLogger("rajomon-init")

const yamlFile = "../rajomon_init/msgraph.yaml"

func SwapKeys(applications []Application) map[string]map[string][]string {
	// swappedGraph := make(map[string]map[string][]string)

	// return swappedGraph
	downstreamMappings := make(map[string]map[string][]string)

	for _, app := range applications {
		callGraph := app.CallGraph
		interfaceName := app.Interface

		// now, copy the value (slices of string) of callGraph into downstreamMappings[serviceName][interfaceName] (slice of string)
		for upstream, downstreams := range callGraph {
			// assign callGraph[upstream] to downstreamMappings[upstream][interfaceName]
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

func get_cwd() string {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return cwd
}

func getCallGraph() map[string]map[string][]string {
	//callGraph := make(map[string]map[string][]string)

	if _, err := os.Stat(yamlFile); os.IsNotExist(err) {
		log.Debug("[getCallGraph]", "cwd", get_cwd())
		panic("Call graph YAML file does not exist: " + yamlFile)
	}

	data, err := os.ReadFile(yamlFile)
	if err != nil {
		log.Error("[getCallGraph]", "err", err.Error())
	}

	var cfg struct {
		Applications []Application `yaml:"applications"`
		Nodes        []Node        `yaml:"nodes"`
	}

	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		log.Error("[getCallGraph]", "err", err.Error())
	}

	//fmt.Printf("applications: %++v\n", cfg.Applications)

	callGraph := SwapKeys(cfg.Applications)

	return callGraph
}

func loadEnvVars(accepted map[string]interface{}) map[string]interface{} {
	// load required environemnt variables
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
		callgraph = getCallGraph()["frontend-grpc"]
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

	priceTable := rajomon.NewRajomon(
		name,
		callgraph,
		options,
	)

	return priceTable
}
