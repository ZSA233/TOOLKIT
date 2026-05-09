package app

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"strings"
	"sync"

	"mtu-tuner/internal/core"
	"mtu-tuner/internal/infra/clash"
	"mtu-tuner/internal/infra/netiface"
	"mtu-tuner/internal/infra/settingsstore"
	"mtu-tuner/internal/proxytest"
	"mtu-tuner/internal/tasks"

	"toolkit/libs/appkit/cmdexec"
)

type Service struct {
	goos      string
	settings  *settingsstore.Store
	netiface  *netiface.Service
	clash     *clash.Service
	proxytest *proxytest.Service
	tasks     *tasks.Manager
	isAdmin   func() bool

	mu               sync.Mutex
	originalMTUByKey map[string]int
}

func NewDefaultService(goos string, runner cmdexec.Runner) (*Service, error) {
	store, err := settingsstore.New("")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(goos) == "" {
		goos = runtime.GOOS
	}
	return &Service{
		goos:             strings.ToLower(goos),
		settings:         store,
		netiface:         netiface.New(goos, runner),
		clash:            clash.New(goos, nil),
		proxytest:        proxytest.New(goos),
		tasks:            tasks.NewManager(),
		isAdmin:          cmdexec.IsAdmin,
		originalMTUByKey: map[string]int{},
	}, nil
}

func (service *Service) Status(ctx context.Context) core.SystemStatus {
	state := service.tasks.Snapshot()
	return core.SystemStatus{
		PlatformName:          service.goos,
		IsAdmin:               service.hasAdminPrivileges(),
		SupportsPersistentMTU: core.SupportsPersistentMTU(service.goos),
		Busy:                  state.Status == core.TaskStatusRunning || state.Status == core.TaskStatusStopping,
		CurrentTaskKind:       state.Kind,
		CurrentTaskStatus:     state.Status,
	}
}

func (service *Service) LoadSavedSettings(ctx context.Context) (core.SavedSettings, error) {
	return service.settings.Load()
}

func (service *Service) SaveSavedSettings(ctx context.Context, settings core.SavedSettings) (core.SavedSettings, error) {
	return service.settings.Save(settings)
}

func (service *Service) ListInterfaces(ctx context.Context) ([]core.InterfaceInfo, error) {
	return service.netiface.ListInterfaces(ctx)
}

func (service *Service) ResolveClashTarget(ctx context.Context, request core.ResolveTargetRequest) (core.ClashTarget, error) {
	return service.clash.ResolveCurrentTarget(ctx, request)
}

func (service *Service) RefreshInterface(ctx context.Context, info core.InterfaceInfo) (core.InterfaceCommandResult, error) {
	mtu, err := service.netiface.CurrentMTU(ctx, info)
	if err != nil {
		return core.InterfaceCommandResult{}, err
	}
	info.MTU = mtu
	originalMTU := service.originalMTU(info)
	if originalMTU == 0 {
		originalMTU = service.recordOriginalMTU(info)
	}
	return core.InterfaceCommandResult{
		Interface:   info,
		OriginalMTU: originalMTU,
	}, nil
}

func (service *Service) SetActiveMTU(ctx context.Context, info core.InterfaceInfo, mtu int) (core.InterfaceCommandResult, error) {
	if err := service.ensureAdminPrivilegesFor("changing MTU"); err != nil {
		return core.InterfaceCommandResult{}, err
	}
	output, err := service.netiface.SetMTU(ctx, info, mtu, false)
	if err != nil {
		return core.InterfaceCommandResult{}, err
	}
	return service.refreshAfterChange(ctx, info, output)
}

func (service *Service) RestoreMTU(ctx context.Context, info core.InterfaceInfo) (core.InterfaceCommandResult, error) {
	original, ok := service.lookupOriginalMTU(info)
	if !ok {
		return core.InterfaceCommandResult{}, fmt.Errorf("original MTU is unknown; detect interface first")
	}
	if err := service.ensureAdminPrivilegesFor("changing MTU"); err != nil {
		return core.InterfaceCommandResult{}, err
	}
	output, err := service.netiface.SetMTU(ctx, info, original, false)
	if err != nil {
		return core.InterfaceCommandResult{}, err
	}
	return service.refreshAfterChange(ctx, info, output)
}

func (service *Service) SetPersistentMTU(ctx context.Context, info core.InterfaceInfo, mtu int) (core.InterfaceCommandResult, error) {
	if err := service.ensureAdminPrivilegesFor("persisting MTU"); err != nil {
		return core.InterfaceCommandResult{}, err
	}
	output, err := service.netiface.SetMTU(ctx, info, mtu, true)
	if err != nil {
		return core.InterfaceCommandResult{}, err
	}
	return service.refreshAfterChange(ctx, info, output)
}

func (service *Service) TaskState(ctx context.Context) core.TaskState {
	return service.tasks.Snapshot()
}

func (service *Service) SubscribeTaskEvents(buffer int) *tasks.Subscription {
	return service.tasks.Subscribe(buffer)
}

func (service *Service) RunTestSync(ctx context.Context, request core.TestRunRequest) (core.TestSummary, error) {
	request, err := service.prepareTestRunRequest(ctx, request)
	if err != nil {
		return core.TestSummary{}, err
	}
	return service.proxytest.RunSuite(ctx, request, nil)
}

func (service *Service) RunSweepSync(ctx context.Context, request core.SweepRunRequest) (core.SweepResult, error) {
	if err := service.ensureAdminPrivilegesFor("running MTU sweep"); err != nil {
		return core.SweepResult{}, err
	}
	request, err := service.prepareSweepRunRequest(ctx, request)
	if err != nil {
		return core.SweepResult{}, err
	}
	return service.proxytest.RunSweep(ctx, service.netiface, request, nil, nil)
}

func (service *Service) StartTest(request core.TestRunRequest) (core.TaskState, error) {
	preparedRequest, err := service.prepareTestRunRequest(context.Background(), request)
	if err != nil {
		return core.TaskState{}, err
	}
	return service.tasks.Start(core.TaskKindConnectivityTest, func(controller *tasks.Controller) error {
		controller.Progress(0, 1, "starting")
		currentMTU, _ := service.netiface.CurrentMTU(controller.Context(), preparedRequest.Interface)
		controller.Log(fmt.Sprintf("Testing proxy=%s at MTU=%d with profile=%s...", preparedRequest.HTTPProxy, currentMTU, preparedRequest.TestProfile))
		progressLog := newTaskProgressLogEmitter("Test progress", controller.Log)

		summary, err := service.proxytest.RunSuite(controller.Context(), preparedRequest, func(done int, total int, label string) {
			normalizedLabel := strings.Split(label, "#")[0]
			controller.Progress(done, total, normalizedLabel)
			progressLog.Log(done, total, normalizedLabel)
		})
		if summary.Browser != "" {
			controller.Log("Browser: " + summary.Browser)
		}
		if summary.ProbeTransport != "" {
			controller.Log("Probe transport: " + summary.ProbeTransport)
		}
		controller.Log(fmt.Sprintf(
			"Result: %d/%d ok (planned %d), failures=%d, avg=%.3fs, p95=%.3fs, bytes=%d",
			summary.OK,
			summary.Total,
			summary.PlannedTotal,
			summary.Failures,
			summary.Avg,
			summary.P95,
			summary.Bytes,
		))
		if summary.Cancelled {
			controller.Log("Test cancelled before all checks completed.")
		}
		if summary.FirstError != "" {
			controller.Log("First error: " + summary.FirstError)
		}
		if len(summary.FailByName) > 0 {
			controller.Log("Failures by target: " + flattenCountMap(summary.FailByName))
		}
		return err
	})
}

func (service *Service) StartSweep(request core.SweepRunRequest) (core.TaskState, error) {
	if err := service.ensureAdminPrivilegesFor("running MTU sweep"); err != nil {
		return core.TaskState{}, err
	}
	preparedRequest, err := service.prepareSweepRunRequest(context.Background(), request)
	if err != nil {
		return core.TaskState{}, err
	}
	return service.tasks.Start(core.TaskKindMTUSweep, func(controller *tasks.Controller) error {
		controller.Progress(0, 1, "starting sweep")
		progressLog := newTaskProgressLogEmitter("Sweep progress", controller.Log)
		result, err := service.proxytest.RunSweep(
			controller.Context(),
			service.netiface,
			preparedRequest,
			func(done int, total int, label string) {
				controller.Progress(done, total, label)
				progressLog.Log(done, total, label)
			},
			func(line string) {
				controller.Log(line)
			},
		)
		if err != nil {
			return err
		}
		controller.Log("CSV written: " + result.OutputPath)
		return nil
	})
}

func (service *Service) CancelTask() core.TaskState {
	return service.tasks.Cancel()
}

func (service *Service) refreshAfterChange(ctx context.Context, info core.InterfaceInfo, output string) (core.InterfaceCommandResult, error) {
	mtu, err := service.netiface.CurrentMTU(ctx, info)
	if err != nil {
		return core.InterfaceCommandResult{}, err
	}
	info.MTU = mtu
	return core.InterfaceCommandResult{
		Interface:   info,
		Output:      output,
		OriginalMTU: service.originalMTU(info),
	}, nil
}

func (service *Service) recordOriginalMTU(info core.InterfaceInfo) int {
	service.mu.Lock()
	defer service.mu.Unlock()
	key := core.InterfaceKey(info)
	if existing, ok := service.originalMTUByKey[key]; ok && existing > 0 {
		return existing
	}
	service.originalMTUByKey[key] = info.MTU
	return info.MTU
}

func (service *Service) lookupOriginalMTU(info core.InterfaceInfo) (int, bool) {
	service.mu.Lock()
	defer service.mu.Unlock()
	mtu, ok := service.originalMTUByKey[core.InterfaceKey(info)]
	return mtu, ok
}

func (service *Service) originalMTU(info core.InterfaceInfo) int {
	mtu, _ := service.lookupOriginalMTU(info)
	return mtu
}

func (service *Service) prepareTestRunRequest(ctx context.Context, request core.TestRunRequest) (core.TestRunRequest, error) {
	targets, err := service.loadConfiguredTestTargets(ctx, request.TestTargets)
	if err != nil {
		return core.TestRunRequest{}, err
	}
	request.TestTargets = targets
	return request, nil
}

func (service *Service) prepareSweepRunRequest(ctx context.Context, request core.SweepRunRequest) (core.SweepRunRequest, error) {
	targets, err := service.loadConfiguredTestTargets(ctx, request.TestTargets)
	if err != nil {
		return core.SweepRunRequest{}, err
	}
	request.TestTargets = targets
	return request, nil
}

func (service *Service) loadConfiguredTestTargets(ctx context.Context, targets []core.TestTarget) ([]core.TestTarget, error) {
	if len(targets) > 0 {
		return targets, nil
	}
	if service == nil || service.settings == nil {
		return core.DefaultTestTargets(), nil
	}

	// Task APIs only carry the selected profile; actual target selection is owned
	// by persisted settings so browser/stress/chrome/quick stay in sync.
	settings, err := service.settings.Load()
	if err != nil {
		return nil, err
	}
	if len(settings.TestTargets) == 0 {
		return core.DefaultTestTargets(), nil
	}
	return settings.TestTargets, nil
}

func flattenCountMap(values map[string]int) string {
	if len(values) == 0 {
		return ""
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, values[key]))
	}
	return strings.Join(parts, ", ")
}

var (
	defaultServiceMu sync.RWMutex
	defaultService   *Service
)

func SetDefaultService(service *Service) {
	defaultServiceMu.Lock()
	defer defaultServiceMu.Unlock()
	defaultService = service
}

func MustDefaultService() *Service {
	defaultServiceMu.RLock()
	defer defaultServiceMu.RUnlock()
	if defaultService == nil {
		panic("default app service is not configured")
	}
	return defaultService
}
