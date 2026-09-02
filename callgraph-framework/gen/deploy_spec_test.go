package gen

import (
	"path/filepath"
	"testing"
)

func TestDeploySpecChain2SidecarModes(t *testing.T) {
	pg, err := ParseCallGraph(filepath.Join("..", "..", "tests", "chain-2", "callgraph.json"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	byService := func(mode, svc string) ServiceDeploySpec {
		for _, s := range DeploySpecForMode(pg, mode) {
			if s.Service == svc {
				return s
			}
		}
		t.Fatalf("missing %s/%s", mode, svc)
		return ServiceDeploySpec{}
	}

	backendSidecar := byService("roshanfer", "backend")
	if backendSidecar.AppCPULimit != 2 || backendSidecar.SidecarCPULimit != 2 ||
		backendSidecar.Replicas != 1 || backendSidecar.GOMAXPROCS != 2 ||
		backendSidecar.CPUCount != 2 {
		t.Fatalf("roshanfer backend: %+v", backendSidecar)
	}

	backendLb := byService("approx", "backend")
	if backendLb.AppCPULimit != 1 || backendLb.SidecarCPULimit != 1 ||
		backendLb.Replicas != 2 || backendLb.GOMAXPROCS != 1 ||
		backendLb.CPUCount != 1 {
		t.Fatalf("approx backend: %+v", backendLb)
	}

	for _, mode := range []string{"p2c", "wrr"} {
		lb := byService(mode, "backend")
		if lb.AppCPULimit != 1 || lb.Replicas != 2 || lb.GOMAXPROCS != 1 {
			t.Fatalf("%s backend: %+v", mode, lb)
		}
		if lb.SidecarCPULimit >= 0 {
			t.Fatalf("%s should have no sidecar limit", mode)
		}
		ingress := byService(mode, "ingress")
		if ingress.SidecarCPULimit != 2 {
			t.Fatalf("%s ingress: %+v", mode, ingress)
		}
	}

	ingressSidecar := byService("roshanfer", "ingress")
	if ingressSidecar.SidecarCPULimit != 2 {
		t.Fatalf("roshanfer ingress: %+v", ingressSidecar)
	}
	ingressLb := byService("approx", "ingress")
	if ingressLb.SidecarCPULimit != 2 {
		t.Fatalf("approx ingress: %+v", ingressLb)
	}
}
