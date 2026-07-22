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

	backendSidecar := byService("sidecar", "backend")
	if backendSidecar.AppCPULimit != 2 || backendSidecar.SidecarCPULimit != 2 ||
		backendSidecar.Replicas != 1 || backendSidecar.GOMAXPROCS != 2 ||
		backendSidecar.CPUCount != 2 {
		t.Fatalf("sidecar backend: %+v", backendSidecar)
	}

	backendLb := byService("sidecar-lb", "backend")
	if backendLb.AppCPULimit != 1 || backendLb.SidecarCPULimit != 0.5 ||
		backendLb.Replicas != 2 || backendLb.GOMAXPROCS != 1 ||
		backendLb.CPUCount != 1 {
		t.Fatalf("sidecar-lb backend: %+v", backendLb)
	}

	plainLb := byService("plain-lb", "backend")
	if plainLb.AppCPULimit != 1 || plainLb.Replicas != 2 || plainLb.GOMAXPROCS != 1 {
		t.Fatalf("plain-lb backend: %+v", plainLb)
	}
	if plainLb.SidecarCPULimit >= 0 {
		t.Fatalf("plain-lb should have no sidecar limit")
	}
	plainLbIngress := byService("plain-lb", "ingress")
	if plainLbIngress.SidecarCPULimit != 2 {
		t.Fatalf("plain-lb ingress: %+v", plainLbIngress)
	}

	ingressSidecar := byService("sidecar", "ingress")
	if ingressSidecar.SidecarCPULimit != 2 {
		t.Fatalf("sidecar ingress: %+v", ingressSidecar)
	}
	ingressLb := byService("sidecar-lb", "ingress")
	if ingressLb.SidecarCPULimit != 2 {
		t.Fatalf("sidecar-lb ingress: %+v", ingressLb)
	}
}
