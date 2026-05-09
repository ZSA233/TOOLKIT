package proxytest

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"mtu-tuner/internal/core"
)

type chromeProbeServer struct {
	httpServer *http.Server
	listener   net.Listener
	resultCh   chan []map[string]any
	errorCh    chan error
	bodyBytes  int
}

func startChromeProbeServer(targets []core.ChromeProbeTarget) (*chromeProbeServer, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	server := &chromeProbeServer{
		listener: listener,
		resultCh: make(chan []map[string]any, 1),
		errorCh:  make(chan error, 1),
	}
	mux := http.NewServeMux()
	page := buildChromeProbeHTML(targets)
	mux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/" && request.URL.Path != "/probe" {
			http.NotFound(writer, request)
			return
		}
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		writer.Header().Set("Cache-Control", "no-store")
		_, _ = io.WriteString(writer, page)
	})
	mux.HandleFunc("/report", func(writer http.ResponseWriter, request *http.Request) {
		defer request.Body.Close()
		body, err := io.ReadAll(io.LimitReader(request.Body, 1024*1024))
		server.bodyBytes = len(body)
		if err != nil {
			server.errorCh <- err
			writer.WriteHeader(http.StatusNoContent)
			return
		}
		var payload []map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			server.errorCh <- err
			writer.WriteHeader(http.StatusNoContent)
			return
		}
		server.resultCh <- payload
		writer.WriteHeader(http.StatusNoContent)
	})
	server.httpServer = &http.Server{Handler: mux}
	go func() {
		_ = server.httpServer.Serve(listener)
	}()
	return server, nil
}

func (server *chromeProbeServer) URL() string {
	return fmt.Sprintf("http://127.0.0.1:%d/probe", server.listener.Addr().(*net.TCPAddr).Port)
}

func (server *chromeProbeServer) Shutdown() {
	_ = server.httpServer.Shutdown(context.Background())
	_ = server.listener.Close()
}

func buildChromeProbeHTML(targetsList []core.ChromeProbeTarget) string {
	targets := make([]map[string]string, 0, len(targetsList))
	for _, target := range targetsList {
		targets = append(targets, map[string]string{
			"name": target.Name,
			"kind": target.Kind,
			"url":  target.URL,
		})
	}
	targetsJSON, _ := json.Marshal(targets)
	return fmt.Sprintf(`<!doctype html>
<html>
<head><meta charset="utf-8"><title>MTU Browser Probe</title></head>
<body>
<pre id="result">PROBE_PENDING</pre>
<script>
const targets = %s;
const timeoutMs = 10000;
function cacheBust(url) {
  const sep = url.includes("?") ? "&" : "?";
  return url + sep + "_probe=" + Date.now() + "_" + Math.random().toString(16).slice(2);
}
function done(item) {
  return { name: item.name, ok: true, kind: item.kind, error: "" };
}
function fail(item, error) {
  return { name: item.name, ok: false, kind: item.kind, error: String(error && error.message ? error.message : error) };
}
function withTimeout(promise) {
  let timer;
  const timeout = new Promise((_, reject) => {
    timer = setTimeout(() => reject(new Error("timeout")), timeoutMs);
  });
  return Promise.race([promise, timeout]).finally(() => clearTimeout(timer));
}
function probeFetch(item) {
  return withTimeout(fetch(cacheBust(item.url), { mode: "no-cors", cache: "no-store" }))
    .then(() => done(item))
    .catch((error) => fail(item, error));
}
function probeImage(item) {
  return withTimeout(new Promise((resolve, reject) => {
    const img = new Image();
    img.onload = () => resolve();
    img.onerror = () => reject(new Error("image error"));
    img.src = cacheBust(item.url);
  })).then(() => done(item))
    .catch((error) => fail(item, error));
}
Promise.all(targets.map((item) => item.kind === "image" ? probeImage(item) : probeFetch(item)))
  .then((results) => {
    document.getElementById("result").textContent = "PROBE_RESULT " + encodeURIComponent(JSON.stringify(results));
    return fetch("/report", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(results),
      keepalive: true
    });
  })
  .catch((error) => {
    const results = [{ name: "probe_page", ok: false, kind: "script", error: String(error) }];
    document.getElementById("result").textContent = "PROBE_RESULT " + encodeURIComponent(JSON.stringify(results));
    return fetch("/report", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(results),
      keepalive: true
    });
  });
</script>
</body>
</html>`, string(targetsJSON))
}

func (service *Service) runChromeSuite(ctx context.Context, proxyURL string, rounds int, browserPath string, targets []core.ChromeProbeTarget, progress ProgressFunc) (core.TestSummary, error) {
	browser, err := service.findBrowserExecutable(browserPath)
	if err != nil {
		return summarizeChecks("chrome", []namedCheck{{name: "browser", result: core.CheckResult{Error: err.Error(), Elapsed: 0}}}, 1, 1, false), nil
	}

	total := maxInt(1, rounds) * len(targets)
	completed := 0
	checks := make([]namedCheck, 0, total)
	if progress != nil {
		progress(0, total, "starting chrome")
	}
	for round := 0; round < maxInt(1, rounds); round++ {
		if ctx.Err() != nil {
			break
		}
		roundChecks := service.runChromeProbeRound(ctx, browser, proxyURL, targets)
		checks = append(checks, roundChecks...)
		for _, item := range roundChecks {
			completed++
			if progress != nil {
				progress(minInt(completed, total), total, item.name)
			}
		}
	}
	summary := summarizeChecks("chrome", checks, 1, total, ctx.Err() != nil)
	summary.Browser = browser
	summary.ProbeTransport = "local-http-report"
	return summary, nil
}

func (service *Service) runChromeProbeRound(
	ctx context.Context,
	browser string,
	proxyURL string,
	targets []core.ChromeProbeTarget,
) []namedCheck {
	start := time.Now()
	server, err := startChromeProbeServer(targets)
	if err != nil {
		return []namedCheck{{name: "chrome_probe", result: core.CheckResult{Error: err.Error(), Elapsed: secondsSince(start)}}}
	}
	defer server.Shutdown()

	userDir, err := os.MkdirTemp("", "mtu-tuner-chrome-")
	if err != nil {
		return []namedCheck{{name: "chrome_probe", result: core.CheckResult{Error: err.Error(), Elapsed: secondsSince(start)}}}
	}
	defer os.RemoveAll(userDir)

	args, err := chromeBaseArgs(service.goos, browser, proxyURL, userDir)
	if err != nil {
		return []namedCheck{{name: "chrome_probe", result: core.CheckResult{Error: err.Error(), Elapsed: secondsSince(start)}}}
	}
	args = append(args, server.URL())

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	cmd := exec.CommandContext(runCtx, args[0], args[1:]...)
	configureChromeCommand(cmd)
	stdoutPipe, _ := cmd.StdoutPipe()
	stderrPipe, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		return []namedCheck{{name: "chrome_probe", result: core.CheckResult{Error: err.Error(), Elapsed: secondsSince(start)}}}
	}

	type processOutput struct {
		stdout []byte
		stderr []byte
	}
	outputCh := make(chan processOutput, 1)
	go func() {
		stdout, _ := io.ReadAll(stdoutPipe)
		stderr, _ := io.ReadAll(stderrPipe)
		outputCh <- processOutput{stdout: stdout, stderr: stderr}
	}()

	deadline := time.NewTimer(18 * time.Second)
	defer deadline.Stop()

	var results []map[string]any
	select {
	case <-ctx.Done():
		cancel()
	case err := <-server.errorCh:
		cancel()
		return []namedCheck{{name: "chrome_probe", result: core.CheckResult{Error: err.Error(), Elapsed: secondsSince(start)}}}
	case payload := <-server.resultCh:
		results = payload
		cancel()
	case <-deadline.C:
		cancel()
	}

	_ = cmd.Wait()
	output := <-outputCh
	elapsed := secondsSince(start)
	outputBytes := len(output.stdout) + len(output.stderr) + server.bodyBytes

	if ctx.Err() != nil && len(results) == 0 {
		return []namedCheck{{name: "chrome_probe", result: core.CheckResult{Error: "cancelled", Elapsed: elapsed, BytesRead: outputBytes}}}
	}
	if len(results) == 0 {
		detail := strings.TrimSpace(string(output.stdout))
		if detail == "" {
			detail = strings.TrimSpace(string(output.stderr))
		}
		if detail == "" {
			detail = "browser did not report before timeout"
		}
		return []namedCheck{{name: "chrome_probe", result: core.CheckResult{Error: "probe did not report: " + detail, Elapsed: elapsed, BytesRead: outputBytes}}}
	}

	checks := make([]namedCheck, 0, len(results))
	for _, item := range results {
		name, _ := item["name"].(string)
		ok, _ := item["ok"].(bool)
		errorText, _ := item["error"].(string)
		checks = append(checks, namedCheck{
			name: "chrome_" + name,
			result: core.CheckResult{
				OK:        ok,
				Elapsed:   elapsed,
				Error:     errorText,
				BytesRead: outputBytes,
			},
		})
	}
	return checks
}

func chromeBaseArgs(goos string, browser string, proxyURL string, userDir string) ([]string, error) {
	proxy, err := normalizeProxyURL(proxyURL)
	if err != nil {
		return nil, err
	}
	args := []string{
		browser,
		"--headless=new",
		"--disable-background-networking",
		"--disable-default-apps",
		"--disable-extensions",
		"--disable-gpu",
		"--disable-quic",
		"--disable-sync",
		"--hide-scrollbars",
		"--no-first-run",
		"--no-default-browser-check",
		"--proxy-server=" + proxy,
		"--proxy-bypass-list=127.0.0.1;localhost",
		"--user-data-dir=" + userDir,
	}
	if goos == "linux" && os.Geteuid() == 0 {
		args = append([]string{browser, "--no-sandbox"}, args[1:]...)
	}
	return args, nil
}

func parseChromeProbeResults(stdout string) []map[string]any {
	parts := strings.Split(stdout, "PROBE_RESULT ")
	if len(parts) < 2 {
		return nil
	}
	encoded := html.UnescapeString(strings.Fields(parts[len(parts)-1])[0])
	decoded, err := url.QueryUnescape(encoded)
	if err != nil {
		return nil
	}
	var payload []map[string]any
	if err := json.Unmarshal([]byte(decoded), &payload); err != nil {
		return nil
	}
	return payload
}
