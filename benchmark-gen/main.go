package main

import (
	"fmt"
	"os"

	"github.com/alexflint/go-arg"
)

type Args struct {
	Command string `arg:"positional" help:"Command: generate, deploy, or destroy"`
	Input   string `arg:"-i,--input" help:"Input YAML file path"`
	Output  string `arg:"-o,--output" help:"Output directory"`
	Config  string `arg:"-c,--config" help:"Deployment config file (for deploy/destroy commands)"`
	Mode    string `arg:"-m,--mode" help:"Deployment mode: plain or roshanfer (for deploy/destroy commands)" default:"plain"`
}

func printHelp() {
	fmt.Println("Usage: benchmark-gen <command> [options]")
	fmt.Println("Commands:")
	fmt.Println("  generate - Generate benchmark artifacts")
	fmt.Println("  deploy - Deploy benchmark to Kubernetes")
	fmt.Println("  destroy - Destroy/delete benchmark deployment from Kubernetes")
	fmt.Println("Options:")
	fmt.Println("  -i, --input <path> - Input YAML file path")
	fmt.Println("  -o, --output <dir> - Output directory")
	fmt.Println("  -c, --config <file> - Deployment config file (for deploy command)")
	fmt.Println("  -m, --mode <mode> - Deployment mode: plain or roshanfer (for deploy command, default: plain)")
}

func main() {
	var args Args
	arg.MustParse(&args)

	// check if input and output are provided (print help and exit if not)
	if args.Input == "" || args.Output == "" {
		printHelp()
		os.Exit(0)
	}

	switch args.Command {
	case "generate":
		if err := generate(args.Input, args.Output); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating benchmark: %v\n", err)
			os.Exit(1)
		}
	case "deploy":
		if err := deploy(args.Input, args.Output, args.Config, args.Mode); err != nil {
			fmt.Fprintf(os.Stderr, "Error deploying benchmark: %v\n", err)
			os.Exit(1)
		}
	case "destroy":
		if err := destroy(args.Output, args.Config, args.Mode); err != nil {
			fmt.Fprintf(os.Stderr, "Error destroying benchmark: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s. Use 'generate', 'deploy', or 'destroy'\n", args.Command)
		os.Exit(1)
	}
}
