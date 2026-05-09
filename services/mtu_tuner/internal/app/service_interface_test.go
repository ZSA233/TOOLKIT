package app

import (
	"context"
	"strings"
	"testing"

	"mtu-tuner/internal/core"
	"mtu-tuner/internal/infra/clash"
	"mtu-tuner/internal/infra/netiface"

	"toolkit/libs/appkit/cmdexec"
)

type runnerStub struct {
	results []cmdexec.Result
}

func (stub *runnerStub) Run(_ context.Context, _ []string, _ cmdexec.Options) (cmdexec.Result, error) {
	if len(stub.results) == 0 {
		return cmdexec.Result{}, nil
	}
	result := stub.results[0]
	stub.results = stub.results[1:]
	return result, nil
}

func TestDetectInterfaceIncludesCandidateInterfaces(t *testing.T) {
	runner := &runnerStub{
		results: []cmdexec.Result{
			{
				ExitCode: 0,
				Stdout: `   route to: 1.1.1.1
interface: en0
gateway: 192.168.0.1
`,
			},
			{
				ExitCode: 0,
				Stdout: `en0: flags=8863<UP,BROADCAST,SMART,RUNNING,SIMPLEX,MULTICAST> mtu 1500
	inet 192.168.0.23 netmask 0xffffff00 broadcast 192.168.0.255
`,
			},
			{
				ExitCode: 0,
				Stdout: `en0: flags=8863<UP,BROADCAST,SMART,RUNNING,SIMPLEX,MULTICAST> mtu 1500
	inet 192.168.0.23 netmask 0xffffff00 broadcast 192.168.0.255
`,
			},
			{
				ExitCode: 0,
				Stdout: `en0: flags=8863<UP,BROADCAST,SMART,RUNNING,SIMPLEX,MULTICAST> mtu 1500
	inet 192.168.0.23 netmask 0xffffff00 broadcast 192.168.0.255
en1: flags=8863<UP,BROADCAST,SMART,RUNNING,SIMPLEX,MULTICAST> mtu 1460
	inet 10.0.0.12 netmask 0xffffff00 broadcast 10.0.0.255
`,
			},
		},
	}

	service := &Service{
		goos:             "darwin",
		netiface:         netiface.New("darwin", runner),
		clash:            clash.New("darwin", nil),
		originalMTUByKey: map[string]int{},
	}

	result, err := service.DetectInterface(context.Background(), core.DetectRequest{Probe: "1.1.1.1"})
	if err != nil {
		t.Fatalf("DetectInterface() error = %v", err)
	}
	if result.Interface.Name != "en0" {
		t.Fatalf("DetectInterface() interface = %q, want en0", result.Interface.Name)
	}
	if result.OriginalMTU != 1500 {
		t.Fatalf("DetectInterface() original MTU = %d, want 1500", result.OriginalMTU)
	}
	if len(result.Candidates) != 2 {
		t.Fatalf("DetectInterface() candidates len = %d, want 2", len(result.Candidates))
	}
	if result.Candidates[0].Name != "en0" {
		t.Fatalf("DetectInterface() first candidate = %q, want en0", result.Candidates[0].Name)
	}
}

func TestDetectInterfacePrefersUnderlyingHardwareInterfaceForVirtualRoute(t *testing.T) {
	runner := &runnerStub{
		results: []cmdexec.Result{
			{
				ExitCode: 0,
				Stdout:   `{"platform":"Windows","name":"Meta","index":"44","mtu":9000,"gateway":"198.18.0.1","local_address":"198.18.0.2","description":"Clash Meta Wintun"}`,
			},
			{
				ExitCode: 0,
				Stdout:   `[{"platform":"Windows","name":"Meta","index":"44","mtu":9000,"gateway":"198.18.0.1","local_address":"198.18.0.2","description":"Clash Meta Wintun"},{"platform":"Windows","name":"Wi-Fi","index":"12","mtu":1500,"gateway":"192.168.1.1","local_address":"192.168.1.25","description":"Intel(R) Wi-Fi 6E AX211"}]`,
			},
			{
				ExitCode: 0,
				Stdout:   `{"platform":"Windows","name":"Wi-Fi","index":"12","mtu":1500,"gateway":"192.168.1.1","local_address":"192.168.1.25","description":"Intel(R) Wi-Fi 6E AX211"}`,
			},
		},
	}

	service := &Service{
		goos:             "windows",
		netiface:         netiface.New("windows", runner),
		clash:            clash.New("windows", nil),
		originalMTUByKey: map[string]int{},
	}

	result, err := service.DetectInterface(context.Background(), core.DetectRequest{Probe: "1.1.1.1"})
	if err != nil {
		t.Fatalf("DetectInterface() error = %v", err)
	}
	if result.Interface.Name != "Wi-Fi" {
		t.Fatalf("DetectInterface() interface = %q, want Wi-Fi", result.Interface.Name)
	}
	if result.OriginalMTU != 1500 {
		t.Fatalf("DetectInterface() original MTU = %d, want 1500", result.OriginalMTU)
	}
	if len(result.Candidates) != 2 {
		t.Fatalf("DetectInterface() candidates len = %d, want 2", len(result.Candidates))
	}
	if result.Candidates[0].Name != "Wi-Fi" {
		t.Fatalf("DetectInterface() first candidate = %q, want Wi-Fi", result.Candidates[0].Name)
	}
	if !strings.Contains(result.Selection.Warning, "Meta") || !strings.Contains(result.Selection.Warning, "Wi-Fi") {
		t.Fatalf("DetectInterface() warning = %q, want Meta -> Wi-Fi explanation", result.Selection.Warning)
	}
}

func TestRefreshInterfaceRecordsOriginalMTUForManualSelection(t *testing.T) {
	runner := &runnerStub{
		results: []cmdexec.Result{{
			ExitCode: 0,
			Stdout: `en1: flags=8863<UP,BROADCAST,SMART,RUNNING,SIMPLEX,MULTICAST> mtu 1460
	inet 10.0.0.12 netmask 0xffffff00 broadcast 10.0.0.255
`,
		}},
	}

	service := &Service{
		goos:             "darwin",
		netiface:         netiface.New("darwin", runner),
		clash:            clash.New("darwin", nil),
		originalMTUByKey: map[string]int{},
	}

	info := core.InterfaceInfo{
		PlatformName: "Darwin",
		Name:         "en1",
		Index:        "8",
		LocalAddress: "10.0.0.12",
		Description:  "USB Ethernet",
	}
	result, err := service.RefreshInterface(context.Background(), info)
	if err != nil {
		t.Fatalf("RefreshInterface() error = %v", err)
	}
	if result.Interface.MTU != 1460 {
		t.Fatalf("RefreshInterface() mtu = %d, want 1460", result.Interface.MTU)
	}
	if result.OriginalMTU != 1460 {
		t.Fatalf("RefreshInterface() original MTU = %d, want 1460", result.OriginalMTU)
	}
	if original := service.originalMTU(info); original != 1460 {
		t.Fatalf("service.originalMTU() = %d, want 1460", original)
	}
}
