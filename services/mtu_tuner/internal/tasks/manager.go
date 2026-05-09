package tasks

import (
	"context"
	"fmt"
	"sync"
	"time"

	"mtu-tuner/internal/core"
)

type Controller struct {
	ctx     context.Context
	kind    string
	manager *Manager
}

func (controller *Controller) Context() context.Context {
	return controller.ctx
}

func (controller *Controller) Kind() string {
	return controller.kind
}

func (controller *Controller) CancelRequested() bool {
	return controller.ctx.Err() != nil
}

func (controller *Controller) Log(line string) {
	controller.manager.emitLog(controller.kind, line)
}

func (controller *Controller) Progress(done int, total int, label string) {
	controller.manager.emitProgress(controller.kind, done, total, label)
}

type Manager struct {
	mu              sync.Mutex
	state           core.TaskState
	cancel          context.CancelFunc
	cancelRequested bool
	subscribers     map[*Subscription]struct{}
}

func NewManager() *Manager {
	return &Manager{
		state: core.TaskState{
			Status: core.TaskStatusIdle,
		},
		subscribers: map[*Subscription]struct{}{},
	}
}

func (manager *Manager) Snapshot() core.TaskState {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	return manager.state
}

func (manager *Manager) Start(kind string, work func(controller *Controller) error) (core.TaskState, error) {
	manager.mu.Lock()
	if manager.state.Status == core.TaskStatusRunning || manager.state.Status == core.TaskStatusStopping {
		state := manager.state
		manager.mu.Unlock()
		return state, fmt.Errorf("task %q is already running", state.Kind)
	}

	ctx, cancel := context.WithCancel(context.Background())
	manager.cancel = cancel
	manager.cancelRequested = false
	manager.state = core.TaskState{
		Kind:            kind,
		Status:          core.TaskStatusRunning,
		CancelRequested: false,
	}
	state := manager.state
	manager.mu.Unlock()

	manager.emitState(state)

	go func() {
		controller := &Controller{
			ctx:     ctx,
			kind:    kind,
			manager: manager,
		}
		err := work(controller)

		finalStatus := core.TaskStatusCompleted
		switch {
		case ctx.Err() != nil:
			finalStatus = core.TaskStatusCancelled
		case err != nil:
			controller.Log("ERROR: " + err.Error())
			finalStatus = core.TaskStatusFailed
		}
		manager.finish(kind, finalStatus)
	}()

	return state, nil
}

func (manager *Manager) Cancel() core.TaskState {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	if manager.state.Status != core.TaskStatusRunning {
		return manager.state
	}
	manager.cancelRequested = true
	manager.state.CancelRequested = true
	manager.state.Status = core.TaskStatusStopping
	if manager.cancel != nil {
		manager.cancel()
	}
	state := manager.state
	go manager.emitState(state)
	return state
}

func (manager *Manager) finish(kind string, status string) {
	manager.mu.Lock()
	manager.state = core.TaskState{
		Kind:            kind,
		Status:          status,
		CancelRequested: manager.cancelRequested,
	}
	finalState := manager.state
	manager.cancel = nil
	manager.cancelRequested = false
	manager.mu.Unlock()

	manager.emitState(finalState)

	manager.mu.Lock()
	manager.state = core.TaskState{Status: core.TaskStatusIdle}
	idleState := manager.state
	manager.mu.Unlock()
	manager.emitState(idleState)
}

func (manager *Manager) emitState(state core.TaskState) {
	manager.broadcast(state)
}

func (manager *Manager) emitProgress(kind string, done int, total int, label string) {
	if total < 1 {
		total = 1
	}
	progress := core.TaskProgress{
		Kind:  kind,
		Done:  done,
		Total: total,
		Label: label,
	}
	manager.broadcast(progress)
}

func (manager *Manager) emitLog(kind string, line string) {
	payload := core.TaskLog{
		Kind: kind,
		Line: line,
		TS:   time.Now().Format(time.RFC3339),
	}
	manager.broadcast(payload)
}
