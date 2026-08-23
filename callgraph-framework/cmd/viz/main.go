package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"callgraph-framework/viz"
)

func findRepoRoot(start string) (string, error) {
	try := func(dir string) (string, error) {
		dir, err := filepath.Abs(dir)
		if err != nil {
			return "", err
		}
		if st, err := os.Stat(dir); err == nil && !st.IsDir() {
			dir = filepath.Dir(dir)
		}
		for i := 0; i < 16; i++ {
			cand := filepath.Join(dir, "benchmarks", "callgraph-framework", "viz", "render_service_pdf.py")
			venv := filepath.Join(dir, ".venv")
			if st, err := os.Stat(cand); err == nil && !st.IsDir() {
				if vi, err := os.Stat(venv); err == nil && vi.IsDir() {
					return dir, nil
				}
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
		return "", errors.New("not found")
	}
	if r, err := try(start); err == nil {
		return r, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if r, err := try(cwd); err == nil {
		return r, nil
	}
	return "", errors.New("could not locate repo root with .venv and benchmarks/callgraph-framework/viz/render_service_pdf.py (run from repo or pass absolute path to callgraph.json)")
}

// resolveVenvPython returns repoRoot/.venv/bin/python*; paper mode requires this venv.
func resolveVenvPython(repoRoot string) (string, error) {
	if runtime.GOOS == "windows" {
		p := filepath.Join(repoRoot, ".venv", "Scripts", "python.exe")
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, nil
		}
		return "", fmt.Errorf("missing %s; create with: python -m venv .venv", p)
	}
	for _, name := range []string{"python3", "python"} {
		p := filepath.Join(repoRoot, ".venv", "bin", name)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, nil
		}
	}
	return "", fmt.Errorf("missing %s/.venv/bin/python3; create at repo root: python3 -m venv .venv && .venv/bin/pip install -r requirements.txt", repoRoot)
}

func outPathFromTrailingFlags() string {
	args := flag.Args()
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-o" || args[i] == "--output" {
			return args[i+1]
		}
	}
	return ""
}

func runPaperViz(callgraphPath, outPath string) error {
	cg, err := filepath.Abs(callgraphPath)
	if err != nil {
		return err
	}
	root, err := findRepoRoot(filepath.Dir(cg))
	if err != nil {
		return err
	}
	script := filepath.Join(root, "benchmarks", "callgraph-framework", "viz", "render_service_pdf.py")
	if _, err := os.Stat(script); err != nil {
		return fmt.Errorf("paper viz script: %w", err)
	}
	py, err := resolveVenvPython(root)
	if err != nil {
		return err
	}
	cmd := exec.Command(py, script, cg, "-o", outPath)
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", py, err)
	}
	return nil
}

func main() {
	paper := flag.Bool("paper", false, "endpoint-level ACM quarter-column PDF with service clusters (requires repo .venv/bin/python3)")
	out := flag.String("o", "", "output PDF path")
	flag.Parse()
	if flag.NArg() < 1 {
		log.Fatal("usage: viz [-paper] <callgraph.json> [-o out.pdf]")
	}
	callgraphPath := flag.Arg(0)
	outPath := *out
	if outPath == "" {
		outPath = outPathFromTrailingFlags()
	}
	if outPath == "" {
		if *paper {
			outPath = filepath.Join(filepath.Dir(callgraphPath), "callgraph-service.pdf")
		} else {
			outPath = filepath.Join(filepath.Dir(callgraphPath), "callgraph.pdf")
		}
	}
	var err error
	if *paper {
		err = runPaperViz(callgraphPath, outPath)
	} else {
		err = viz.Visualize(callgraphPath, outPath)
	}
	if err != nil {
		log.Fatalf("viz: %v", err)
	}
	fmt.Printf("Generated %s\n", outPath)
}
