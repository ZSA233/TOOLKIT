package app

import (
	"context"
	"testing"
	"time"

	"mtu-tuner/internal/core"
	"mtu-tuner/internal/infra/netiface"
	"mtu-tuner/internal/proxytest"
	"mtu-tuner/internal/tasks"

	"toolkit/libs/appkit/cmdexec"
)

func TestStatusReportsRenamedTaskKinds(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		kind string
	}{
		{name: "connectivity test", kind: core.TaskKindConnectivityTest},
		{name: "mtu sweep", kind: core.TaskKindMTUSweep},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			manager := tasks.NewManager()
			release := make(chan struct{})
			started := make(chan struct{})

			if _, err := manager.Start(tt.kind, func(controller *tasks.Controller) error {
				close(started)
				<-release
				return nil
			}); err != nil {
				t.Fatalf("Start() error = %v", err)
			}
			defer close(release)

			<-started

			service := &Service{
				goos:    "darwin",
				tasks:   manager,
				isAdmin: func() bool { return true },
			}

			status := service.Status(nil)
			if !status.Busy {
				t.Fatal("Status().Busy = false, want true")
			}
			if status.CurrentTaskStatus != core.TaskStatusRunning {
				t.Fatalf("Status().CurrentTaskStatus = %q, want %q", status.CurrentTaskStatus, core.TaskStatusRunning)
			}
			if status.CurrentTaskKind != tt.kind {
				t.Fatalf("Status().CurrentTaskKind = %q, want %q", status.CurrentTaskKind, tt.kind)
			}
		})
	}
}

func TestStartTestPublishesConnectivityTaskKindAtRuntime(t *testing.T) {
	t.Parallel()

	service := &Service{
		goos:      "darwin",
		netiface:  netiface.New("darwin", &runnerStub{results: []cmdexec.Result{{ExitCode: 0, Stdout: "mtu 1500"}}}),
		proxytest: proxytest.New("darwin"),
		tasks:     tasks.NewManager(),
		isAdmin:   func() bool { return true },
	}

	subscription := service.SubscribeTaskEvents(8)
	defer subscription.Close()

	if _, err := service.StartTest(core.TestRunRequest{
		Interface:   core.InterfaceInfo{Name: "en0"},
		TestProfile: "unknown-profile",
	}); err != nil {
		t.Fatalf("StartTest() error = %v", err)
	}

	assertRuntimeTaskKind(t, service, subscription, core.TaskKindConnectivityTest)
	waitForTaskIdle(t, service.tasks)
}

func TestStartSweepPublishesMTUSweepTaskKindAtRuntime(t *testing.T) {
	t.Parallel()

	service := &Service{
		goos:      "darwin",
		proxytest: proxytest.New("darwin"),
		tasks:     tasks.NewManager(),
		isAdmin:   func() bool { return true },
	}

	subscription := service.SubscribeTaskEvents(8)
	defer subscription.Close()

	if _, err := service.StartSweep(core.SweepRunRequest{
		SweepMTUs: "bad-mtu-list",
	}); err != nil {
		t.Fatalf("StartSweep() error = %v", err)
	}

	assertRuntimeTaskKind(t, service, subscription, core.TaskKindMTUSweep)
	waitForTaskIdle(t, service.tasks)
}

func assertRuntimeTaskKind(t *testing.T, service *Service, subscription *tasks.Subscription, wantKind string) {
	t.Helper()

	deadline := time.After(2 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for running task state; last snapshot = %#v", service.TaskState(context.Background()))
		case event, ok := <-subscription.Events():
			if !ok {
				t.Fatalf("subscription closed early: %v", subscription.Err())
			}
			state, ok := event.(core.TaskState)
			if !ok || state.Status != core.TaskStatusRunning {
				continue
			}
			if state.Kind != wantKind {
				t.Fatalf("running TaskState.Kind = %q, want %q", state.Kind, wantKind)
			}
			status := service.Status(context.Background())
			if status.CurrentTaskKind != wantKind {
				t.Fatalf("Status().CurrentTaskKind = %q, want %q", status.CurrentTaskKind, wantKind)
			}
			return
		}
	}
}

func waitForTaskIdle(t *testing.T, manager *tasks.Manager) {
	t.Helper()

	deadline, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	for {
		if manager.Snapshot().Status == core.TaskStatusIdle {
			return
		}
		select {
		case <-deadline.Done():
			t.Fatalf("manager did not become idle; last state = %#v", manager.Snapshot())
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}
