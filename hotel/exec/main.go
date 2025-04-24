package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/alexflint/go-arg"
)

var cancel context.CancelFunc
var ctx context.Context
var wg *sync.WaitGroup

func main() {
	var args struct {
		//Duration string `arg:"required"`
		//Rate     string `arg:"required"`
		Sidecar bool `arg:"--sidecar,-s" help:"Run sidecar"`
		Ppm     bool `arg:"--ppm,-p" help:"Run sidecar with PPM"`
	}

	arg.MustParse(&args)
	wg = &sync.WaitGroup{}
	fmt.Printf("args: %+v\n", args)

	serviceList := [][]string{
		{"geo", "1"},
		{"rate", "3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20"},
		{"search", "26,27,28,29,30"},
		{"profile", "20,21,22,23,24,25"},
		{"reservation", "31,32,33,34,35,36,37,38,38,40,46"},
		{"frontend", "41,42,43,44,45"},
	}

	if args.Ppm && !args.Sidecar {
		fmt.Println("PPM can only be used with sidecar")
		os.Exit(1)
	}

	// listen for SIGINT (Ctrl-C)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		<-sigCh
		fmt.Println("Received Ctrl-C, cancelling context")
		cancel()
		wg.Wait()
		os.Exit(0)
	}()

	ctx, cancel = context.WithCancel(context.Background())

	// run memcached and mongodb
	run_docker_compose()

	time.Sleep(time.Second * 2)

	// run the services
	run_servicees("./.env", serviceList)

	time.Sleep(time.Minute * 10)

	cancel()
	wg.Wait()
}

func run_docker_compose() {
	folder := "."
	dir := get_cwd() + "/" + folder
	c := exec.CommandContext(ctx, "sudo", "docker-compose", "-f", "docker-compose.yaml", "up", "-d")
	no_env_run(c, dir, false, "docker-compose")
}

func run_servicees(env string, serviceList [][]string) {
	for _, tuple := range serviceList {
		name := tuple[0]
		cpuset := tuple[1]

		// build the service
		folder := fmt.Sprintf("../%s", name)
		dir := get_cwd() + "/" + folder
		c := exec.CommandContext(ctx, "go", "build", "-o", fmt.Sprintf("%s.o", name), ".")
		c.Dir = dir
		no_env_run(c, dir, false, name)

		// run the service
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			fmt.Printf("Running %s\n", name)
			c = exec.CommandContext(ctx, "taskset", "-c", cpuset, fmt.Sprintf("./%s.o", name))
			//c = exec.CommandContext(ctx, fmt.Sprintf("./%s.o", name))
			env_run(c, dir, env)
		}(name)
	}
}

func read_env(name string) []string {
	envFile, err := os.ReadFile(name)
	if err != nil {
		fmt.Println("Error reading .env file:", err)
		cancel()
		panic(err)
	}
	envs := make([]string, 0)
	lines := strings.Split(string(envFile), "\n")
	for _, line := range lines {
		if line == "" || line[0] == '#' {
			continue // skip empty lines and comments
		}
		if !strings.Contains(line, "=") {
			continue // skip lines without '='
		}
		envs = append(envs, line)
	}

	// log the environment variables
	for _, env := range envs {
		fmt.Println("Environment variable:", env)
	}
	return envs
}

func no_env_run(c *exec.Cmd, dir string, profile bool, name string) {
	c.Dir = dir
	/* f, err := os.OpenFile(outputFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		fmt.Printf("Error opening file %s: %v\n", outputFile, err)
		cancel()
		panic(err)
	}
	defer f.Close()

	multiWriter := io.MultiWriter(os.Stdout, f)
	c.Stdout = multiWriter
	c.Stderr = multiWriter */
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	// create a new process group for the command
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// start the command
	if err := c.Start(); err != nil {
		cancel()
		panic(err)
	}

	pid := c.Process.Pid

	var prof_c *exec.Cmd
	if profile {
		prof_c = exec.Command("perf", "record", "-F", "999", "-g", "-p", fmt.Sprintf("%d", pid),
			"-o", fmt.Sprintf("%s.prof", name), "--call-graph", "dwarf")
		prof_c.Stdout = os.Stdout
		prof_c.Stderr = os.Stderr
		//prof_c.Dir = dir
		prof_c.Start()
	}

	done := make(chan error)
	go func() {
		done <- c.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			cancel()
			panic(err)
		}
	case <-ctx.Done():
		// on cancellation, kill the entire process group
		// negative PID kills the process group
		fmt.Println("Killing process group")
		syscall.Kill(-pid, syscall.SIGKILL)
		if prof_c != nil {
			prof_c.Wait()
		}
		<-done // wait for c.Wait() to return
	}
}

func env_run(c *exec.Cmd, dir, env string) {
	c.Dir = dir
	/* f, err := os.OpenFile(outputFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		fmt.Printf("Error opening file %s: %v\n", outputFile, err)
		cancel()
		panic(err)
	}
	defer f.Close()

	multiWriter := io.MultiWriter(os.Stdout, f)
	c.Stdout = multiWriter
	c.Stderr = multiWriter */
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Env = read_env(env)
	// create a new process group for the command
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := c.Start(); err != nil {
		cancel()
		panic(err)
	}

	pid := c.Process.Pid

	done := make(chan error)
	go func() {
		done <- c.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			cancel()
			panic(err)
		}
	case <-ctx.Done():
		// on cancellation, kill the entire process group
		fmt.Println("Killing process group")
		syscall.Kill(-pid, syscall.SIGKILL)
		<-done
	}
}

func get_cwd() string {
	cwd, err := os.Getwd()
	if err != nil {
		cancel()
		panic(err)
	}
	return cwd
}
