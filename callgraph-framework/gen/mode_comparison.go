package gen

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func fmtCPU(v float64) string {
	if v < 0 {
		return "—"
	}
	if v == float64(int(v)) {
		return strconv.Itoa(int(v))
	}
	return strconv.FormatFloat(v, 'g', -1, 64)
}

func fmtInt(v int) string {
	if v < 0 {
		return "—"
	}
	return strconv.Itoa(v)
}

func fmtOverCommit(s ServiceDeploySpec) string {
	if !s.HasOverCommit {
		return "—"
	}
	return strconv.FormatFloat(s.OverCommitment, 'f', 1, 64)
}

func fmtRatio(r float64) string {
	if math.IsNaN(r) || math.IsInf(r, 0) {
		return "—"
	}
	return strconv.FormatFloat(r, 'g', 3, 64)
}

func renderMarkdownComparison(pg *ParsedGraph) string {
	var b strings.Builder
	b.WriteString("# Mode comparison\n\n")
	b.WriteString("Deploy resource comparison across generated modes (from callgraph.json).\n\n")

	b.WriteString("## Workloads\n\n")
	b.WriteString("| mode | service | app_cpu_limit | sidecar_cpu_limit | replicas | GOMAXPROCS | cpu_count | over_commitment | num_threads |\n")
	b.WriteString("|------|---------|---------------|-------------------|----------|------------|-----------|-----------------|-------------|\n")

	var summaryRows []struct {
		mode, totalApp, totalSidecar, ratio string
	}

	for _, mode := range allDeployModes {
		specs := DeploySpecForMode(pg, mode)
		for _, s := range specs {
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %d | %s | %s | %s | %s |\n",
				s.Mode, s.Service,
				fmtCPU(s.AppCPULimit), fmtCPU(s.SidecarCPULimit),
				s.Replicas, fmtInt(s.GOMAXPROCS),
				fmtInt(s.CPUCount), fmtOverCommit(s),
				fmtInt(s.NumThreads),
			)
		}
		app, side := ModeClusterTotals(specs)
		summaryRows = append(summaryRows, struct {
			mode, totalApp, totalSidecar, ratio string
		}{
			mode:         mode,
			totalApp:     fmtCPU(app),
			totalSidecar: fmtCPU(side),
			ratio:        fmtRatio(appSidecarRatio(app, side)),
		})
	}

	b.WriteString("\n## Cluster totals\n\n")
	b.WriteString("| mode | total_app_cores | total_sidecar_cores | app_sidecar_ratio |\n")
	b.WriteString("|------|-----------------|---------------------|-------------------|\n")
	for _, r := range summaryRows {
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", r.mode, r.totalApp, r.totalSidecar, r.ratio)
	}
	return b.String()
}

func csvEscape(s string) string {
	if strings.ContainsAny(s, ",\"\n") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}

func renderCSVComparison(pg *ParsedGraph) string {
	var lines []string
	header := "mode,service,app_cpu_limit,sidecar_cpu_limit,replicas,gomaxprocs,cpu_count,over_commitment,num_threads"
	lines = append(lines, header)

	for _, mode := range allDeployModes {
		specs := DeploySpecForMode(pg, mode)
		for _, s := range specs {
			row := []string{
				csvEscape(s.Mode),
				csvEscape(s.Service),
				csvEscape(fmtCPU(s.AppCPULimit)),
				csvEscape(fmtCPU(s.SidecarCPULimit)),
				strconv.Itoa(s.Replicas),
				csvEscape(fmtInt(s.GOMAXPROCS)),
				csvEscape(fmtInt(s.CPUCount)),
				csvEscape(fmtOverCommit(s)),
				csvEscape(fmtInt(s.NumThreads)),
			}
			lines = append(lines, strings.Join(row, ","))
		}
		app, side := ModeClusterTotals(specs)
		row := []string{
			csvEscape(mode + "_TOTAL"),
			"",
			csvEscape(fmtCPU(app)),
			csvEscape(fmtCPU(side)),
			"", "", "", "", "",
		}
		lines = append(lines, strings.Join(row, ","))
	}
	return strings.Join(lines, "\n") + "\n"
}

// WriteModeComparison writes mode-comparison.md and mode-comparison.csv to outDir.
func WriteModeComparison(pg *ParsedGraph, outDir string) error {
	mdPath := filepath.Join(outDir, "mode-comparison.md")
	csvPath := filepath.Join(outDir, "mode-comparison.csv")
	if err := os.WriteFile(mdPath, []byte(renderMarkdownComparison(pg)), 0644); err != nil {
		return fmt.Errorf("write mode-comparison.md: %w", err)
	}
	if err := os.WriteFile(csvPath, []byte(renderCSVComparison(pg)), 0644); err != nil {
		return fmt.Errorf("write mode-comparison.csv: %w", err)
	}
	return nil
}
