package gen

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed embedded/dagor/*.go
var dagorEmbedded embed.FS

// GenerateDagor writes the embedded dagor package into outDir/dagor.
func GenerateDagor(outDir string) error {
	dir := filepath.Join(outDir, "dagor")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	entries, err := fs.ReadDir(dagorEmbedded, "embedded/dagor")
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		data, err := dagorEmbedded.ReadFile("embedded/dagor/" + name)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0644); err != nil {
			return err
		}
	}
	return nil
}

func dagorBusinessMapAndBmax(pg *ParsedGraph) (businessMapGo string, bmax int) {
	entries := pg.EntryInterfaces()
	sort.Slice(entries, func(i, j int) bool { return entries[i].Interface < entries[j].Interface })
	var b strings.Builder
	b.WriteString("map[string]int{\n")
	for i, n := range entries {
		b.WriteString(fmt.Sprintf("\t\t%q: %d,\n", n.Interface, i+1))
	}
	b.WriteString("\t}")
	n := len(entries)
	if n < 2 {
		bmax = 2
	} else {
		bmax = n
	}
	return b.String(), bmax
}

const dagorInitTmpl = `package dagorinit

import (
	"{{.Module}}/utils"
	"strings"
	"time"

	dagor "{{.Module}}/dagor"
)

var log = utils.GetLogger("dagor-init")

var dagorParams = dagor.DagorParam{
	NodeName:                     "",
	BusinessMap:                  {{.BusinessMap}},
	EntryService:                 false,
	IsEnduser:                    false,
	QueuingThresh:                {{printf "%g" .QueuingThreshMs}} * time.Millisecond,
	AdmissionLevelUpdateInterval: 10 * time.Millisecond,
	Alpha:                        {{printf "%g" .Alpha}},
	Beta:                         {{printf "%g" .Beta}},
	Umax:                         15,
	Bmax:                         {{.Bmax}},
	Debug:                        false,
	NumUsers:                     200,
	UseSyncMap:                   true,
	AddmissionUpdateN:            80,
}

func GetDagorNode(name string, entry, enduser bool) *dagor.Dagor {
	dagorParams.NodeName = name
	dagorParams.EntryService = entry
	dagorParams.IsEnduser = enduser

	if enduser {
		if utils.GetEnvVar("DAGOR_CLIENT_DEBUG", false) == "true" {
			log.Debug("[GetDagorNode] turning on dagor debug", "service", name)
			dagorParams.Debug = true
		}
	} else {
		if utils.GetEnvVar("DAGOR_"+strings.ToUpper(name)+"_DEBUG", false) == "true" {
			log.Debug("[GetDagorNode] turning on dagor debug", "service", name)
			dagorParams.Debug = true
		}
	}

	if utils.GetEnvVar("DAGOR_DEBUG", false) == "true" {
		log.Debug("[GetDagorNode] turning on dagor debug", "service", name)
		dagorParams.Debug = true
	}

	alpha := utils.GetEnvVar("Alpha", false)
	if alpha != "" {
		dagorParams.Alpha = utils.StrToFloat64(alpha)
		log.Info("Alpha got updated", "alpha", alpha)
	}
	beta := utils.GetEnvVar("Beta", false)
	if beta != "" {
		dagorParams.Beta = utils.StrToFloat64(beta)
		log.Info("Beta got updated", "beta", beta)
	}

	node := dagor.NewDagorNode(dagorParams)

	return node
}
`

// GenerateDagorInit writes dagor_init/dagor-config.go with BusinessMap from entry APIs.
func GenerateDagorInit(pg *ParsedGraph, module string, outDir string) error {
	dir := filepath.Join(outDir, "dagor_init")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	bm, bmax := dagorBusinessMapAndBmax(pg)
	return renderTemplate(dagorInitTmpl, map[string]interface{}{
		"Module":           module,
		"BusinessMap":      bm,
		"Bmax":             bmax,
		"QueuingThreshMs":  pg.DagorQueuingThreshMs,
		"Alpha":            pg.DagorAlpha,
		"Beta":             pg.DagorBeta,
	}, filepath.Join(dir, "dagor-config.go"))
}
