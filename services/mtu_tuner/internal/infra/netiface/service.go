package netiface

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"time"

	"mtu-tuner/internal/core"

	"toolkit/libs/appkit/cmdexec"
)

type Service struct {
	goos   string
	runner cmdexec.Runner
}

func New(goos string, runner cmdexec.Runner) *Service {
	if strings.TrimSpace(goos) == "" {
		goos = runtime.GOOS
	}
	if runner == nil {
		runner = cmdexec.ExecRunner{}
	}
	return &Service{
		goos:   strings.ToLower(goos),
		runner: runner,
	}
}

func (service *Service) DetectInterface(ctx context.Context, probeIP string) (core.InterfaceInfo, error) {
	switch service.goos {
	case "windows":
		return service.detectWindows(ctx, probeIP)
	case "darwin":
		return service.detectDarwin(ctx, probeIP)
	case "linux":
		return service.detectLinux(ctx, probeIP)
	default:
		return core.InterfaceInfo{}, fmt.Errorf("unsupported platform: %s", service.goos)
	}
}

func (service *Service) CurrentMTU(ctx context.Context, info core.InterfaceInfo) (int, error) {
	switch service.goos {
	case "windows":
		return service.currentWindows(ctx, info)
	case "darwin":
		return service.currentDarwin(ctx, info)
	case "linux":
		return service.currentLinux(ctx, info)
	default:
		return 0, fmt.Errorf("unsupported platform: %s", service.goos)
	}
}

func (service *Service) SetMTU(ctx context.Context, info core.InterfaceInfo, mtu int, persistent bool) (string, error) {
	if err := core.ValidateMTU(mtu); err != nil {
		return "", err
	}
	switch service.goos {
	case "windows":
		return service.setWindows(ctx, info, mtu, persistent)
	case "darwin":
		return service.setDarwin(ctx, info, mtu, persistent)
	case "linux":
		return service.setLinux(ctx, info, mtu, persistent)
	default:
		return "", fmt.Errorf("unsupported platform: %s", service.goos)
	}
}

func (service *Service) detectWindows(ctx context.Context, probeIP string) (core.InterfaceInfo, error) {
	script := fmt.Sprintf(`
$routes = @(Find-NetRoute -RemoteIPAddress %s)
if (-not $routes) { throw 'No route found' }
$route = $routes | Where-Object { $_.NextHop } | Select-Object -First 1
if (-not $route) { $route = $routes | Select-Object -First 1 }
$ipif = Get-NetIPInterface -InterfaceIndex $route.InterfaceIndex -AddressFamily IPv4
$adapter = Get-NetAdapter -InterfaceIndex $route.InterfaceIndex -ErrorAction SilentlyContinue
[pscustomobject]@{
  platform='Windows'
  name=$ipif.InterfaceAlias
  index=[string]$route.InterfaceIndex
  mtu=[int]$ipif.NlMtu
  gateway=[string]$route.NextHop
  local_address=[string]$route.IPAddress
  description=[string]$adapter.InterfaceDescription
} | ConvertTo-Json -Compress
`, cmdexec.PowerShellQuote(probeIP))

	timeoutCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	result, err := cmdexec.RunPowerShell(timeoutCtx, service.runner, script)
	if err != nil {
		return core.InterfaceInfo{}, err
	}
	if result.ExitCode != 0 {
		return core.InterfaceInfo{}, commandResultError(result.Stdout, result.Stderr)
	}

	var payload struct {
		Platform     string `json:"platform"`
		Name         string `json:"name"`
		Index        string `json:"index"`
		MTU          int    `json:"mtu"`
		Gateway      string `json:"gateway"`
		LocalAddress string `json:"local_address"`
		Description  string `json:"description"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &payload); err != nil {
		return core.InterfaceInfo{}, fmt.Errorf("decode route detection result: %w", err)
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

func (service *Service) currentWindows(ctx context.Context, info core.InterfaceInfo) (int, error) {
	selector := fmt.Sprintf("-InterfaceAlias %s", cmdexec.PowerShellQuote(info.Name))
	if strings.TrimSpace(info.Index) != "" {
		selector = fmt.Sprintf("-InterfaceIndex %s", strings.TrimSpace(info.Index))
	}
	script := fmt.Sprintf("(Get-NetIPInterface %s -AddressFamily IPv4).NlMtu", selector)
	timeoutCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	result, err := cmdexec.RunPowerShell(timeoutCtx, service.runner, script)
	if err != nil {
		return 0, err
	}
	if result.ExitCode != 0 {
		return 0, commandResultError(result.Stdout, result.Stderr)
	}
	var mtu int
	if _, err := fmt.Sscanf(strings.TrimSpace(result.Stdout), "%d", &mtu); err != nil {
		return 0, fmt.Errorf("parse windows mtu: %w", err)
	}
	return mtu, nil
}

func (service *Service) setWindows(ctx context.Context, info core.InterfaceInfo, mtu int, persistent bool) (string, error) {
	store := "active"
	if persistent {
		store = "persistent"
	}
	target := strings.TrimSpace(info.Name)
	if target == "" {
		target = strings.TrimSpace(info.Index)
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	result, err := service.runner.Run(timeoutCtx, []string{
		"netsh",
		"interface",
		"ipv4",
		"set",
		"subinterface",
		"interface=" + target,
		fmt.Sprintf("mtu=%d", mtu),
		"store=" + store,
	}, cmdexec.Options{HiddenWindow: true})
	if err != nil {
		return "", err
	}
	if result.ExitCode != 0 {
		return "", commandResultError(result.Stdout, result.Stderr)
	}
	return strings.TrimSpace(result.Stdout + result.Stderr), nil
}

func (service *Service) detectDarwin(ctx context.Context, probeIP string) (core.InterfaceInfo, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	result, err := service.runner.Run(timeoutCtx, []string{"route", "-n", "get", probeIP}, cmdexec.Options{})
	if err != nil {
		return core.InterfaceInfo{}, err
	}
	if result.ExitCode != 0 {
		return core.InterfaceInfo{}, commandResultError(result.Stdout, result.Stderr)
	}
	routeFields := parseRouteFields(result.Stdout)
	iface := routeFields["interface"]
	if iface == "" {
		return core.InterfaceInfo{}, fmt.Errorf("could not parse macOS route interface")
	}
	mtu, err := service.currentDarwin(ctx, core.InterfaceInfo{Name: iface})
	if err != nil {
		return core.InterfaceInfo{}, err
	}
	ifconfigCtx, cancelIfconfig := context.WithTimeout(ctx, 10*time.Second)
	defer cancelIfconfig()
	ifconfigResult, _ := service.runner.Run(ifconfigCtx, []string{"ifconfig", iface}, cmdexec.Options{})
	localAddress := firstIPv4FromIfconfig(ifconfigResult.Stdout)
	return core.InterfaceInfo{
		PlatformName: "Darwin",
		Name:         iface,
		MTU:          mtu,
		Gateway:      routeFields["gateway"],
		LocalAddress: localAddress,
		Description:  iface,
	}, nil
}

func (service *Service) currentDarwin(ctx context.Context, info core.InterfaceInfo) (int, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	result, err := service.runner.Run(timeoutCtx, []string{"ifconfig", info.Name}, cmdexec.Options{})
	if err != nil {
		return 0, err
	}
	if result.ExitCode != 0 {
		return 0, commandResultError(result.Stdout, result.Stderr)
	}
	match := regexp.MustCompile(`\bmtu\s+(\d+)`).FindStringSubmatch(result.Stdout)
	if len(match) != 2 {
		return 0, fmt.Errorf("could not parse MTU from ifconfig")
	}
	var mtu int
	if _, err := fmt.Sscanf(match[1], "%d", &mtu); err != nil {
		return 0, fmt.Errorf("parse macOS mtu: %w", err)
	}
	return mtu, nil
}

func (service *Service) setDarwin(ctx context.Context, info core.InterfaceInfo, mtu int, persistent bool) (string, error) {
	if persistent {
		return "", fmt.Errorf("persistent MTU is not implemented for macOS")
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	result, err := service.runner.Run(timeoutCtx, []string{"ifconfig", info.Name, "mtu", fmt.Sprintf("%d", mtu)}, cmdexec.Options{})
	if err != nil {
		return "", err
	}
	if result.ExitCode != 0 {
		return "", commandResultError(result.Stdout, result.Stderr)
	}
	return strings.TrimSpace(result.Stdout), nil
}

func (service *Service) detectLinux(ctx context.Context, probeIP string) (core.InterfaceInfo, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	result, err := service.runner.Run(timeoutCtx, []string{"ip", "-j", "route", "get", probeIP}, cmdexec.Options{})
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
	}
	if err := json.Unmarshal([]byte(result.Stdout), &payload); err != nil {
		return core.InterfaceInfo{}, fmt.Errorf("decode linux route response: %w", err)
	}
	if len(payload) == 0 || payload[0].Device == "" {
		return core.InterfaceInfo{}, fmt.Errorf("could not parse linux route interface")
	}
	info := core.InterfaceInfo{
		PlatformName: "Linux",
		Name:         payload[0].Device,
		Gateway:      payload[0].Gateway,
		LocalAddress: payload[0].Preferred,
		Description:  payload[0].Device,
	}
	mtu, err := service.currentLinux(ctx, info)
	if err != nil {
		return core.InterfaceInfo{}, err
	}
	info.MTU = mtu
	return info, nil
}

func (service *Service) currentLinux(ctx context.Context, info core.InterfaceInfo) (int, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	result, err := service.runner.Run(timeoutCtx, []string{"ip", "-j", "link", "show", "dev", info.Name}, cmdexec.Options{})
	if err != nil {
		return 0, err
	}
	if result.ExitCode != 0 {
		return 0, commandResultError(result.Stdout, result.Stderr)
	}
	var payload []struct {
		MTU int `json:"mtu"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &payload); err != nil {
		return 0, fmt.Errorf("decode linux link response: %w", err)
	}
	if len(payload) == 0 || payload[0].MTU == 0 {
		return 0, fmt.Errorf("could not parse linux MTU")
	}
	return payload[0].MTU, nil
}

func (service *Service) setLinux(ctx context.Context, info core.InterfaceInfo, mtu int, persistent bool) (string, error) {
	if persistent {
		return "", fmt.Errorf("persistent MTU differs by distro and is not implemented for Linux")
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	result, err := service.runner.Run(timeoutCtx, []string{"ip", "link", "set", "dev", info.Name, "mtu", fmt.Sprintf("%d", mtu)}, cmdexec.Options{})
	if err != nil {
		return "", err
	}
	if result.ExitCode != 0 {
		return "", commandResultError(result.Stdout, result.Stderr)
	}
	return strings.TrimSpace(result.Stdout), nil
}

func commandResultError(stdout string, stderr string) error {
	message := strings.TrimSpace(stderr + stdout)
	if message == "" {
		message = "command failed"
	}
	return fmt.Errorf("%s", message)
}

func parseRouteFields(output string) map[string]string {
	fields := map[string]string{}
	for _, line := range strings.Split(output, "\n") {
		if !strings.Contains(line, ":") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		fields[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return fields
}

func firstIPv4FromIfconfig(output string) string {
	match := regexp.MustCompile(`\binet\s+([0-9.]+)`).FindStringSubmatch(output)
	if len(match) == 2 {
		return match[1]
	}
	return ""
}
