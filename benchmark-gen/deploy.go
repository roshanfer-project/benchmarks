package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type DeployConfig struct {
	Kubeconfig  string `yaml:"kubeconfig,omitempty"`
	Namespace   string `yaml:"namespace"`
	BuildImages bool   `yaml:"build_images,omitempty"`
}

func deploy(inputFile, outputDir, configFile, mode string) error {
	// Validate mode
	if mode != "plain" && mode != "roshanfer" {
		return fmt.Errorf("invalid mode: %s (must be 'plain' or 'roshanfer')", mode)
	}
	// Read deploy config
	var deployConfig DeployConfig
	if configFile != "" {
		data, err := os.ReadFile(configFile)
		if err != nil {
			return fmt.Errorf("failed to read deploy config: %w", err)
		}
		if err := yaml.Unmarshal(data, &deployConfig); err != nil {
			return fmt.Errorf("failed to parse deploy config: %w", err)
		}
	}

	if deployConfig.Namespace == "" {
		deployConfig.Namespace = "app"
	}

	// Determine kubectl command (check if kubectl exists, otherwise use k3s or microk8s)
	kubectlCmd := "kubectl"
	if _, err := exec.LookPath("kubectl"); err != nil {
		// kubectl not in PATH, try k3s first
		if _, err := exec.LookPath("k3s"); err == nil {
			kubectlCmd = "k3s"
		} else if _, err := exec.LookPath("microk8s"); err == nil {
			kubectlCmd = "microk8s"
		} else {
			return fmt.Errorf("none of kubectl, k3s, or microk8s found in PATH")
		}
	}

	// Check if namespace exists and is terminating, wait for it to be deleted
	var checkNsCmd *exec.Cmd
	if kubectlCmd == "microk8s" {
		checkNsCmd = exec.Command("sudo", "microk8s", "kubectl", "get", "namespace", deployConfig.Namespace, "--ignore-not-found=true", "-o", "jsonpath={.status.phase}")
	} else if kubectlCmd == "k3s" {
		checkNsCmd = exec.Command("sudo", "k3s", "kubectl", "get", "namespace", deployConfig.Namespace, "--ignore-not-found=true", "-o", "jsonpath={.status.phase}")
	} else {
		checkNsCmd = exec.Command("kubectl", "get", "namespace", deployConfig.Namespace, "--ignore-not-found=true", "-o", "jsonpath={.status.phase}")
		if deployConfig.Kubeconfig != "" {
			checkNsCmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", deployConfig.Kubeconfig))
		}
	}

	nsPhase, err := checkNsCmd.Output()
	nsPhaseStr := strings.TrimSpace(string(nsPhase))
	if err == nil && nsPhaseStr == "Terminating" {
		fmt.Printf("Namespace %s is being terminated, waiting for deletion to complete...\n", deployConfig.Namespace)
		// Wait for namespace to be deleted (max 30 seconds)
		deleted := false
		for i := 0; i < 30; i++ {
			time.Sleep(1 * time.Second)
			var waitCmd *exec.Cmd
			if kubectlCmd == "microk8s" {
				waitCmd = exec.Command("sudo", "microk8s", "kubectl", "get", "namespace", deployConfig.Namespace, "--ignore-not-found=true", "-o", "jsonpath={.status.phase}")
			} else if kubectlCmd == "k3s" {
				waitCmd = exec.Command("sudo", "k3s", "kubectl", "get", "namespace", deployConfig.Namespace, "--ignore-not-found=true", "-o", "jsonpath={.status.phase}")
			} else {
				waitCmd = exec.Command("kubectl", "get", "namespace", deployConfig.Namespace, "--ignore-not-found=true", "-o", "jsonpath={.status.phase}")
				if deployConfig.Kubeconfig != "" {
					waitCmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", deployConfig.Kubeconfig))
				}
			}
			waitPhase, waitErr := waitCmd.Output()
			waitPhaseStr := strings.TrimSpace(string(waitPhase))
			// Check if namespace still exists - if command fails or returns empty, namespace is gone
			if waitErr != nil || len(waitPhaseStr) == 0 {
				// Namespace is deleted
				fmt.Printf("Namespace %s has been deleted.\n", deployConfig.Namespace)
				deleted = true
				break
			}
			if waitPhaseStr != "Terminating" {
				// Namespace is no longer terminating (might be Active now)
				break
			}
		}
		// If namespace is still terminating after 30 seconds, try to force delete it
		if !deleted {
			fmt.Printf("Namespace %s is stuck in Terminating state, attempting to force delete...\n", deployConfig.Namespace)
			var forceDeleteCmd *exec.Cmd
			if kubectlCmd == "microk8s" {
				// First try to remove finalizers
				patchCmd := exec.Command("sudo", "microk8s", "kubectl", "patch", "namespace", deployConfig.Namespace, "-p", `{"metadata":{"finalizers":[]}}`, "--type=merge")
				patchCmd.Run() // Ignore errors
				forceDeleteCmd = exec.Command("sudo", "microk8s", "kubectl", "delete", "namespace", deployConfig.Namespace, "--force", "--grace-period=0", "--ignore-not-found=true")
			} else if kubectlCmd == "k3s" {
				patchCmd := exec.Command("sudo", "k3s", "kubectl", "patch", "namespace", deployConfig.Namespace, "-p", `{"metadata":{"finalizers":[]}}`, "--type=merge")
				patchCmd.Run() // Ignore errors
				forceDeleteCmd = exec.Command("sudo", "k3s", "kubectl", "delete", "namespace", deployConfig.Namespace, "--force", "--grace-period=0", "--ignore-not-found=true")
			} else {
				patchCmd := exec.Command("kubectl", "patch", "namespace", deployConfig.Namespace, "-p", `{"metadata":{"finalizers":[]}}`, "--type=merge")
				if deployConfig.Kubeconfig != "" {
					patchCmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", deployConfig.Kubeconfig))
				}
				patchCmd.Run() // Ignore errors
				forceDeleteCmd = exec.Command("kubectl", "delete", "namespace", deployConfig.Namespace, "--force", "--grace-period=0", "--ignore-not-found=true")
				if deployConfig.Kubeconfig != "" {
					forceDeleteCmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", deployConfig.Kubeconfig))
				}
			}
			forceDeleteCmd.Run() // Ignore errors, just try
			// Wait a bit more and verify deletion
			time.Sleep(3 * time.Second)
			var verifyCmd *exec.Cmd
			if kubectlCmd == "microk8s" {
				verifyCmd = exec.Command("sudo", "microk8s", "kubectl", "get", "namespace", deployConfig.Namespace, "--ignore-not-found=true", "-o", "jsonpath={.status.phase}")
			} else if kubectlCmd == "k3s" {
				verifyCmd = exec.Command("sudo", "k3s", "kubectl", "get", "namespace", deployConfig.Namespace, "--ignore-not-found=true", "-o", "jsonpath={.status.phase}")
			} else {
				verifyCmd = exec.Command("kubectl", "get", "namespace", deployConfig.Namespace, "--ignore-not-found=true", "-o", "jsonpath={.status.phase}")
				if deployConfig.Kubeconfig != "" {
					verifyCmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", deployConfig.Kubeconfig))
				}
			}
			verifyPhase, _ := verifyCmd.Output()
			verifyPhaseStr := strings.TrimSpace(string(verifyPhase))
			if len(verifyPhaseStr) == 0 {
				fmt.Printf("Namespace %s has been force deleted.\n", deployConfig.Namespace)
			} else {
				fmt.Printf("Warning: Namespace %s is still in state: %s\n", deployConfig.Namespace, verifyPhaseStr)
			}
		}
	}

	k8sDir := filepath.Join(outputDir, "k8s")

	// Build images if requested
	if deployConfig.BuildImages {
		if err := buildImages(outputDir); err != nil {
			return fmt.Errorf("failed to build images: %w", err)
		}
	}

	// Import sidecar image to k3s if in roshanfer mode (sidecars are only used in roshanfer mode)
	if mode == "roshanfer" {
		if err := importSidecarImageToK3s(); err != nil {
			fmt.Printf("Warning: failed to import sidecar image to k3s: %v\n", err)
		}
	}

	// Apply Kubernetes manifests
	manifests := []string{
		"namespace.yaml",
		"services.yaml",
	}

	// Add NodePort service based on mode
	var nodeportFile string
	if mode == "roshanfer" {
		nodeportFile = "nodeport-services-roshanfer.yaml"
	} else {
		nodeportFile = "nodeport-services-plain.yaml"
	}
	nodeportPath := filepath.Join(k8sDir, nodeportFile)
	if _, err := os.Stat(nodeportPath); err == nil {
		manifests = append(manifests, nodeportFile)
	}

	// Add pod files based on mode
	files, err := os.ReadDir(k8sDir)
	if err != nil {
		return fmt.Errorf("failed to read k8s directory: %w", err)
	}

	podSuffix := "-pod-plain.yaml"
	configmapSuffix := "-configmap.yaml"
	if mode == "roshanfer" {
		podSuffix = "-pod-roshanfer.yaml"
	}

	// Collect pod files and calculate deployment order
	podFiles := make(map[string]string) // service name -> filename
	var allServices []string
	for _, file := range files {
		fileName := file.Name()
		if strings.HasSuffix(fileName, podSuffix) {
			// Extract service name from filename (e.g., "app-pod-roshanfer.yaml" -> "app")
			serviceName := strings.TrimSuffix(fileName, podSuffix)
			serviceName = strings.TrimSuffix(serviceName, "-pod")
			podFiles[serviceName] = fileName
			allServices = append(allServices, serviceName)
		}
	}

	// Read benchmark config to determine deployment order
	var config *BenchmarkConfig
	if inputFile != "" {
		if data, err := os.ReadFile(inputFile); err == nil {
			var cfg BenchmarkConfig
			if err := yaml.Unmarshal(data, &cfg); err == nil {
				config = &cfg
			}
		}
	}

	// Add configmaps BEFORE pods (only used in roshanfer mode, but always include)
	for _, file := range files {
		fileName := file.Name()
		if strings.HasSuffix(fileName, configmapSuffix) {
			if mode == "roshanfer" {
				manifests = append(manifests, fileName)
			}
		}
	}

	// Calculate deployment order based on dependencies
	var deploymentOrder []string
	if config != nil {
		deploymentOrder = calculateDeploymentOrder(config, allServices)
	} else {
		// Fallback: use alphabetical order
		sort.Strings(allServices)
		deploymentOrder = allServices
	}

	// Add pods in dependency order
	for _, serviceName := range deploymentOrder {
		if fileName, ok := podFiles[serviceName]; ok {
			manifests = append(manifests, fileName)
		}
	}

	// Add ingress pod last (only in roshanfer mode)
	if mode == "roshanfer" {
		ingressPodPath := filepath.Join(k8sDir, "ingress-pod.yaml")
		if _, err := os.Stat(ingressPodPath); err == nil {
			manifests = append(manifests, "ingress-pod.yaml")
		}
	}

	// Apply manifests
	for _, manifest := range manifests {
		manifestPath := filepath.Join(k8sDir, manifest)
		if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
			continue
		}

		var cmd *exec.Cmd
		if kubectlCmd == "microk8s" {
			// Use sudo microk8s kubectl
			cmd = exec.Command("sudo", "microk8s", "kubectl", "apply", "-f", manifestPath, "--validate=false")
		} else if kubectlCmd == "k3s" {
			cmd = exec.Command("sudo", "k3s", "kubectl", "apply", "-f", manifestPath, "--validate=false")
		} else {
			cmd = exec.Command("kubectl", "apply", "-f", manifestPath, "--validate=false")
			if deployConfig.Kubeconfig != "" {
				cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", deployConfig.Kubeconfig))
			}
		}
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to apply %s: %w", manifest, err)
		}

		// Wait for pod to be ready if this is a pod manifest
		if strings.Contains(manifest, "-pod-") || manifest == "ingress-pod.yaml" {
			// Extract pod name from manifest filename
			var podName string
			if manifest == "ingress-pod.yaml" {
				// Ingress pod is always named "ingress-sidecar"
				podName = "ingress-sidecar"
			} else {
				podName = strings.TrimSuffix(manifest, ".yaml")
				podName = strings.TrimSuffix(podName, "-pod-roshanfer")
				podName = strings.TrimSuffix(podName, "-pod-plain")
			}

			fmt.Printf("Waiting for pod %s to be ready...\n", podName)
			if err := waitForPodReady(kubectlCmd, deployConfig.Kubeconfig, deployConfig.Namespace, podName); err != nil {
				fmt.Printf("Warning: pod %s may not be ready: %v\n", podName, err)
			}
		}
	}

	fmt.Printf("Successfully deployed benchmark to namespace %s\n", deployConfig.Namespace)
	return nil
}

func destroy(outputDir, configFile, mode string) error {
	// Validate mode
	if mode != "plain" && mode != "roshanfer" {
		return fmt.Errorf("invalid mode: %s (must be 'plain' or 'roshanfer')", mode)
	}

	// Read deploy config
	var deployConfig DeployConfig
	if configFile != "" {
		data, err := os.ReadFile(configFile)
		if err != nil {
			return fmt.Errorf("failed to read deploy config: %w", err)
		}
		if err := yaml.Unmarshal(data, &deployConfig); err != nil {
			return fmt.Errorf("failed to parse deploy config: %w", err)
		}
	}

	if deployConfig.Namespace == "" {
		deployConfig.Namespace = "app"
	}

	k8sDir := filepath.Join(outputDir, "k8s")

	// Determine kubectl command (check if kubectl exists, otherwise use k3s or microk8s)
	kubectlCmd := "kubectl"
	if _, err := exec.LookPath("kubectl"); err != nil {
		// kubectl not in PATH, try k3s first
		if _, err := exec.LookPath("k3s"); err == nil {
			kubectlCmd = "k3s"
		} else if _, err := exec.LookPath("microk8s"); err == nil {
			kubectlCmd = "microk8s"
		} else {
			return fmt.Errorf("none of kubectl, k3s, or microk8s found in PATH")
		}
	}

	// Collect manifests to delete (in reverse order: pods/configmaps first, then services, then namespace)
	var manifests []string

	// Add pod files based on mode
	files, err := os.ReadDir(k8sDir)
	if err != nil {
		return fmt.Errorf("failed to read k8s directory: %w", err)
	}

	podSuffix := "-pod-plain.yaml"
	configmapSuffix := "-configmap.yaml"
	if mode == "roshanfer" {
		podSuffix = "-pod-roshanfer.yaml"
	}

	var nodeportFile string
	if mode == "roshanfer" {
		nodeportFile = "nodeport-services-roshanfer.yaml"
	} else {
		nodeportFile = "nodeport-services-plain.yaml"
	}

	for _, file := range files {
		fileName := file.Name()
		// Add pods matching the selected mode
		if strings.HasSuffix(fileName, podSuffix) {
			manifests = append(manifests, fileName)
		}
		// Add configmaps (only used in roshanfer mode)
		if strings.HasSuffix(fileName, configmapSuffix) {
			if mode == "roshanfer" {
				manifests = append(manifests, fileName)
			}
		}
		// Add ingress pod (only in roshanfer mode)
		if mode == "roshanfer" && fileName == "ingress-pod.yaml" {
			manifests = append(manifests, fileName)
		}
		// Add ingress configmap (only in roshanfer mode)
		if mode == "roshanfer" && fileName == "ingress-configmap.yaml" {
			manifests = append(manifests, fileName)
		}
		// Add NodePort services (based on mode)
		if fileName == nodeportFile {
			manifests = append(manifests, fileName)
		}
	}

	// Add services (delete before namespace)
	manifests = append(manifests, "services.yaml")

	// Delete manifests
	for _, manifest := range manifests {
		manifestPath := filepath.Join(k8sDir, manifest)
		if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
			continue
		}

		var cmd *exec.Cmd
		if kubectlCmd == "microk8s" {
			// Use sudo microk8s kubectl
			cmd = exec.Command("sudo", "microk8s", "kubectl", "delete", "-f", manifestPath, "--ignore-not-found=true")
		} else if kubectlCmd == "k3s" {
			cmd = exec.Command("sudo", "k3s", "kubectl", "delete", "-f", manifestPath, "--ignore-not-found=true")
		} else {
			cmd = exec.Command("kubectl", "delete", "-f", manifestPath, "--ignore-not-found=true")
			if deployConfig.Kubeconfig != "" {
				cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", deployConfig.Kubeconfig))
			}
		}
		cmd.Stdout = os.Stdout

		// Capture stderr to filter out watch warnings
		stderrPipe, err := cmd.StderrPipe()
		if err != nil {
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				fmt.Printf("Warning: failed to delete %s (may not exist)\n", manifest)
			}
		} else {
			// Start the command
			if err := cmd.Start(); err != nil {
				fmt.Printf("Warning: failed to start delete command for %s: %v\n", manifest, err)
				continue
			}

			// Filter stderr to remove watch warnings in a goroutine
			done := make(chan bool)
			go func() {
				scanner := bufio.NewScanner(stderrPipe)
				for scanner.Scan() {
					line := scanner.Text()
					// Filter out kubectl watch warnings
					if !strings.Contains(line, "reflector.go") &&
						!strings.Contains(line, "watch of") &&
						!strings.Contains(line, "Unexpected watch close") &&
						!strings.Contains(line, "very short watch") {
						fmt.Fprintf(os.Stderr, "%s\n", line)
					}
				}
				done <- true
			}()

			// Wait for command to complete
			if err := cmd.Wait(); err != nil {
				// Continue even if resource doesn't exist (--ignore-not-found handles this)
				fmt.Printf("Warning: failed to delete %s (may not exist)\n", manifest)
			}

			// Wait for stderr reading to complete
			<-done
		}
	}

	// Delete all remaining resources in the namespace
	// kubectl delete all --all doesn't delete pods and configmaps, so delete them explicitly
	fmt.Printf("Deleting all remaining resources in namespace %s...\n", deployConfig.Namespace)

	// Delete pods
	var deletePodsCmd *exec.Cmd
	if kubectlCmd == "microk8s" {
		deletePodsCmd = exec.Command("sudo", "microk8s", "kubectl", "delete", "pods", "--all", "-n", deployConfig.Namespace, "--ignore-not-found=true")
	} else if kubectlCmd == "k3s" {
		deletePodsCmd = exec.Command("sudo", "k3s", "kubectl", "delete", "pods", "--all", "-n", deployConfig.Namespace, "--ignore-not-found=true")
	} else {
		deletePodsCmd = exec.Command("kubectl", "delete", "pods", "--all", "-n", deployConfig.Namespace, "--ignore-not-found=true")
		if deployConfig.Kubeconfig != "" {
			deletePodsCmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", deployConfig.Kubeconfig))
		}
	}
	deletePodsCmd.Stdout = os.Stdout
	deletePodsCmd.Stderr = os.Stderr
	deletePodsCmd.Run() // Ignore errors - resources may not exist

	// Delete configmaps
	var deleteConfigMapsCmd *exec.Cmd
	if kubectlCmd == "microk8s" {
		deleteConfigMapsCmd = exec.Command("sudo", "microk8s", "kubectl", "delete", "configmaps", "--all", "-n", deployConfig.Namespace, "--ignore-not-found=true")
	} else if kubectlCmd == "k3s" {
		deleteConfigMapsCmd = exec.Command("sudo", "k3s", "kubectl", "delete", "configmaps", "--all", "-n", deployConfig.Namespace, "--ignore-not-found=true")
	} else {
		deleteConfigMapsCmd = exec.Command("kubectl", "delete", "configmaps", "--all", "-n", deployConfig.Namespace, "--ignore-not-found=true")
		if deployConfig.Kubeconfig != "" {
			deleteConfigMapsCmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", deployConfig.Kubeconfig))
		}
	}
	deleteConfigMapsCmd.Stdout = os.Stdout
	deleteConfigMapsCmd.Stderr = os.Stderr
	deleteConfigMapsCmd.Run() // Ignore errors - resources may not exist

	// Delete other resources (pods, services, etc.)
	var deleteAllCmd *exec.Cmd
	if kubectlCmd == "microk8s" {
		deleteAllCmd = exec.Command("sudo", "microk8s", "kubectl", "delete", "all", "--all", "-n", deployConfig.Namespace, "--ignore-not-found=true")
	} else if kubectlCmd == "k3s" {
		deleteAllCmd = exec.Command("sudo", "k3s", "kubectl", "delete", "all", "--all", "-n", deployConfig.Namespace, "--ignore-not-found=true")
	} else {
		deleteAllCmd = exec.Command("kubectl", "delete", "all", "--all", "-n", deployConfig.Namespace, "--ignore-not-found=true")
		if deployConfig.Kubeconfig != "" {
			deleteAllCmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", deployConfig.Kubeconfig))
		}
	}
	deleteAllCmd.Stdout = os.Stdout
	deleteAllCmd.Stderr = os.Stderr
	deleteAllCmd.Run() // Ignore errors - resources may not exist

	// Finally, delete the namespace (this will cascade delete everything)
	fmt.Printf("Deleting namespace %s...\n", deployConfig.Namespace)
	var deleteNsCmd *exec.Cmd
	if kubectlCmd == "microk8s" {
		deleteNsCmd = exec.Command("sudo", "microk8s", "kubectl", "delete", "namespace", deployConfig.Namespace, "--ignore-not-found=true", "--wait=false")
	} else if kubectlCmd == "k3s" {
		deleteNsCmd = exec.Command("sudo", "k3s", "kubectl", "delete", "namespace", deployConfig.Namespace, "--ignore-not-found=true", "--wait=false")
	} else {
		deleteNsCmd = exec.Command("kubectl", "delete", "namespace", deployConfig.Namespace, "--ignore-not-found=true", "--wait=false")
		if deployConfig.Kubeconfig != "" {
			deleteNsCmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", deployConfig.Kubeconfig))
		}
	}
	deleteNsCmd.Stdout = os.Stdout

	// Filter stderr for namespace deletion too
	nsStderrPipe, err := deleteNsCmd.StderrPipe()
	if err != nil {
		deleteNsCmd.Stderr = os.Stderr
		deleteNsCmd.Run()
	} else {
		deleteNsCmd.Start()
		nsDone := make(chan bool)
		go func() {
			scanner := bufio.NewScanner(nsStderrPipe)
			for scanner.Scan() {
				line := scanner.Text()
				if !strings.Contains(line, "reflector.go") &&
					!strings.Contains(line, "watch of") &&
					!strings.Contains(line, "Unexpected watch close") &&
					!strings.Contains(line, "very short watch") {
					fmt.Fprintf(os.Stderr, "%s\n", line)
				}
			}
			nsDone <- true
		}()
		deleteNsCmd.Wait()
		<-nsDone
	}

	// Wait for namespace deletion to complete and handle stuck namespaces
	fmt.Printf("Waiting for namespace %s to be deleted...\n", deployConfig.Namespace)
	deleted := false
	for i := 0; i < 60; i++ {
		time.Sleep(100 * time.Millisecond)
		var checkCmd *exec.Cmd
		if kubectlCmd == "microk8s" {
			checkCmd = exec.Command("sudo", "microk8s", "kubectl", "get", "namespace", deployConfig.Namespace, "--ignore-not-found=true", "-o", "jsonpath={.status.phase}")
		} else if kubectlCmd == "k3s" {
			checkCmd = exec.Command("sudo", "k3s", "kubectl", "get", "namespace", deployConfig.Namespace, "--ignore-not-found=true", "-o", "jsonpath={.status.phase}")
		} else {
			checkCmd = exec.Command("kubectl", "get", "namespace", deployConfig.Namespace, "--ignore-not-found=true", "-o", "jsonpath={.status.phase}")
			if deployConfig.Kubeconfig != "" {
				checkCmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", deployConfig.Kubeconfig))
			}
		}
		phase, err := checkCmd.Output()
		phaseStr := strings.TrimSpace(string(phase))
		// Check if namespace still exists - if command fails or returns empty, namespace is gone
		if err != nil || len(phaseStr) == 0 {
			// Namespace is deleted
			fmt.Printf("Namespace %s has been deleted.\n", deployConfig.Namespace)
			deleted = true
			break
		}
		if phaseStr == "Terminating" {
			// Still terminating, continue waiting
			continue
		}
		// Namespace is no longer terminating (might be Active now, which shouldn't happen)
		fmt.Printf("Warning: Namespace %s is in unexpected state: %s\n", deployConfig.Namespace, phaseStr)
		break
	}

	// If namespace is still terminating after 60 seconds, try to force delete it
	if !deleted {
		fmt.Printf("Namespace %s is stuck in Terminating state, attempting to force delete...\n", deployConfig.Namespace)
		var forceDeleteCmd *exec.Cmd
		if kubectlCmd == "microk8s" {
			// First try to remove finalizers
			patchCmd := exec.Command("sudo", "microk8s", "kubectl", "patch", "namespace", deployConfig.Namespace, "-p", `{"metadata":{"finalizers":[]}}`, "--type=merge")
			patchCmd.Run() // Ignore errors
			forceDeleteCmd = exec.Command("sudo", "microk8s", "kubectl", "delete", "namespace", deployConfig.Namespace, "--force", "--grace-period=0", "--ignore-not-found=true")
		} else if kubectlCmd == "k3s" {
			patchCmd := exec.Command("sudo", "k3s", "kubectl", "patch", "namespace", deployConfig.Namespace, "-p", `{"metadata":{"finalizers":[]}}`, "--type=merge")
			patchCmd.Run() // Ignore errors
			forceDeleteCmd = exec.Command("sudo", "k3s", "kubectl", "delete", "namespace", deployConfig.Namespace, "--force", "--grace-period=0", "--ignore-not-found=true")
		} else {
			patchCmd := exec.Command("kubectl", "patch", "namespace", deployConfig.Namespace, "-p", `{"metadata":{"finalizers":[]}}`, "--type=merge")
			if deployConfig.Kubeconfig != "" {
				patchCmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", deployConfig.Kubeconfig))
			}
			patchCmd.Run() // Ignore errors
			forceDeleteCmd = exec.Command("kubectl", "delete", "namespace", deployConfig.Namespace, "--force", "--grace-period=0", "--ignore-not-found=true")
			if deployConfig.Kubeconfig != "" {
				forceDeleteCmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", deployConfig.Kubeconfig))
			}
		}
		forceDeleteCmd.Stdout = os.Stdout
		forceDeleteCmd.Stderr = os.Stderr
		forceDeleteCmd.Run() // Ignore errors, just try
		// Wait a bit more and verify deletion
		time.Sleep(3 * time.Second)
		var verifyCmd *exec.Cmd
		if kubectlCmd == "microk8s" {
			verifyCmd = exec.Command("sudo", "microk8s", "kubectl", "get", "namespace", deployConfig.Namespace, "--ignore-not-found=true", "-o", "jsonpath={.status.phase}")
		} else if kubectlCmd == "k3s" {
			verifyCmd = exec.Command("sudo", "k3s", "kubectl", "get", "namespace", deployConfig.Namespace, "--ignore-not-found=true", "-o", "jsonpath={.status.phase}")
		} else {
			verifyCmd = exec.Command("kubectl", "get", "namespace", deployConfig.Namespace, "--ignore-not-found=true", "-o", "jsonpath={.status.phase}")
			if deployConfig.Kubeconfig != "" {
				verifyCmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", deployConfig.Kubeconfig))
			}
		}
		verifyPhase, _ := verifyCmd.Output()
		verifyPhaseStr := strings.TrimSpace(string(verifyPhase))
		if len(verifyPhaseStr) == 0 {
			fmt.Printf("Namespace %s has been force deleted.\n", deployConfig.Namespace)
		} else {
			fmt.Printf("Warning: Namespace %s is still in state: %s\n", deployConfig.Namespace, verifyPhaseStr)
		}
	}

	fmt.Printf("Successfully destroyed benchmark deployment in namespace %s\n", deployConfig.Namespace)
	return nil
}

func buildImages(outputDir string) error {
	// Build frontend image
	frontendDir := filepath.Join(outputDir, "services", "frontend")
	if _, err := os.Stat(frontendDir); err == nil {
		if err := buildFrontendImage(frontendDir); err != nil {
			return fmt.Errorf("failed to build frontend image: %w", err)
		}
	}

	// Build backend image only if backend directory exists
	backendDir := filepath.Join(outputDir, "services", "backend")
	if _, err := os.Stat(backendDir); err == nil {
		if err := buildBackendImage(backendDir); err != nil {
			return fmt.Errorf("failed to build backend image: %w", err)
		}
	}

	return nil
}

func buildFrontendImage(serviceDir string) error {
	// Generate Dockerfile
	dockerfile := `FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy protobuf files (need to generate first)
COPY protobuf ./protobuf

# Copy test utilities
COPY test ./test

# Copy service source
COPY services/frontend ./services/frontend

# Build the service
WORKDIR /app/services/frontend
RUN go mod tidy
RUN go build -o frontend .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/services/frontend/frontend .
CMD ["./frontend"]
`

	dockerfilePath := filepath.Join(serviceDir, "..", "..", "Dockerfile.frontend")
	if err := os.WriteFile(dockerfilePath, []byte(dockerfile), 0644); err != nil {
		return fmt.Errorf("failed to write Dockerfile: %w", err)
	}

	// Build image
	buildDir := filepath.Join(serviceDir, "..", "..")
	imageName := "benchmark-frontend:latest"
	cmd := exec.Command("docker", "build", "-f", "Dockerfile.frontend", "-t", imageName, ".")
	cmd.Dir = buildDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build frontend image: %w", err)
	}

	// Import to k3s if present
	if err := importImageToK3s(imageName); err != nil {
		fmt.Printf("Warning: failed to import image to k3s: %v\n", err)
	}

	return nil
}

func buildBackendImage(serviceDir string) error {
	// Generate Dockerfile
	dockerfile := `FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy protobuf files (need to generate first)
COPY protobuf ./protobuf

# Copy test utilities
COPY test ./test

# Copy service source
COPY services/backend ./services/backend

# Build the service
WORKDIR /app/services/backend
RUN go mod tidy
RUN go build -o backend .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/services/backend/backend .
CMD ["./backend"]
`

	dockerfilePath := filepath.Join(serviceDir, "..", "..", "Dockerfile.backend")
	if err := os.WriteFile(dockerfilePath, []byte(dockerfile), 0644); err != nil {
		return fmt.Errorf("failed to write Dockerfile: %w", err)
	}

	// Build image
	buildDir := filepath.Join(serviceDir, "..", "..")
	imageName := "benchmark-backend:latest"
	cmd := exec.Command("docker", "build", "-f", "Dockerfile.backend", "-t", imageName, ".")
	cmd.Dir = buildDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build backend image: %w", err)
	}

	// Import to k3s if present
	if err := importImageToK3s(imageName); err != nil {
		fmt.Printf("Warning: failed to import image to k3s: %v\n", err)
	}

	return nil
}

// importImageToK3s imports a docker image into k3s containerd if k3s is installed
// It only imports if the image doesn't exist or has changed (different digest)
func importImageToK3s(imageName string) error {
	// Check if k3s is installed
	if _, err := exec.LookPath("k3s"); err != nil {
		// k3s not found, skip import
		return nil
	}

	// Get Docker image ID (sha256 hash)
	dockerInspectCmd := exec.Command("docker", "inspect", "--format={{.Id}}", imageName)
	dockerId, err := dockerInspectCmd.Output()
	if err != nil {
		// Image doesn't exist in Docker, skip import
		fmt.Printf("Image %s not found in Docker, skipping import\n", imageName)
		return nil
	}
	dockerIdStr := strings.TrimSpace(string(dockerId))
	// Extract hash part (remove sha256: prefix if present)
	dockerHash := dockerIdStr
	if strings.HasPrefix(dockerIdStr, "sha256:") {
		dockerHash = strings.TrimPrefix(dockerIdStr, "sha256:")
	}

	// Check if image exists in k3s and compare image ID
	// Try both the short name and the full docker.io/library/ prefix name
	k3sImageNames := []string{imageName}
	if !strings.Contains(imageName, "/") {
		// If no registry prefix, also try with docker.io/library/ prefix
		k3sImageNames = append(k3sImageNames, fmt.Sprintf("docker.io/library/%s", imageName))
	}

	var k3sHash string
	var imageFound bool
	for _, k3sImageName := range k3sImageNames {
		k3sInspectCmd := exec.Command("sudo", "k3s", "ctr", "images", "inspect", k3sImageName)
		k3sOutput, err := k3sInspectCmd.Output()
		if err == nil {
			imageFound = true
			// Image exists in k3s, try to parse JSON to get image ID
			var k3sInspect map[string]interface{}
			if err := json.Unmarshal(k3sOutput, &k3sInspect); err == nil {
				// Get the image ID from k3s inspect output (JSON format)
				// The image ID is typically in the "config" or "target" field
				if target, ok := k3sInspect["target"].(map[string]interface{}); ok {
					if digest, ok := target["digest"].(string); ok {
						k3sHash = strings.TrimPrefix(digest, "sha256:")
					}
				}
				// Also try to get from config digest
				if k3sHash == "" {
					if config, ok := k3sInspect["config"].(map[string]interface{}); ok {
						if digest, ok := config["digest"].(string); ok {
							k3sHash = strings.TrimPrefix(digest, "sha256:")
						}
					}
				}
			} else {
				// If JSON parsing fails, try to extract config digest from text output
				// k3s inspect output format: "...application/vnd.oci.image.config.v1+json @sha256:... (bytes)"
				outputStr := string(k3sOutput)
				if strings.Contains(outputStr, "application/vnd.oci.image.config.v1+json") {
					// Try to extract sha256:... from the config line
					lines := strings.Split(outputStr, "\n")
					for _, line := range lines {
						if strings.Contains(line, "application/vnd.oci.image.config.v1+json") && strings.Contains(line, "@sha256:") {
							parts := strings.Split(line, "@sha256:")
							if len(parts) > 1 {
								hashPart := strings.Fields(parts[1])[0]
								k3sHash = strings.TrimPrefix(hashPart, "sha256:")
								break
							}
						}
					}
				}
			}
			break
		}
	}

	if imageFound {
		// Compare hashes - if they match, skip import
		if k3sHash != "" && dockerHash == k3sHash {
			fmt.Printf("Image %s already exists in k3s with same ID, skipping import\n", imageName)
			return nil
		}
		fmt.Printf("Image %s exists in k3s but ID differs, re-importing...\n", imageName)
	} else {
		fmt.Printf("Image %s not found in k3s, importing...\n", imageName)
	}

	fmt.Printf("Importing image %s to k3s...\n", imageName)

	// Save docker image to tar
	// Use a temp file or specific name
	tarFile := fmt.Sprintf("%s.tar", strings.ReplaceAll(imageName, ":", "_"))
	saveCmd := exec.Command("docker", "save", "-o", tarFile, imageName)
	if out, err := saveCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to save docker image: %s: %w", string(out), err)
	}
	defer os.Remove(tarFile)

	// Import into k3s
	// sudo k3s ctr images import
	importCmd := exec.Command("sudo", "k3s", "ctr", "images", "import", tarFile)
	if out, err := importCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to import image into k3s: %s: %w", string(out), err)
	}

	fmt.Printf("Successfully imported %s to k3s\n", imageName)
	return nil
}

// calculateDeploymentOrder calculates the order in which services should be deployed
// based on their dependencies (edges). Services with no dependencies come first.
// Returns services in topological order: backends first, then frontends.
func calculateDeploymentOrder(config *BenchmarkConfig, allServices []string) []string {
	// Build dependency map: service -> list of services it depends on
	dependencies := make(map[string]map[string]bool)
	for _, serviceName := range allServices {
		dependencies[serviceName] = make(map[string]bool)
	}

	// Process edges: if A -> B, then A depends on B
	for _, edge := range config.Edges {
		if _, ok := dependencies[edge.From]; ok {
			dependencies[edge.From][edge.To] = true
		}
	}

	// Topological sort: services with no dependencies first
	var ordered []string
	visited := make(map[string]bool)
	tempMark := make(map[string]bool)

	var visit func(string)
	visit = func(service string) {
		if tempMark[service] {
			return // Cycle detected, skip
		}
		if visited[service] {
			return
		}
		tempMark[service] = true
		// Visit dependencies first
		for dep := range dependencies[service] {
			visit(dep)
		}
		tempMark[service] = false
		visited[service] = true
		ordered = append(ordered, service)
	}

	// Visit all services
	for _, service := range allServices {
		if !visited[service] {
			visit(service)
		}
	}

	return ordered
}

// waitForPodReady waits for a pod to be in Ready state
func waitForPodReady(kubectlCmd, kubeconfig, namespace, podName string) error {
	maxWait := 120     // Maximum wait time in seconds
	checkInterval := 1 // Check every second

	for i := 0; i < maxWait; i++ {
		var cmd *exec.Cmd
		if kubectlCmd == "microk8s" {
			cmd = exec.Command("sudo", "microk8s", "kubectl", "get", "pod", podName, "-n", namespace, "-o", "jsonpath={.status.phase}")
		} else if kubectlCmd == "k3s" {
			cmd = exec.Command("sudo", "k3s", "kubectl", "get", "pod", podName, "-n", namespace, "-o", "jsonpath={.status.phase}")
		} else {
			cmd = exec.Command("kubectl", "get", "pod", podName, "-n", namespace, "-o", "jsonpath={.status.phase}")
			if kubeconfig != "" {
				cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", kubeconfig))
			}
		}

		output, err := cmd.Output()
		if err == nil {
			phase := strings.TrimSpace(string(output))
			if phase == "Running" {
				// Check if all containers are ready
				var readyCmd *exec.Cmd
				if kubectlCmd == "microk8s" {
					readyCmd = exec.Command("sudo", "microk8s", "kubectl", "get", "pod", podName, "-n", namespace, "-o", "jsonpath={.status.containerStatuses[*].ready}")
				} else if kubectlCmd == "k3s" {
					readyCmd = exec.Command("sudo", "k3s", "kubectl", "get", "pod", podName, "-n", namespace, "-o", "jsonpath={.status.containerStatuses[*].ready}")
				} else {
					readyCmd = exec.Command("kubectl", "get", "pod", podName, "-n", namespace, "-o", "jsonpath={.status.containerStatuses[*].ready}")
					if kubeconfig != "" {
						readyCmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", kubeconfig))
					}
				}

				readyOutput, err := readyCmd.Output()
				if err == nil {
					readyStatuses := strings.TrimSpace(string(readyOutput))
					// Check if all containers report "true"
					statuses := strings.Fields(readyStatuses)
					allReady := true
					for _, status := range statuses {
						if status != "true" {
							allReady = false
							break
						}
					}
					if allReady && len(statuses) > 0 {
						return nil // Pod is ready
					}
				}
			} else if phase == "Failed" || phase == "Error" {
				return fmt.Errorf("pod %s is in %s state", podName, phase)
			}
		}

		time.Sleep(time.Duration(checkInterval) * time.Second)
	}

	return fmt.Errorf("timeout waiting for pod %s to be ready", podName)
}

// importSidecarImageToK3s imports the sidecar image from Docker to k3s if it exists
func importSidecarImageToK3s() error {
	imageName := "sidecar-sidecar:latest"

	// Check if k3s is installed
	if _, err := exec.LookPath("k3s"); err != nil {
		// k3s not found, skip import
		return nil
	}

	// Check if image exists in Docker
	checkCmd := exec.Command("docker", "images", "-q", imageName)
	output, err := checkCmd.Output()
	if err != nil || len(strings.TrimSpace(string(output))) == 0 {
		// Image doesn't exist in Docker, skip import
		fmt.Printf("Sidecar image %s not found in Docker, skipping import\n", imageName)
		return nil
	}

	// Import the image to k3s
	return importImageToK3s(imageName)
}
