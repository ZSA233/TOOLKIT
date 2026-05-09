package netiface

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"mtu-tuner/internal/core"

	"toolkit/libs/appkit/cmdexec"
)

func (service *Service) ListInterfaces(ctx context.Context) ([]core.InterfaceInfo, error) {
	switch service.goos {
	case "windows":
		return service.listWindows(ctx)
	case "darwin":
		return service.listDarwin(ctx)
	case "linux":
		return service.listLinux(ctx)
	default:
		return nil, fmt.Errorf("unsupported platform: %s", service.goos)
	}
}

func (service *Service) listWindows(ctx context.Context) ([]core.InterfaceInfo, error) {
	script := `
$rows = @()
$ipifs = Get-NetIPInterface -AddressFamily IPv4 | Where-Object { $_.NlMtu -gt 0 -and $_.InterfaceAlias -notmatch 'Loopback' }
foreach ($ipif in $ipifs) {
  $adapter = Get-NetAdapter -InterfaceIndex $ipif.InterfaceIndex -ErrorAction SilentlyContinue
  if ($adapter -and $adapter.Status -ne 'Up') { continue }
  $ip = Get-NetIPAddress -AddressFamily IPv4 -InterfaceIndex $ipif.InterfaceIndex -ErrorAction SilentlyContinue |
    Where-Object { $_.IPAddress -and $_.IPAddress -notlike '169.254*' } |
    Select-Object -First 1
  if (-not $ip) { continue }
  $route = Get-NetRoute -InterfaceIndex $ipif.InterfaceIndex -AddressFamily IPv4 -DestinationPrefix '0.0.0.0/0' -ErrorAction SilentlyContinue |
    Where-Object { $_.NextHop } |
    Select-Object -First 1
  $rows += [pscustomobject]@{
    platform='Windows'
    name=[string]$ipif.InterfaceAlias
    index=[string]$ipif.InterfaceIndex
    mtu=[int]$ipif.NlMtu
    gateway=[string]$route.NextHop
    local_address=[string]$ip.IPAddress
    description=[string]$adapter.InterfaceDescription
  }
}
@($rows) | ConvertTo-Json -Compress
`
	timeoutCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	result, err := cmdexec.RunPowerShell(timeoutCtx, service.runner, script)
	if err != nil {
		return nil, err
	}
	if result.ExitCode != 0 {
		return nil, commandResultError(result.Stdout, result.Stderr)
	}
	interfaces, err := decodeInterfaceListJSON(result.Stdout)
	if err != nil {
		return nil, err
	}
	return normalizeInterfaceList(interfaces), nil
}

func (service *Service) listDarwin(ctx context.Context) ([]core.InterfaceInfo, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	result, err := service.runner.Run(timeoutCtx, []string{"ifconfig"}, cmdexec.Options{})
	if err != nil {
		return nil, err
	}
	if result.ExitCode != 0 {
		return nil, commandResultError(result.Stdout, result.Stderr)
	}

	interfaces := []core.InterfaceInfo{}
	blocks := splitIfconfigBlocks(result.Stdout)
	upPattern := regexp.MustCompile(`flags=.*<([^>]+)>`)
	mtuPattern := regexp.MustCompile(`\bmtu\s+(\d+)`)
	ipv4Pattern := regexp.MustCompile(`\binet\s+([0-9.]+)`)

	for _, block := range blocks {
		header := block[0]
		name := strings.TrimSpace(strings.SplitN(header, ":", 2)[0])
		if name == "" {
			continue
		}
		flagsMatch := upPattern.FindStringSubmatch(header)
		if len(flagsMatch) != 2 {
			continue
		}
		flags := flagsMatch[1]
		if !strings.Contains(flags, "UP") || strings.Contains(flags, "LOOPBACK") {
			continue
		}
		blockText := strings.Join(block, "\n")
		mtuMatch := mtuPattern.FindStringSubmatch(blockText)
		ipMatch := ipv4Pattern.FindStringSubmatch(blockText)
		if len(mtuMatch) != 2 || len(ipMatch) != 2 {
			continue
		}
		var mtu int
		if _, err := fmt.Sscanf(mtuMatch[1], "%d", &mtu); err != nil || mtu <= 0 {
			continue
		}
		interfaces = append(interfaces, core.InterfaceInfo{
			PlatformName: "Darwin",
			Name:         name,
			MTU:          mtu,
			LocalAddress: ipMatch[1],
			Description:  name,
		})
	}
	return normalizeInterfaceList(interfaces), nil
}

func (service *Service) listLinux(ctx context.Context) ([]core.InterfaceInfo, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	result, err := service.runner.Run(timeoutCtx, []string{"ip", "-j", "addr", "show", "up"}, cmdexec.Options{})
	if err != nil {
		return nil, err
	}
	if result.ExitCode != 0 {
		return nil, commandResultError(result.Stdout, result.Stderr)
	}
	var payload []struct {
		Name      string `json:"ifname"`
		MTU       int    `json:"mtu"`
		LinkType  string `json:"link_type"`
		OperState string `json:"operstate"`
		AddrInfo  []struct {
			Family string `json:"family"`
			Local  string `json:"local"`
		} `json:"addr_info"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &payload); err != nil {
		return nil, fmt.Errorf("decode linux interface list: %w", err)
	}
	interfaces := make([]core.InterfaceInfo, 0, len(payload))
	for _, item := range payload {
		if item.Name == "" || item.MTU <= 0 || strings.EqualFold(item.LinkType, "loopback") {
			continue
		}
		local := ""
		for _, addr := range item.AddrInfo {
			if addr.Family == "inet" && strings.TrimSpace(addr.Local) != "" {
				local = strings.TrimSpace(addr.Local)
				break
			}
		}
		if local == "" {
			continue
		}
		interfaces = append(interfaces, core.InterfaceInfo{
			PlatformName: "Linux",
			Name:         item.Name,
			MTU:          item.MTU,
			LocalAddress: local,
			Description:  item.Name,
		})
	}
	return normalizeInterfaceList(interfaces), nil
}

func decodeInterfaceListJSON(raw string) ([]core.InterfaceInfo, error) {
	if strings.TrimSpace(raw) == "" {
		return []core.InterfaceInfo{}, nil
	}
	var payload []struct {
		Platform     string `json:"platform"`
		Name         string `json:"name"`
		Index        string `json:"index"`
		MTU          int    `json:"mtu"`
		Gateway      string `json:"gateway"`
		LocalAddress string `json:"local_address"`
		Description  string `json:"description"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, fmt.Errorf("decode windows interface list: %w", err)
	}
	interfaces := make([]core.InterfaceInfo, 0, len(payload))
	for _, item := range payload {
		interfaces = append(interfaces, core.InterfaceInfo{
			PlatformName: item.Platform,
			Name:         item.Name,
			Index:        item.Index,
			MTU:          item.MTU,
			Gateway:      item.Gateway,
			LocalAddress: item.LocalAddress,
			Description:  item.Description,
		})
	}
	return interfaces, nil
}

func splitIfconfigBlocks(output string) [][]string {
	lines := strings.Split(output, "\n")
	blocks := make([][]string, 0)
	var current []string
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if len(line) > 0 && line[0] != '\t' && line[0] != ' ' {
			if len(current) > 0 {
				blocks = append(blocks, current)
			}
			current = []string{line}
			continue
		}
		if len(current) > 0 {
			current = append(current, line)
		}
	}
	if len(current) > 0 {
		blocks = append(blocks, current)
	}
	return blocks
}

func normalizeInterfaceList(values []core.InterfaceInfo) []core.InterfaceInfo {
	if len(values) == 0 {
		return []core.InterfaceInfo{}
	}
	normalized := make([]core.InterfaceInfo, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		if strings.TrimSpace(value.Name) == "" || value.MTU <= 0 || strings.TrimSpace(value.LocalAddress) == "" {
			continue
		}
		if strings.TrimSpace(value.Description) == "" {
			value.Description = value.Name
		}
		key := core.InterfaceKey(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, value)
	}
	sort.SliceStable(normalized, func(i int, j int) bool {
		if normalized[i].Name == normalized[j].Name {
			return normalized[i].Index < normalized[j].Index
		}
		return normalized[i].Name < normalized[j].Name
	})
	return normalized
}
