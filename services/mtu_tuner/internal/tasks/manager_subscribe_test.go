package tasks

import (
	"errors"
	"testing"
	"time"

	"mtu-tuner/internal/core"
)

func TestManagerSubscriptionReceivesStateProgressAndLog(t *testing.T) {
	t.Parallel()

	manager := NewManager()
	subscription := manager.Subscribe(8)
	defer subscription.Close()

	if _, err := manager.Start(core.TaskKindMTUSweep, func(controller *Controller) error {
		controller.Progress(2, 5, "MTU 1440")
		controller.Log("testing MTU 1440")
		return nil
	}); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	var gotState core.TaskState
	var gotProgress core.TaskProgress
	var gotLog core.TaskLog

	deadline := time.After(2 * time.Second)
	for gotState.Status == "" || gotProgress.Label == "" || gotLog.Line == "" {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for task events: state=%#v progress=%#v log=%#v", gotState, gotProgress, gotLog)
		case event, ok := <-subscription.Events():
			if !ok {
				t.Fatalf("subscription closed early: %v", subscription.Err())
			}
			switch payload := event.(type) {
			case core.TaskState:
				if payload.Status == core.TaskStatusRunning {
					gotState = payload
				}
			case core.TaskProgress:
				gotProgress = payload
			case core.TaskLog:
				gotLog = payload
			}
		}
	}

	if gotState.Kind != core.TaskKindMTUSweep || gotState.Status != core.TaskStatusRunning {
		t.Fatalf("running TaskState = %#v", gotState)
	}
	if gotProgress.Kind != core.TaskKindMTUSweep || gotProgress.Done != 2 || gotProgress.Total != 5 || gotProgress.Label != "MTU 1440" {
		t.Fatalf("TaskProgress = %#v", gotProgress)
	}
	if gotLog.Kind != core.TaskKindMTUSweep || gotLog.Line != "testing MTU 1440" || gotLog.TS == "" {
		t.Fatalf("TaskLog = %#v", gotLog)
	}
}

func TestManagerSubscriptionOverflowDoesNotBlockTask(t *testing.T) {
	t.Parallel()

	manager := NewManager()
	subscription := manager.Subscribe(1)
	defer subscription.Close()

	if _, err := manager.Start(core.TaskKindConnectivityTest, func(controller *Controller) error {
		for index := 0; index < 128; index++ {
			controller.Log("step")
		}
		return nil
	}); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	waitForIdle(t, manager)

	deadline := time.After(2 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for overflowed subscription to close")
		case _, ok := <-subscription.Events():
			if ok {
				continue
			}
			if !errors.Is(subscription.Err(), ErrSubscriptionOverflow) {
				t.Fatalf("subscription.Err() = %v, want %v", subscription.Err(), ErrSubscriptionOverflow)
			}
			return
		}
	}
}
