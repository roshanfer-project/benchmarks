package gen

import "math"

// ServiceDeploySpec is the deploy resource picture for one (mode, service) pair.
type ServiceDeploySpec struct {
	Mode            string
	Service         string
	AppCPULimit     float64 // negative = N/A
	SidecarCPULimit float64 // negative = N/A
	Replicas        int
	GOMAXPROCS      int // negative = N/A
	CPUCount        int // negative = N/A (sidecar runtime)
	OverCommitment  float64
	NumThreads      int // negative = N/A
	HasOverCommit   bool
}

const na = -1.0

func sidecarRuntimeCPUCount(pg *ParsedGraph, svc string, lb bool) int {
	cpu := pg.CPUForService(svc)
	if lb {
		replicas := pg.ReplicasForService(svc)
		return GOMAXPROCSForPerReplicaCPU(PerReplicaCPU(cpu, replicas))
	}
	return int(cpu)
}

func plainPodAppSpec(pg *ParsedGraph, mode, svc string) ServiceDeploySpec {
	cpu := pg.CPUForService(svc)
	appCPU := float64(int(cpu))
	gmp := int(cpu)
	return ServiceDeploySpec{
		Mode:           mode,
		Service:        svc,
		AppCPULimit:    appCPU,
		SidecarCPULimit: na,
		Replicas:       1,
		GOMAXPROCS:     gmp,
		CPUCount:       int(na),
		NumThreads:     int(na),
	}
}

func lbAppSpec(pg *ParsedGraph, mode, svc string) ServiceDeploySpec {
	cpu := pg.CPUForService(svc)
	replicas := pg.ReplicasForService(svc)
	perCPU := PerReplicaCPU(cpu, replicas)
	gmp := GOMAXPROCSForPerReplicaCPU(perCPU)
	return ServiceDeploySpec{
		Mode:           mode,
		Service:        svc,
		AppCPULimit:    perCPU,
		SidecarCPULimit: na,
		Replicas:       replicas,
		GOMAXPROCS:     gmp,
		CPUCount:       int(na),
		NumThreads:     int(na),
	}
}

func sidecarWorkloadSpec(pg *ParsedGraph, mode, svc string, lb bool) ServiceDeploySpec {
	cpu := pg.CPUForService(svc)
	replicas := pg.ReplicasForService(svc)
	sidecarCPU := pg.SidecarCPUForService(svc)
	oc := pg.OverCommitmentForService(svc)
	threads := sidecarCPU

	var appCPU, sidecarLimit float64
	var gmp, reps int
	if lb {
		perCPU := PerReplicaCPU(cpu, replicas)
		appCPU = perCPU
		sidecarLimit = float64(sidecarCPU) * 0.5
		gmp = GOMAXPROCSForPerReplicaCPU(perCPU)
		reps = replicas
	} else {
		appCPU = float64(int(cpu))
		sidecarLimit = float64(sidecarCPU * 2)
		gmp = int(cpu)
		reps = 1
	}
	return ServiceDeploySpec{
		Mode:            mode,
		Service:         svc,
		AppCPULimit:     appCPU,
		SidecarCPULimit: sidecarLimit,
		Replicas:        reps,
		GOMAXPROCS:      gmp,
		CPUCount:        sidecarRuntimeCPUCount(pg, svc, lb),
		OverCommitment:  oc,
		HasOverCommit:   true,
		NumThreads:      threads,
	}
}

func envoyWorkloadSpec(pg *ParsedGraph, mode, svc string) ServiceDeploySpec {
	cpu := pg.CPUForService(svc)
	appCPU := float64(int(cpu))
	envoyCPU := float64(pg.SidecarCPUForService(svc))
	return ServiceDeploySpec{
		Mode:            mode,
		Service:         svc,
		AppCPULimit:     appCPU,
		SidecarCPULimit: envoyCPU,
		Replicas:        1,
		GOMAXPROCS:      int(cpu),
		CPUCount:        int(na),
		NumThreads:      int(na),
	}
}

func ingressSidecarSpec(pg *ParsedGraph, mode string, _ bool) ServiceDeploySpec {
	limit := float64(pg.UserEntryCount() * 2)
	return ServiceDeploySpec{
		Mode:            mode,
		Service:         "ingress",
		AppCPULimit:     na,
		SidecarCPULimit: limit,
		Replicas:        1,
		GOMAXPROCS:      int(na),
		CPUCount:        int(na),
		NumThreads:      int(na),
	}
}

func ingressEnvoySpec(mode string, concurrency int) ServiceDeploySpec {
	return ServiceDeploySpec{
		Mode:            mode,
		Service:         "ingress",
		AppCPULimit:     na,
		SidecarCPULimit: float64(concurrency),
		Replicas:        1,
		GOMAXPROCS:      int(na),
		CPUCount:        int(na),
		NumThreads:      int(na),
	}
}

func rajomonClientSpec(pg *ParsedGraph, mode string, lb bool) ServiceDeploySpec {
	entryMS := pg.EntryMicroservice()
	entryCPU := pg.CPUForService(entryMS)
	if lb {
		entryReplicas := pg.ReplicasForService(entryMS)
		perCPU := PerReplicaCPU(entryCPU, entryReplicas)
		gmp := GOMAXPROCSForPerReplicaCPU(perCPU)
		return ServiceDeploySpec{
			Mode:           mode,
			Service:        "rajomon-client",
			AppCPULimit:    perCPU,
			SidecarCPULimit: na,
			Replicas:       entryReplicas,
			GOMAXPROCS:     gmp,
			CPUCount:       int(na),
			NumThreads:     int(na),
		}
	}
	rcCPU := float64(int(entryCPU) + 1)
	return ServiceDeploySpec{
		Mode:           mode,
		Service:        "rajomon-client",
		AppCPULimit:    rcCPU,
		SidecarCPULimit: na,
		Replicas:       1,
		GOMAXPROCS:     int(rcCPU),
		CPUCount:       int(na),
		NumThreads:     int(na),
	}
}

// DeploySpecForMode returns deploy specs for all workloads in a deploy mode.
func DeploySpecForMode(pg *ParsedGraph, mode string) []ServiceDeploySpec {
	svcNames := sortedServices(pg)
	var specs []ServiceDeploySpec

	switch mode {
	case "plain":
		for _, svc := range svcNames {
			specs = append(specs, plainPodAppSpec(pg, mode, svc))
		}
	case "p2c", "wrr":
		for _, svc := range svcNames {
			specs = append(specs, lbAppSpec(pg, mode, svc))
		}
		specs = append(specs, ingressEnvoySpec(mode, pg.UserEntryCount()*2))
	case "sidecar":
		for _, svc := range svcNames {
			specs = append(specs, sidecarWorkloadSpec(pg, mode, svc, false))
		}
		specs = append(specs, ingressSidecarSpec(pg, mode, false))
	case "approx", "approx-fcfs", "approx-edf":
		for _, svc := range svcNames {
			specs = append(specs, sidecarWorkloadSpec(pg, mode, svc, true))
		}
		specs = append(specs, ingressSidecarSpec(pg, mode, true))
	case "envoy":
		for _, svc := range svcNames {
			specs = append(specs, envoyWorkloadSpec(pg, mode, svc))
		}
		specs = append(specs, ingressEnvoySpec(mode, envoyIngressConcurrency))
	case "rajomon", "dagor":
		for _, svc := range svcNames {
			specs = append(specs, plainPodAppSpec(pg, mode, svc))
		}
		specs = append(specs, rajomonClientSpec(pg, mode, false))
	case "rajomon-lb", "dagor-lb":
		for _, svc := range svcNames {
			specs = append(specs, lbAppSpec(pg, mode, svc))
		}
		specs = append(specs, rajomonClientSpec(pg, mode, true))
	}
	return specs
}

var allDeployModes = []string{
	"plain", "p2c", "wrr", "sidecar", "approx", "approx-fcfs", "approx-edf", "envoy",
	"rajomon", "rajomon-lb", "dagor", "dagor-lb",
}

// ModeClusterTotals sums allocated cores for a mode.
func ModeClusterTotals(specs []ServiceDeploySpec) (totalApp, totalSidecar float64) {
	for _, s := range specs {
		if s.AppCPULimit >= 0 {
			totalApp += s.AppCPULimit * float64(s.Replicas)
		}
		if s.SidecarCPULimit >= 0 {
			totalSidecar += s.SidecarCPULimit * float64(s.Replicas)
		}
	}
	return totalApp, totalSidecar
}

func appSidecarRatio(totalApp, totalSidecar float64) float64 {
	if totalSidecar <= 0 {
		return math.NaN()
	}
	return totalApp / totalSidecar
}
