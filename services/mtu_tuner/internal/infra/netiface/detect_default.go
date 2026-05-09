package netiface

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"mtu-tuner/internal/core"

	"toolkit/libs/appkit/cmdexec"
)

func (service *Service) DetectDefaultInterface(ctx context.Context) (core.InterfaceInfo, error) {
	switch service.goos {
	case "windows":
		return service.detectWindowsDefault(ctx)
	case "darwin":
		return service.detectDarwinDefault(ctx)
	case "linux":
		return service.detectLinuxDefault(ctx)
	default:
		return core.InterfaceInfo{}, fmt.Errorf("unsupported platform: %s", service.goos)
	}
}

func (service *Service) detectWindowsDefault(ctx context.Context) (core.InterfaceInfo, error) {
	script := `
$rows = @()
$ipifs = Get-NetIPInterface -AddressFamily IPv4 | Where-Object { $_.NlMtu -gt 0 -and $_.InterfaceAlias -notmatch 'Loopback' }
foreach ($ipif in $ipifs) {
  $adapter = Get-NetAdapter -InterfaceIndex $ipif.InterfaceIndex -ErrorAction SilentlyContinue
  if (-not $adapter -or $adapter.Status -ne 'Up' -or -not $adapter.HardwareInterface) { continue }
  $ip = Get-NetIPAddress -AddressFamily IPv4 -InterfaceIndex $ipif.InterfaceIndex -ErrorAction SilentlyContinue |
    Where-Object { $_.IPAddress -and $_.IPAddress -notlike '169.254*' } |
    Select-Object -First 1
  if (-not $ip) { continue }
  $route = Get-NetRoute -InterfaceIndex $ipif.InterfaceIndex -AddressFamily IPv4 -DestinationPrefix '0.0.0.0/0' -ErrorAction SilentlyContinue |
    Where-Object { $_.NextHop } |
    Sort-Object @{ Expression = { [int]$_.RouteMetric + [int]$ipif.InterfaceMetric } } |
    Select-Object -First 1
  if (-not $route) { continue }
  $rows += [pscustomobject]@{
    platform='Windows'
    name=[string]$ipif.InterfaceAlias
    index=[string]$ipif.InterfaceIndex
    mtu=[int]$ipif.NlMtu
    gateway=[string]$route.NextHop
    local_address=[string]$ip.IPAddress
    description=[string]$adapter.InterfaceDescription
    score=([int]$route.RouteMetric + [int]$ipif.InterfaceMetric)
  }
}
$item = $rows | Sort-Object score | Select-Object -First 1
if (-not $item) { throw 'No default hardware interface found' }
$item | Select-Object platform,name,index,mtu,gateway,local_address,description | ConvertTo-Json -Compress
`

	timeoutCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	result, err := cmdexec.RunPowerShell(timeoutCtx, service.runner, script)
	if err != nil {
		return core.InterfaceInfo{}, err
	}
	if result.ExitCode != 0 {
		return core.InterfaceInfo{}, commandResultError(result.Stdout, result.Stderr)
	}
	return decodeInterfaceJSON(result.Stdout)
}

func (service *Service) detectDarwinDefault(ctx context.Context) (core.InterfaceInfo, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	result, err := service.runner.Run(timeoutCtx, []string{"route", "-n", "get", "default"}, cmdexec.Options{})
	if err != nil {
		return core.InterfaceInfo{}, err
	}
	if result.ExitCode != 0 {
		return core.InterfaceInfo{}, commandResultError(result.Stdout, result.Stderr)
	}
	routeFields := parseRouteFields(result.Stdout)
	iface := routeFields["interface"]
	if iface == "" {
		return core.InterfaceInfo{}, fmt.Errorf("could not parse macOS default interface")
	}
	mtu, err := service.currentDarwin(ctx, core.InterfaceInfo{Name: iface})
	if err != nil {
		return core.InterfaceInfo{}, err
	}
	ifconfigCtx, cancelIfconfig := context.WithTimeout(ctx, 10*time.Second)
	defer cancelIfconfig()
	ifconfigResult, _ := service.runner.Run(ifconfigCtx, []string{"ifconfig", iface}, cmdexec.Options{})
	return core.InterfaceInfo{
		PlatformName: "Darwin",
		Name:         iface,
		MTU:          mtu,
		Gateway:      routeFields["gateway"],
		LocalAddress: firstIPv4FromIfconfig(ifconfigResult.Stdout),
		Description:  iface,
	}, nil
}

func (service *Service) detectLinuxDefault(ctx context.Context) (core.InterfaceInfo, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	result, err := service.runner.Run(timeoutCtx, []string{"ip", "-j", "route", "show", "default"}, cmdexec.Options{})
	if err != nil {
		return core.InterfaceInfo{}, err
	}
	if result.ExitCode != 0 {
		return core.InterfaceInfo{}, commandResultError(result.Stdout, result.Stderr)
	}
	var payload []struct {
		Device    string `json:"dev"`
		Gateway   string `json:"gateway"`
		Preferred string `json:"prefsrc"`
		Metric    int    `json:"metric"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &payload); err != nil {
		return core.InterfaceInfo{}, fmt.Errorf("decode linux default route response: %w", err)
	}
	best := -1
	for index, item := range payload {
		if strings.TrimSpace(item.Device) == "" {
			continue
		}
		if best == -1 || item.Metric < payload[best].Metric {
			best = index
		}
	}
	if best == -1 {
		return core.InterfaceInfo{}, fmt.Errorf("could not parse linux default interface")
	}
	info := core.InterfaceInfo{
		PlatformName: "Linux",
		Name:         payload[best].Device,
		Gateway:      payload[best].Gateway,
		LocalAddress: payload[best].Preferred,
		Description:  payload[best].Device,
	}
	mtu, err := service.currentLinux(ctx, info)
	if err != nil {
		return core.InterfaceInfo{}, err
	}
	info.MTU = mtu
	return info, nil
}

func decodeInterfaceJSON(raw string) (core.InterfaceInfo, error) {
	var payload struct {
		Platform     string `json:"platform"`
		Name         string `json:"name"`
		Index        string `json:"index"`
		MTU          int    `json:"mtu"`
		Gateway      string `json:"gateway"`
		LocalAddress string `json:"local_address"`
		Description  string `json:"description"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return core.InterfaceInfo{}, fmt.Errorf("decode interface result: %w", err)
	}
	return core.InterfaceInfo{
		PlatformName: payload.Platform,
		Name:         payload.Name,
		Index:        payload.Index,
		MTU:          payload.MTU,
		Gateway:      payload.Gateway,
		LocalAddress: payload.LocalAddress,
		Description:  payload.Description,
	}, nil
}
