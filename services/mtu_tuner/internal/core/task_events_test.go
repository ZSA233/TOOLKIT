package core

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestTaskEventJSONUsesFrontendFieldNames(t *testing.T) {
	t.Parallel()

	progressJSON, err := json.Marshal(TaskProgress{
		Kind:  TaskKindMTUSweep,
		Done:  3,
		Total: 8,
		Label: "MTU 1440",
	})
	if err != nil {
		t.Fatalf("json.Marshal(TaskProgress) error = %v", err)
	}
	expectedProgressJSON := fmt.Sprintf(`{"kind":"%s","done":3,"total":8,"label":"MTU 1440"}`, TaskKindMTUSweep)
	if string(progressJSON) != expectedProgressJSON {
		t.Fatalf("TaskProgress JSON = %s", progressJSON)
	}

	logJSON, err := json.Marshal(TaskLog{
		Kind: TaskKindMTUSweep,
		Line: "-- MTU 1440 --",
		TS:   "2026-05-06T12:00:00Z",
	})
	if err != nil {
		t.Fatalf("json.Marshal(TaskLog) error = %v", err)
	}
	expectedLogJSON := fmt.Sprintf(`{"kind":"%s","line":"-- MTU 1440 --","ts":"2026-05-06T12:00:00Z"}`, TaskKindMTUSweep)
	if string(logJSON) != expectedLogJSON {
		t.Fatalf("TaskLog JSON = %s", logJSON)
	}

	stateJSON, err := json.Marshal(TaskState{
		Kind:            TaskKindMTUSweep,
		Status:          TaskStatusRunning,
		CancelRequested: true,
	})
	if err != nil {
		t.Fatalf("json.Marshal(TaskState) error = %v", err)
	}
	expectedStateJSON := fmt.Sprintf(`{"kind":"%s","status":"running","cancel_requested":true}`, TaskKindMTUSweep)
	if string(stateJSON) != expectedStateJSON {
		t.Fatalf("TaskState JSON = %s", stateJSON)
	}
}
