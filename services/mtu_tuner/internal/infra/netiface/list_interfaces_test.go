package netiface

import (
	"context"
	"testing"

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

func TestListInterfacesWindowsNormalizesResults(t *testing.T) {
	service := New("windows", &runnerStub{
		results: []cmdexec.Result{{
			ExitCode: 0,
			Stdout:   `[{"platform":"Windows","name":"Ethernet","index":"12","mtu":1500,"gateway":"10.0.0.1","local_address":"10.0.0.8","description":"Intel NIC"},{"platform":"Windows","name":"Ethernet","index":"12","mtu":1500,"gateway":"10.0.0.1","local_address":"10.0.0.8","description":""},{"platform":"Windows","name":"NoIPv4","index":"13","mtu":1500,"gateway":"","local_address":"","description":""}]`,
		}},
	})

	interfaces, err := service.ListInterfaces(context.Background())
	if err != nil {
		t.Fatalf("ListInterfaces() error = %v", err)
	}
	if len(interfaces) != 1 {
		t.Fatalf("ListInterfaces() len = %d, want 1", len(interfaces))
	}
	if interfaces[0].Name != "Ethernet" {
		t.Fatalf("ListInterfaces() name = %q, want Ethernet", interfaces[0].Name)
	}
	if interfaces[0].Description != "Intel NIC" {
		t.Fatalf("ListInterfaces() description = %q, want Intel NIC", interfaces[0].Description)
	}
}

func TestListInterfacesDarwinFiltersLoopbackAndRequiresIPv4(t *testing.T) {
	service := New("darwin", &runnerStub{
		results: []cmdexec.Result{{
			ExitCode: 0,
			Stdout: `lo0: flags=8049<UP,LOOPBACK,RUNNING,MULTICAST> mtu 16384
	inet 127.0.0.1 netmask 0xff000000
en0: flags=8863<UP,BROADCAST,SMART,RUNNING,SIMPLEX,MULTICAST> mtu 1500
	inet 192.168.0.23 netmask 0xffffff00 broadcast 192.168.0.255
en5: flags=8822<BROADCAST,SMART,SIMPLEX,MULTICAST> mtu 1500
	inet 10.0.0.5 netmask 0xffffff00 broadcast 10.0.0.255
utun4: flags=8051<UP,POINTOPOINT,RUNNING,MULTICAST> mtu 1380
	inet6 fe80::1%utun4 prefixlen 64 scopeid 0x14
`,
		}},
	})

	interfaces, err := service.ListInterfaces(context.Background())
	if err != nil {
		t.Fatalf("ListInterfaces() error = %v", err)
	}
	if len(interfaces) != 1 {
		t.Fatalf("ListInterfaces() len = %d, want 1", len(interfaces))
	}
	if interfaces[0].Name != "en0" {
		t.Fatalf("ListInterfaces() name = %q, want en0", interfaces[0].Name)
	}
	if interfaces[0].LocalAddress != "192.168.0.23" {
		t.Fatalf("ListInterfaces() local = %q, want 192.168.0.23", interfaces[0].LocalAddress)
	}
}

func TestListInterfacesLinuxFiltersLoopbackAndMissingIPv4(t *testing.T) {
	service := New("linux", &runnerStub{
		results: []cmdexec.Result{{
			ExitCode: 0,
			Stdout: `[
  {"ifname":"lo","mtu":65536,"link_type":"loopback","operstate":"UNKNOWN","addr_info":[{"family":"inet","local":"127.0.0.1"}]},
  {"ifname":"eth0","mtu":1500,"link_type":"ether","operstate":"UP","addr_info":[{"family":"inet","local":"192.168.1.30"}]},
  {"ifname":"wg0","mtu":1420,"link_type":"none","operstate":"UP","addr_info":[{"family":"inet6","local":"fe80::1"}]}
]`,
		}},
	})

	interfaces, err := service.ListInterfaces(context.Background())
	if err != nil {
		t.Fatalf("ListInterfaces() error = %v", err)
	}
	if len(interfaces) != 1 {
		t.Fatalf("ListInterfaces() len = %d, want 1", len(interfaces))
	}
	if interfaces[0].Name != "eth0" {
		t.Fatalf("ListInterfaces() name = %q, want eth0", interfaces[0].Name)
	}
	if interfaces[0].MTU != 1500 {
		t.Fatalf("ListInterfaces() mtu = %d, want 1500", interfaces[0].MTU)
	}
}
