package proxytest

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"mtu-tuner/internal/core"
)

type ProgressFunc func(done int, total int, label string)

type LogFunc func(line string)

type MTUAdapter interface {
	CurrentMTU(ctx context.Context, info core.InterfaceInfo) (int, error)
	SetMTU(ctx context.Context, info core.InterfaceInfo, mtu int, persistent bool) (string, error)
}

type Service struct {
	goos string
}

func New(goos string) *Service {
	if strings.TrimSpace(goos) == "" {
		goos = runtime.GOOS
	}
	return &Service{goos: strings.ToLower(goos)}
}

func (service *Service) RunSuite(ctx context.Context, request core.TestRunRequest, progress ProgressFunc) (core.TestSummary, error) {
	plan, err := planSuite(request)
	if err != nil {
		return core.TestSummary{}, err
	}
	if len(plan.chromeTargets) > 0 || plan.profile == "chrome" {
		return service.runChromeSuite(ctx, request.HTTPProxy, plan.rounds, request.BrowserPath, plan.chromeTargets, progress)
	}
	return runSpecsSuite(ctx, request.HTTPProxy, plan.profile, plan.specs, plan.concurrency, progress), nil
}

func (service *Service) EstimateTestTotal(profile string, rounds int) int {
	return service.estimateTestTotal(core.TestRunRequest{TestProfile: profile, Rounds: rounds})
}

func (service *Service) RunSweep(
	ctx context.Context,
	adapter MTUAdapter,
	request core.SweepRunRequest,
	progress ProgressFunc,
	log LogFunc,
) (core.SweepResult, error) {
	mtus, err := core.ParseMTUList(request.SweepMTUs)
	if err != nil {
		return core.SweepResult{}, err
	}
	startMTU, err := adapter.CurrentMTU(ctx, request.Interface)
	if err != nil {
		return core.SweepResult{}, err
	}
	perMTUTotal := service.estimateTestTotal(core.TestRunRequest{
		HTTPProxy:   request.HTTPProxy,
		TestProfile: request.TestProfile,
		BrowserPath: request.BrowserPath,
		TestTargets: request.TestTargets,
		Rounds:      request.Rounds,
		Concurrency: request.Concurrency,
	})
	grandTotal := maxInt(1, len(mtus)*perMTUTotal)

	rows := make([]core.SweepRow, 0, len(mtus))
	completedOffset := 0
	cancelled := false

	if log != nil {
		log(fmt.Sprintf("Starting sweep on %s; restore target is %d; profile=%s.", request.Interface.Name, startMTU, request.TestProfile))
	}

	restore := func() int {
		_, _ = adapter.SetMTU(context.Background(), request.Interface, startMTU, false)
		mtu, currentErr := adapter.CurrentMTU(context.Background(), request.Interface)
		if currentErr != nil {
			return startMTU
		}
		return mtu
	}

	defer func() {
		restored := restore()
		_ = restored
	}()

	for _, mtu := range mtus {
		select {
		case <-ctx.Done():
			cancelled = true
			break
		default:
		}
		if progress != nil {
			progress(completedOffset, grandTotal, fmt.Sprintf("MTU %d", mtu))
		}
		if log != nil {
			log(fmt.Sprintf("-- MTU %d --", mtu))
		}

		if _, err := adapter.SetMTU(ctx, request.Interface, mtu, false); err != nil {
			if log != nil {
				log("Set MTU failed: " + err.Error())
			}
			rows = append(rows, core.SweepRow{
				MTU:          mtu,
				Effective:    0,
				Profile:      request.TestProfile,
				PlannedTotal: perMTUTotal,
				FirstError:   err.Error(),
			})
			completedOffset += perMTUTotal
			if progress != nil {
				progress(minInt(grandTotal, completedOffset), grandTotal, fmt.Sprintf("MTU %d set failed", mtu))
			}
			continue
		}
		if err := waitForSettle(ctx, 1500*time.Millisecond); err != nil {
			cancelled = true
			rows = append(rows, core.SweepRow{
				MTU:          mtu,
				Profile:      request.TestProfile,
				PlannedTotal: perMTUTotal,
				FirstError:   "cancelled before test",
				Cancelled:    true,
			})
			break
		}
		effective, err := adapter.CurrentMTU(ctx, request.Interface)
		if err != nil {
			if log != nil {
				log("Readback MTU failed: " + err.Error())
			}
			rows = append(rows, core.SweepRow{
				MTU:          mtu,
				Profile:      request.TestProfile,
				PlannedTotal: perMTUTotal,
				FirstError:   err.Error(),
			})
			completedOffset += perMTUTotal
			if progress != nil {
				progress(minInt(grandTotal, completedOffset), grandTotal, fmt.Sprintf("MTU %d readback failed", mtu))
			}
			continue
		}
		if effective != mtu {
			if log != nil {
				log(fmt.Sprintf("MTU readback mismatch: requested %d, effective %d.", mtu, effective))
			}
			rows = append(rows, core.SweepRow{
				MTU:          mtu,
				Effective:    effective,
				Profile:      request.TestProfile,
				PlannedTotal: perMTUTotal,
				FirstError:   "MTU did not apply",
			})
			completedOffset += perMTUTotal
			if progress != nil {
				progress(minInt(grandTotal, completedOffset), grandTotal, fmt.Sprintf("MTU %d did not apply", mtu))
			}
			continue
		}

		summary, err := service.RunSuite(ctx, core.TestRunRequest{
			Interface:   request.Interface,
			HTTPProxy:   request.HTTPProxy,
			TestProfile: request.TestProfile,
			BrowserPath: request.BrowserPath,
			TestTargets: request.TestTargets,
			Rounds:      request.Rounds,
			Concurrency: request.Concurrency,
		}, func(done int, total int, label string) {
			if progress != nil {
				progress(minInt(grandTotal, completedOffset+done), grandTotal, fmt.Sprintf("MTU %d %s", mtu, strings.Split(label, "#")[0]))
			}
		})
		row := core.SweepRow{
			MTU:          mtu,
			Effective:    effective,
			Profile:      summary.Profile,
			PlannedTotal: summary.PlannedTotal,
			Total:        summary.Total,
			OK:           summary.OK,
			Failures:     summary.Failures,
			FirstError:   summary.FirstError,
			Cancelled:    summary.Cancelled,
		}
		if err != nil {
			row.FirstError = err.Error()
		}
		rows = append(rows, row)
		if log != nil {
			log(fmt.Sprintf("%d/%d ok (planned %d), failures=%d, avg=%.3fs, p95=%.3fs, bytes=%d", summary.OK, summary.Total, summary.PlannedTotal, summary.Failures, summary.Avg, summary.P95, summary.Bytes))
			if row.FirstError != "" {
				log("First error: " + row.FirstError)
			}
		}
		completedOffset += perMTUTotal
		if progress != nil {
			progress(minInt(grandTotal, completedOffset), grandTotal, fmt.Sprintf("MTU %d complete", mtu))
		}
		if summary.Cancelled || ctx.Err() != nil {
			cancelled = true
			break
		}
	}

	restoredMTU := restore()
	outputPath, csvErr := writeSweepCSV(rows)
	if csvErr != nil {
		return core.SweepResult{}, csvErr
	}
	if log != nil {
		if cancelled {
			log("Sweep stopped; original MTU restored.")
		} else {
			log(fmt.Sprintf("Sweep complete; restored MTU %d.", restoredMTU))
		}
	}
	if progress != nil && !cancelled {
		progress(grandTotal, grandTotal, "complete")
	}
	return core.SweepResult{
		Rows:        rows,
		OutputPath:  outputPath,
		StartMTU:    startMTU,
		RestoredMTU: restoredMTU,
		Cancelled:   cancelled,
	}, nil
}

func pickRounds(value int, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func pickConcurrency(value int, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func parseProxy(proxyURL string) (string, int, error) {
	value := strings.TrimSpace(proxyURL)
	if !strings.Contains(value, "://") {
		value = "http://" + value
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return "", 0, err
	}
	if strings.ToLower(parsed.Scheme) != "http" || parsed.Hostname() == "" || parsed.Port() == "" {
		return "", 0, fmt.Errorf("proxy must look like http://127.0.0.1:7890")
	}
	port, err := net.LookupPort("tcp", parsed.Port())
	if err != nil {
		return "", 0, err
	}
	return parsed.Hostname(), port, nil
}

func normalizeProxyURL(proxyURL string) (string, error) {
	host, port, err := parseProxy(proxyURL)
	if err != nil {
		return "", fmt.Errorf("browser profile needs an HTTP proxy such as http://127.0.0.1:7890: %w", err)
	}
	return fmt.Sprintf("http://%s:%d", host, port), nil
}

type namedSpec struct {
	name string
	spec core.TestSpec
}

type namedCheck struct {
	name   string
	result core.CheckResult
}

func expandSpecs(baseSpecs []core.TestSpec, rounds int) []namedSpec {
	specs := make([]namedSpec, 0, len(baseSpecs)*maxInt(1, rounds))
	for round := 0; round < maxInt(1, rounds); round++ {
		for _, spec := range baseSpecs {
			specs = append(specs, namedSpec{
				name: fmt.Sprintf("%s#%d", spec.Name, round+1),
				spec: spec,
			})
		}
	}
	return specs
}

func (service *Service) estimateTestTotal(request core.TestRunRequest) int {
	plan, err := planSuite(request)
	if err != nil {
		return 1
	}
	if plan.profile == "chrome" {
		return plan.rounds * len(plan.chromeTargets)
	}
	return len(plan.specs)
}

func runSpecsSuite(ctx context.Context, proxyURL string, profile string, specs []namedSpec, concurrency int, progress ProgressFunc) core.TestSummary {
	workers := maxInt(1, minInt(concurrency, maxInt(len(specs), 1)))
	total := len(specs)
	if progress != nil {
		progress(0, total, "starting")
	}

	jobs := make(chan namedSpec)
	results := make(chan namedCheck, total)
	var workersWG sync.WaitGroup
	for workerIndex := 0; workerIndex < workers; workerIndex++ {
		workersWG.Add(1)
		go func() {
			defer workersWG.Done()
			for spec := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
				}
				results <- namedCheck{
					name:   spec.name,
					result: httpsCheckViaHTTPProxy(ctx, proxyURL, spec.spec),
				}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for _, spec := range specs {
			select {
			case <-ctx.Done():
				return
			case jobs <- spec:
			}
		}
	}()

	go func() {
		workersWG.Wait()
		close(results)
	}()

	completed := 0
	checks := make([]namedCheck, 0, total)
	for result := range results {
		checks = append(checks, result)
		completed++
		if progress != nil {
			progress(completed, total, result.name)
		}
	}
	return summarizeChecks(profile, checks, workers, total, ctx.Err() != nil)
}

func summarizeChecks(profile string, checks []namedCheck, concurrency int, plannedTotal int, cancelled bool) core.TestSummary {
	okCount := 0
	durations := make([]float64, 0, len(checks))
	failByName := map[string]int{}
	failByError := map[string]int{}
	firstError := ""
	totalBytes := 0
	for _, item := range checks {
		totalBytes += item.result.BytesRead
		if item.result.OK {
			okCount++
			durations = append(durations, item.result.Elapsed)
			continue
		}
		durations = append(durations, item.result.Elapsed)
		name := strings.Split(item.name, "#")[0]
		failByName[name]++
		errText := strings.TrimSpace(strings.ReplaceAll(item.result.Error, "\n", " "))
		if errText == "" {
			errText = "unknown error"
		}
		if len(errText) > 180 {
			errText = errText[:177] + "..."
		}
		failByError[errText]++
		if firstError == "" {
			firstError = item.name + ": " + item.result.Error
		}
	}
	avg := 0.0
	p95 := 0.0
	if len(durations) > 0 {
		sort.Float64s(durations)
		sum := 0.0
		for _, value := range durations {
			sum += value
		}
		avg = sum / float64(len(durations))
		index := int(float64(len(durations)-1) * 0.95)
		if index < 0 {
			index = 0
		}
		p95 = durations[index]
	}
	return core.TestSummary{
		Profile:      profile,
		Concurrency:  concurrency,
		PlannedTotal: plannedTotal,
		Total:        len(checks),
		OK:           okCount,
		Failures:     len(checks) - okCount,
		Avg:          avg,
		P95:          p95,
		Bytes:        totalBytes,
		FailByName:   failByName,
		FailByError:  failByError,
		FirstError:   firstError,
		Cancelled:    cancelled,
	}
}

func httpsCheckViaHTTPProxy(ctx context.Context, proxyURL string, spec core.TestSpec) core.CheckResult {
	start := time.Now()
	proxyHost, proxyPort, err := parseProxy(proxyURL)
	if err != nil {
		return core.CheckResult{Error: err.Error(), Elapsed: secondsSince(start)}
	}

	timeout := time.Duration(spec.TimeoutSec * float64(time.Second))
	dialer := &net.Dialer{Timeout: timeout}
	rawConn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", proxyHost, proxyPort))
	if err != nil {
		return core.CheckResult{Error: err.Error(), Elapsed: secondsSince(start)}
	}
	defer rawConn.Close()
	_ = rawConn.SetDeadline(time.Now().Add(timeout))

	connectRequest := fmt.Sprintf("CONNECT %s:443 HTTP/1.1\r\nHost: %s:443\r\nProxy-Connection: keep-alive\r\nUser-Agent: MTUTuner/1.0\r\n\r\n", spec.Host, spec.Host)
	if _, err := rawConn.Write([]byte(connectRequest)); err != nil {
		return core.CheckResult{Error: err.Error(), Elapsed: secondsSince(start)}
	}
	connectResponse, err := readUntil(rawConn, []byte("\r\n\r\n"), 64*1024)
	if err != nil {
		return core.CheckResult{Error: err.Error(), Elapsed: secondsSince(start)}
	}
	firstLine := strings.SplitN(string(connectResponse), "\r\n", 2)[0]
	if !strings.Contains(firstLine, " 200 ") {
		return core.CheckResult{Error: "CONNECT failed: " + firstLine, Elapsed: secondsSince(start)}
	}

	tlsConn := tlsClient(rawConn, spec.Host)
	defer tlsConn.Close()
	_ = tlsConn.SetDeadline(time.Now().Add(timeout))
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		return core.CheckResult{Error: err.Error(), Elapsed: secondsSince(start)}
	}

	accept := "*/*"
	if strings.EqualFold(spec.Method, http.MethodGet) {
		accept = "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8"
	}
	request := fmt.Sprintf(
		"%s %s HTTP/1.1\r\nHost: %s\r\nUser-Agent: %s\r\nAccept: %s\r\nAccept-Language: en-US,en;q=0.9\r\nAccept-Encoding: identity\r\nCache-Control: no-cache\r\nPragma: no-cache\r\nConnection: close\r\n\r\n",
		strings.ToUpper(spec.Method),
		spec.Path,
		spec.Host,
		core.BrowserUserAgent,
		accept,
	)
	if _, err := tlsConn.Write([]byte(request)); err != nil {
		return core.CheckResult{Error: err.Error(), Elapsed: secondsSince(start)}
	}

	response, err := readUntil(tlsConn, []byte("\r\n\r\n"), 64*1024)
	if err != nil {
		return core.CheckResult{Error: err.Error(), Elapsed: secondsSince(start)}
	}
	headerParts := strings.SplitN(string(response), "\r\n\r\n", 2)
	statusLine := strings.SplitN(headerParts[0], "\r\n", 2)[0]
	code := 0
	fmt.Sscanf(statusLine, "HTTP/%*s %d", &code)
	ok := code >= 200 && code < 400
	bytesRead := 0
	if len(headerParts) == 2 {
		bytesRead = len([]byte(headerParts[1]))
	}
	if ok && !strings.EqualFold(spec.Method, http.MethodHead) && spec.ReadBodyBytes > bytesRead {
		buffer := make([]byte, minInt(32768, spec.ReadBodyBytes-bytesRead))
		for bytesRead < spec.ReadBodyBytes {
			readCount, readErr := tlsConn.Read(buffer)
			bytesRead += readCount
			if readErr != nil {
				break
			}
			if len(buffer) != minInt(32768, spec.ReadBodyBytes-bytesRead) {
				buffer = make([]byte, minInt(32768, spec.ReadBodyBytes-bytesRead))
			}
		}
	}
	errText := ""
	if !ok {
		errText = statusLine
	}
	return core.CheckResult{
		OK:        ok,
		Code:      code,
		Elapsed:   secondsSince(start),
		Error:     errText,
		BytesRead: bytesRead,
	}
}

func waitForSettle(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

func writeSweepCSV(rows []core.SweepRow) (string, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	path := filepath.Join(workingDir, "mtu-tuner-sweep_"+time.Now().Format("20060102_150405")+".csv")
	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	if _, err := fmt.Fprintln(file, "mtu,effective,profile,planned_total,total,ok,failures,first_error,cancelled"); err != nil {
		return "", err
	}
	for _, row := range rows {
		if _, err := fmt.Fprintf(
			file,
			"%d,%d,%s,%d,%d,%d,%d,%q,%t\n",
			row.MTU,
			row.Effective,
			row.Profile,
			row.PlannedTotal,
			row.Total,
			row.OK,
			row.Failures,
			row.FirstError,
			row.Cancelled,
		); err != nil {
			return "", err
		}
	}
	return path, nil
}

func browserCandidates(goos string) []string {
	candidates := []string{}
	if envValue := strings.TrimSpace(os.Getenv(core.BrowserEnvKey)); envValue != "" {
		candidates = append(candidates, envValue)
	}
	switch goos {
	case "windows":
		for _, root := range []string{os.Getenv("PROGRAMFILES"), os.Getenv("PROGRAMFILES(X86)"), os.Getenv("LOCALAPPDATA")} {
			if strings.TrimSpace(root) == "" {
				continue
			}
			candidates = append(candidates,
				filepath.Join(root, "Google", "Chrome", "Application", "chrome.exe"),
				filepath.Join(root, "Microsoft", "Edge", "Application", "msedge.exe"),
			)
		}
		candidates = append(candidates, "chrome.exe", "msedge.exe")
	case "darwin":
		candidates = append(candidates,
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"google-chrome",
			"chromium",
		)
	default:
		candidates = append(candidates, "google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "microsoft-edge")
	}
	return candidates
}

func (service *Service) findBrowserExecutable(explicit string) (string, error) {
	explicit = strings.Trim(strings.TrimSpace(explicit), "\"")
	if explicit != "" {
		if stat, err := os.Stat(explicit); err == nil && !stat.IsDir() {
			return explicit, nil
		}
		if resolved, err := exec.LookPath(explicit); err == nil {
			return resolved, nil
		}
		return "", fmt.Errorf("configured browser executable does not exist: %s", explicit)
	}
	for _, candidate := range browserCandidates(service.goos) {
		if stat, err := os.Stat(candidate); err == nil && !stat.IsDir() {
			return candidate, nil
		}
		if resolved, err := exec.LookPath(candidate); err == nil {
			return resolved, nil
		}
	}
	return "", fmt.Errorf("could not find Chrome/Edge/Chromium; set %s to the browser executable path", core.BrowserEnvKey)
}
