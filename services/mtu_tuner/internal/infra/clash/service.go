package clash

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"mtu-tuner/internal/core"
)

type Service struct {
	goos       string
	http       *http.Client
	lookupIPv4 func(ctx context.Context, host string) (string, error)
}

type parsedConfig struct {
	Secret  string
	Hosts   map[string]string
	Proxies map[string]proxyConfig
}

type proxyConfig struct {
	Name   string `yaml:"name"`
	Server string `yaml:"server"`
	Port   int    `yaml:"port"`
}

func New(goos string, httpClient *http.Client) *Service {
	if strings.TrimSpace(goos) == "" {
		goos = runtime.GOOS
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 8 * time.Second}
	}
	return &Service{
		goos: strings.ToLower(goos),
		http: httpClient,
		lookupIPv4: func(ctx context.Context, host string) (string, error) {
			addrs, err := net.DefaultResolver.LookupNetIP(ctx, "ip4", host)
			if err != nil {
				return "", err
			}
			if len(addrs) == 0 {
				return "", fmt.Errorf("no A record for %s", host)
			}
			return addrs[0].String(), nil
		},
	}
}

func (service *Service) ResolveCurrentTarget(ctx context.Context, request core.ResolveTargetRequest) (core.ClashTarget, error) {
	configPath, err := service.findConfigPath(strings.TrimSpace(request.ConfigPath))
	if err != nil {
		return core.ClashTarget{}, err
	}
	parsed, err := service.parseConfig(configPath)
	if err != nil {
		return core.ClashTarget{}, err
	}

	secret := strings.TrimSpace(request.Secret)
	if secret == "" {
		secret = parsed.Secret
	}

	leaf, chain, err := service.resolveLeaf(ctx, strings.TrimSpace(request.Controller), secret, strings.TrimSpace(request.Group))
	if err != nil {
		return core.ClashTarget{}, err
	}
	node, ok := parsed.Proxies[leaf]
	if !ok {
		return core.ClashTarget{}, fmt.Errorf("current leaf proxy %q was not found in %s", leaf, configPath)
	}
	if strings.TrimSpace(node.Server) == "" {
		return core.ClashTarget{}, fmt.Errorf("current leaf proxy %q has no server field", leaf)
	}

	resolvedIP, source, err := service.resolveNodeServer(ctx, strings.TrimSpace(request.Controller), secret, strings.TrimSpace(node.Server), parsed.Hosts)
	if err != nil {
		return core.ClashTarget{}, err
	}
	return core.ClashTarget{
		Group:      strings.Join(chain, " -> "),
		Leaf:       leaf,
		Server:     node.Server,
		Port:       node.Port,
		ResolvedIP: resolvedIP,
		ConfigPath: configPath,
		Source:     source,
	}, nil
}

func (service *Service) SelectProbe(ctx context.Context, request core.DetectRequest, allowFallback bool) (core.ProbeSelection, error) {
	if !core.IsAutoProbe(request.Probe) {
		probeIP, err := service.probeToIP(ctx, request.Probe)
		if err != nil {
			return core.ProbeSelection{}, err
		}
		return core.ProbeSelection{ProbeIP: probeIP}, nil
	}

	target, err := service.ResolveCurrentTarget(ctx, core.ResolveTargetRequest{
		Controller: request.Controller,
		Secret:     request.Secret,
		Group:      request.Group,
		ConfigPath: request.ConfigPath,
	})
	if err == nil {
		return core.ProbeSelection{
			ProbeIP: target.ResolvedIP,
			Target:  &target,
		}, nil
	}
	if !allowFallback {
		return core.ProbeSelection{}, err
	}
	fallback := request.FallbackProbe
	if strings.TrimSpace(fallback) == "" {
		fallback = core.DefaultFallbackProbe
	}
	fallbackIP, fallbackErr := service.probeToIP(ctx, fallback)
	if fallbackErr != nil {
		return core.ProbeSelection{}, fmt.Errorf("auto clash probe failed: %w; fallback %q also failed: %v", err, fallback, fallbackErr)
	}
	return core.ProbeSelection{
		ProbeIP: fallbackIP,
		Warning: fmt.Sprintf("Auto Clash probe failed: %v. Falling back to %s (%s) for route detection.", err, fallback, fallbackIP),
	}, nil
}

func (service *Service) probeToIP(ctx context.Context, probe string) (string, error) {
	value := strings.TrimSpace(probe)
	if value == "" {
		return "", fmt.Errorf("probe cannot be empty")
	}
	if addr, err := netip.ParseAddr(value); err == nil {
		return addr.String(), nil
	}
	return service.lookupIPv4(ctx, value)
}

func (service *Service) resolveLeaf(ctx context.Context, controller string, secret string, group string) (string, []string, error) {
	if core.IsAutoGroup(group) {
		autoGroup, err := service.autoSelectGroup(ctx, controller, secret)
		if err != nil {
			return "", nil, err
		}
		group = autoGroup
	}
	return service.resolveLeafNamed(ctx, controller, secret, group)
}

func (service *Service) autoSelectGroup(ctx context.Context, controller string, secret string) (string, error) {
	proxies, err := service.allProxies(ctx, controller, secret)
	if err != nil {
		return "", err
	}

	candidates := make([]string, 0, len(proxies))
	for _, name := range core.CommonClashGroups {
		if data, ok := proxies[name]; ok && isGroupProxy(data) {
			candidates = append(candidates, name)
		}
	}
	for name, data := range proxies {
		if isGroupProxy(data) && !containsString(candidates, name) {
			candidates = append(candidates, name)
		}
	}
	for _, candidate := range candidates {
		leaf, _, err := service.resolveLeafNamed(ctx, controller, secret, candidate)
		if err != nil {
			continue
		}
		switch strings.ToUpper(leaf) {
		case "DIRECT", "REJECT", "REJECT-DROP":
			continue
		default:
			return candidate, nil
		}
	}
	return "", fmt.Errorf("could not auto-select a usable Clash proxy group")
}

func (service *Service) resolveLeafNamed(ctx context.Context, controller string, secret string, group string) (string, []string, error) {
	seen := map[string]struct{}{}
	chain := make([]string, 0, 8)
	name := strings.TrimSpace(group)
	for depth := 0; depth < 12; depth++ {
		if _, ok := seen[name]; ok {
			return "", nil, fmt.Errorf("proxy group recursion detected: %s", strings.Join(append(chain, name), " -> "))
		}
		seen[name] = struct{}{}
		chain = append(chain, name)
		data, err := service.proxy(ctx, controller, secret, name)
		if err != nil {
			return "", nil, err
		}
		nowValue, _ := data["now"].(string)
		proxyType, _ := data["type"].(string)
		_, hasAll := data["all"]
		if nowValue != "" && hasAll {
			name = nowValue
			continue
		}
		switch strings.ToLower(proxyType) {
		case "selector", "urltest", "fallback", "loadbalance":
			if nowValue != "" {
				name = nowValue
				continue
			}
		}
		return name, chain, nil
	}
	return "", nil, fmt.Errorf("too many nested proxy groups")
}

func (service *Service) resolveNodeServer(ctx context.Context, controller string, secret string, server string, hosts map[string]string) (string, string, error) {
	server = strings.TrimSuffix(strings.TrimSpace(server), ".")
	if addr, err := netip.ParseAddr(server); err == nil {
		return addr.String(), "server-is-ip", nil
	}
	if controller != "" {
		if ip, err := service.dnsQuery(ctx, controller, secret, server); err == nil && ip != "" {
			return ip, "clash-dns", nil
		}
	}
	if hosts != nil {
		if mappedIP, ok := hosts[server]; ok && strings.TrimSpace(mappedIP) != "" {
			return mappedIP, "config-hosts", nil
		}
	}
	ip, err := service.lookupIPv4(ctx, server)
	if err != nil {
		return "", "", err
	}
	return ip, "system-dns", nil
}

func (service *Service) dnsQuery(ctx context.Context, controller string, secret string, name string) (string, error) {
	data, err := service.apiGet(ctx, controller, secret, "/dns/query", map[string]string{
		"name": name,
		"type": "A",
	})
	if err != nil {
		return "", err
	}
	answers, _ := data["Answer"].([]any)
	for _, answer := range answers {
		answerMap, ok := answer.(map[string]any)
		if !ok {
			continue
		}
		answerType, _ := answerMap["type"].(float64)
		if int(answerType) != 1 {
			continue
		}
		if value, ok := answerMap["data"].(string); ok && value != "" {
			return value, nil
		}
	}
	return "", fmt.Errorf("no A record in Clash DNS answer")
}

func (service *Service) apiGet(ctx context.Context, controller string, secret string, path string, query map[string]string) (map[string]any, error) {
	base := strings.TrimSpace(controller)
	if base == "" {
		base = core.DefaultController
	}
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		base = "http://" + base
	}
	targetURL, err := url.Parse(strings.TrimRight(base, "/") + path)
	if err != nil {
		return nil, err
	}
	values := targetURL.Query()
	for key, value := range query {
		values.Set(key, value)
	}
	targetURL.RawQuery = values.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL.String(), nil)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(secret) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(secret))
	}
	resp, err := service.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, targetURL.String())
	}
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (service *Service) proxy(ctx context.Context, controller string, secret string, name string) (map[string]any, error) {
	return service.apiGet(ctx, controller, secret, "/proxies/"+url.PathEscape(name), nil)
}

func (service *Service) allProxies(ctx context.Context, controller string, secret string) (map[string]map[string]any, error) {
	data, err := service.apiGet(ctx, controller, secret, "/proxies", nil)
	if err != nil {
		return nil, err
	}
	rawProxies, ok := data["proxies"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected Clash /proxies response")
	}
	proxies := make(map[string]map[string]any, len(rawProxies))
	for name, raw := range rawProxies {
		typed, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		proxies[name] = typed
	}
	return proxies, nil
}

func (service *Service) findConfigPath(explicit string) (string, error) {
	if explicit != "" {
		path := expandPath(explicit)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
		return "", fmt.Errorf("config file does not exist: %s", path)
	}
	for _, candidate := range service.defaultConfigPaths() {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("could not find Clash Verge config file")
}

func (service *Service) defaultConfigPaths() []string {
	homeDir, _ := os.UserHomeDir()
	switch service.goos {
	case "windows":
		candidates := []string{}
		if appData := strings.TrimSpace(os.Getenv("APPDATA")); appData != "" {
			candidates = append(candidates, filepath.Join(appData, "io.github.clash-verge-rev.clash-verge-rev", "clash-verge.yaml"))
		}
		if localAppData := strings.TrimSpace(os.Getenv("LOCALAPPDATA")); localAppData != "" {
			candidates = append(candidates, filepath.Join(localAppData, "io.github.clash-verge-rev.clash-verge-rev", "clash-verge.yaml"))
		}
		return candidates
	case "darwin":
		return []string{filepath.Join(homeDir, "Library", "Application Support", "io.github.clash-verge-rev.clash-verge-rev", "clash-verge.yaml")}
	default:
		return []string{
			filepath.Join(homeDir, ".config", "io.github.clash-verge-rev.clash-verge-rev", "clash-verge.yaml"),
			filepath.Join(homeDir, ".local", "share", "io.github.clash-verge-rev.clash-verge-rev", "clash-verge.yaml"),
		}
	}
}

func (service *Service) parseConfig(path string) (parsedConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return parsedConfig{}, fmt.Errorf("read clash config: %w", err)
	}
	var payload struct {
		Secret  string            `yaml:"secret"`
		Hosts   map[string]string `yaml:"hosts"`
		Proxies []proxyConfig     `yaml:"proxies"`
	}
	if err := yaml.Unmarshal(data, &payload); err != nil {
		return parsedConfig{}, fmt.Errorf("parse clash config: %w", err)
	}
	proxies := make(map[string]proxyConfig, len(payload.Proxies))
	for _, item := range payload.Proxies {
		if strings.TrimSpace(item.Name) == "" {
			continue
		}
		proxies[item.Name] = item
	}
	hosts := make(map[string]string, len(payload.Hosts))
	for host, value := range payload.Hosts {
		hosts[strings.TrimSuffix(host, ".")] = value
	}
	return parsedConfig{
		Secret:  strings.TrimSpace(payload.Secret),
		Hosts:   hosts,
		Proxies: proxies,
	}, nil
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func isGroupProxy(data map[string]any) bool {
	proxyType, _ := data["type"].(string)
	normalized := strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(proxyType), "-", ""), " ", "")
	switch normalized {
	case "selector", "urltest", "fallback", "loadbalance":
		return true
	}
	_, hasAll := data["all"]
	return hasAll
}

func expandPath(value string) string {
	if strings.HasPrefix(value, "~") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(homeDir, strings.TrimPrefix(value, "~"))
		}
	}
	return value
}
