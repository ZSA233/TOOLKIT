package clash

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestParseConfigReadsSecretHostsAndProxies(t *testing.T) {
	t.Parallel()

	service := New("darwin", nil)
	path := filepath.Join(t.TempDir(), "clash.yaml")
	payload := `
secret: secret-token
hosts:
  edge.example.com.: 1.2.3.4
proxies:
  - name: HK
    server: edge.example.com
    port: 443
`
	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	parsed, err := service.parseConfig(path)
	if err != nil {
		t.Fatalf("parseConfig() error = %v", err)
	}

	if parsed.Secret != "secret-token" {
		t.Fatalf("parsed.Secret = %q, want %q", parsed.Secret, "secret-token")
	}
	if parsed.Hosts["edge.example.com"] != "1.2.3.4" {
		t.Fatalf("parsed.Hosts[edge.example.com] = %q", parsed.Hosts["edge.example.com"])
	}
	if parsed.Proxies["HK"].Server != "edge.example.com" {
		t.Fatalf("parsed.Proxies[HK].Server = %q", parsed.Proxies["HK"].Server)
	}
}

func TestResolveNodeServerPrefersConfigHosts(t *testing.T) {
	t.Parallel()

	service := New("darwin", nil)
	service.lookupIPv4 = func(ctx context.Context, host string) (string, error) {
		t.Fatalf("lookupIPv4() should not be called for config-host hits")
		return "", nil
	}

	ip, source, err := service.resolveNodeServer(
		context.Background(),
		"",
		"",
		"edge.example.com.",
		map[string]string{"edge.example.com": "1.2.3.4"},
	)
	if err != nil {
		t.Fatalf("resolveNodeServer() error = %v", err)
	}
	if ip != "1.2.3.4" {
		t.Fatalf("resolveNodeServer() ip = %q, want %q", ip, "1.2.3.4")
	}
	if source != "config-hosts" {
		t.Fatalf("resolveNodeServer() source = %q, want %q", source, "config-hosts")
	}
}
