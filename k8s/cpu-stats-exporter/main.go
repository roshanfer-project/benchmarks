package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ContainerStats holds raw CPU stats for a container
type ContainerStats struct {
	ContainerID      string `json:"container_id"`
	CPUUsageNanosecs int64  `json:"cpu_usage_nanoseconds"`
}

// MetricsResponse is the JSON response structure
type MetricsResponse struct {
	Timestamp  string           `json:"timestamp"`
	Containers []ContainerStats `json:"containers"`
}

var (
	metricsCache MetricsResponse
	cacheMutex   sync.RWMutex
	nodeName     string
)

func main() {
	nodeName = os.Getenv("NODE_NAME")
	if nodeName == "" {
		log.Fatal("NODE_NAME environment variable must be set")
	}

	// Start background updater
	go updateLoop()

	// HTTP server
	http.HandleFunc("/metrics", metricsHandler)
	log.Printf("Starting HTTP server on :9100 for node %s", nodeName)
	if err := http.ListenAndServe(":9100", nil); err != nil {
		log.Fatal(err)
	}
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metricsCache); err != nil {
		log.Printf("Error encoding JSON: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func updateLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// Update immediately on start
	updateMetrics()

	for range ticker.C {
		updateMetrics()
	}
}

func updateMetrics() {
	containers := collectCPUStats()

	cacheMutex.Lock()
	metricsCache = MetricsResponse{
		Timestamp:  time.Now().UTC().Format(time.RFC3339Nano),
		Containers: containers,
	}
	cacheMutex.Unlock()

	log.Printf("Updated metrics: %d containers", len(containers))
}

func collectCPUStats() []ContainerStats {
	var containers []ContainerStats

	cgroupRoot := "/sys/fs/cgroup"

	// Try cgroup v2 first
	if _, err := os.Stat(filepath.Join(cgroupRoot, "cgroup.controllers")); err == nil {
		containers = collectCgroupV2(cgroupRoot)
	} else {
		// Fall back to cgroup v1
		containers = collectCgroupV1(cgroupRoot)
	}

	return containers
}

func collectCgroupV2(root string) []ContainerStats {
	var containers []ContainerStats

	// In cgroup v2, containers are typically under:
	// /sys/fs/cgroup/kubepods.slice/kubepods-<qos>.slice/kubepods-<qos>-pod<uid>.slice/cri-containerd-<container-id>.scope/
	patterns := []string{
		filepath.Join(root, "kubepods.slice/*/kubepods-*-pod*.slice/cri-containerd-*.scope"),
		filepath.Join(root, "kubepods.slice/kubepods-*-pod*.slice/cri-containerd-*.scope"),
	}

	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}

		for _, cgroupPath := range matches {
			// Extract container ID from path
			containerID := extractContainerIDV2(cgroupPath)
			if containerID == "" {
				continue
			}

			// Read CPU usage
			cpuUsage := readCPUUsageV2(cgroupPath)
			if cpuUsage == 0 {
				continue
			}

			containers = append(containers, ContainerStats{
				ContainerID:      containerID,
				CPUUsageNanosecs: cpuUsage,
			})
		}
	}

	return containers
}

func collectCgroupV1(root string) []ContainerStats {
	var containers []ContainerStats

	// In cgroup v1, CPU stats are under:
	// /sys/fs/cgroup/cpu,cpuacct/kubepods/<qos>/pod<uid>/<container-id>/cpuacct.usage
	cpuRoots := []string{
		filepath.Join(root, "cpu,cpuacct/kubepods"),
		filepath.Join(root, "cpu/kubepods"),
		filepath.Join(root, "cpuacct/kubepods"),
	}

	for _, cpuRoot := range cpuRoots {
		if _, err := os.Stat(cpuRoot); err != nil {
			continue
		}

		patterns := []string{
			filepath.Join(cpuRoot, "*/pod*/*"),
			filepath.Join(cpuRoot, "pod*/*"),
		}

		for _, pattern := range patterns {
			matches, err := filepath.Glob(pattern)
			if err != nil {
				continue
			}

			for _, cgroupPath := range matches {
				// Extract container ID from path
				containerID := extractContainerIDV1(filepath.Base(cgroupPath))
				if containerID == "" {
					continue
				}

				// Read CPU usage
				cpuUsage := readCPUUsageV1(cgroupPath)
				if cpuUsage == 0 {
					continue
				}

				containers = append(containers, ContainerStats{
					ContainerID:      containerID,
					CPUUsageNanosecs: cpuUsage,
				})
			}
		}
	}

	return containers
}

func extractContainerIDV2(cgroupPath string) string {
	// Path format: .../cri-containerd-<container-id>.scope
	base := filepath.Base(cgroupPath)
	if !strings.HasPrefix(base, "cri-containerd-") {
		return ""
	}
	id := strings.TrimPrefix(base, "cri-containerd-")
	id = strings.TrimSuffix(id, ".scope")
	return id
}

func extractContainerIDV1(basename string) string {
	// Remove runtime prefixes
	id := basename
	prefixes := []string{"crio-", "docker-", "containerd-"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(id, prefix) {
			id = strings.TrimPrefix(id, prefix)
			break
		}
	}
	return id
}

func readCPUUsageV2(cgroupPath string) int64 {
	// In cgroup v2, cpu.stat contains usage_usec
	statFile := filepath.Join(cgroupPath, "cpu.stat")
	data, err := ioutil.ReadFile(statFile)
	if err != nil {
		return 0
	}

	// Parse cpu.stat format:
	// usage_usec 123456
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "usage_usec" {
			usec, err := strconv.ParseInt(fields[1], 10, 64)
			if err == nil {
				return usec * 1000 // Convert to nanoseconds
			}
		}
	}

	return 0
}

func readCPUUsageV1(cgroupPath string) int64 {
	// In cgroup v1, cpuacct.usage contains total CPU time in nanoseconds
	usageFile := filepath.Join(cgroupPath, "cpuacct.usage")
	data, err := ioutil.ReadFile(usageFile)
	if err != nil {
		return 0
	}

	usage, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0
	}

	return usage
}
