package tasks

import (
	"testing"

	"mtu-tuner/internal/core"
)

func TestTaskStateDTOMapsCurrentTaskState(t *testing.T) {
	t.Parallel()

	state := core.TaskState{
		Kind:            "connectivity_test",
		Status:          core.TaskStatusRunning,
		CancelRequested: true,
	}

	got := taskStateDTO(state)
	if got.Kind != state.Kind || got.Status != state.Status || got.CancelRequested != state.CancelRequested {
		t.Fatalf("taskStateDTO() = %#v, want %#v", got, state)
	}
}

func TestTaskEventMessageEncodesKnownPayloads(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		event core.TaskEvent
		kind  string
		check func(t *testing.T, message *TaskEventMessage)
	}{
		{
			name: "state",
			event: core.TaskState{
				Kind:   "connectivity_test",
				Status: core.TaskStatusRunning,
			},
			kind: TaskEventMessageTypeState,
			check: func(t *testing.T, message *TaskEventMessage) {
				t.Helper()
				data, err := message.DecodeState()
				if err != nil {
					t.Fatalf("DecodeState() error = %v", err)
				}
				if data.Kind != "connectivity_test" || data.Status != core.TaskStatusRunning {
					t.Fatalf("DecodeState() = %#v", data)
				}
			},
		},
		{
			name: "progress",
			event: core.TaskProgress{
				Kind:  "mtu_sweep",
				Done:  2,
				Total: 5,
				Label: "MTU 1440",
			},
			kind: TaskEventMessageTypeProgress,
			check: func(t *testing.T, message *TaskEventMessage) {
				t.Helper()
				data, err := message.DecodeProgress()
				if err != nil {
					t.Fatalf("DecodeProgress() error = %v", err)
				}
				if data.Kind != "mtu_sweep" || data.Done != 2 || data.Total != 5 || data.Label != "MTU 1440" {
					t.Fatalf("DecodeProgress() = %#v", data)
				}
			},
		},
		{
			name: "log",
			event: core.TaskLog{
				Kind: "mtu_sweep",
				Line: "testing",
				TS:   "2026-05-09T00:00:00Z",
			},
			kind: TaskEventMessageTypeLog,
			check: func(t *testing.T, message *TaskEventMessage) {
				t.Helper()
				data, err := message.DecodeLog()
				if err != nil {
					t.Fatalf("DecodeLog() error = %v", err)
				}
				if data.Kind != "mtu_sweep" || data.Line != "testing" || data.Ts != "2026-05-09T00:00:00Z" {
					t.Fatalf("DecodeLog() = %#v", data)
				}
			},
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			message, err := taskEventMessage(tt.event)
			if err != nil {
				t.Fatalf("taskEventMessage() error = %v", err)
			}
			if message.Type != tt.kind {
				t.Fatalf("taskEventMessage().Type = %q, want %q", message.Type, tt.kind)
			}
			tt.check(t, message)
		})
	}
}
