package tasks

import (
	"errors"

	"mtu-tuner/internal/core"
	providers "mtu-tuner/internal/views/providers"
	apitypes "mtu-tuner/internal/views/routes/api/_gen_types"
	sharedroute "mtu-tuner/internal/views/routes/api/shared"
)

func taskStateDTO(state core.TaskState) *apitypes.TaskState {
	return &apitypes.TaskState{
		Kind:            state.Kind,
		Status:          state.Status,
		CancelRequested: state.CancelRequested,
	}
}

func taskProgressDTO(progress core.TaskProgress) *apitypes.TaskProgress {
	return &apitypes.TaskProgress{
		Kind:  progress.Kind,
		Done:  progress.Done,
		Total: progress.Total,
		Label: progress.Label,
	}
}

func taskLogDTO(log core.TaskLog) *apitypes.TaskLog {
	return &apitypes.TaskLog{
		Kind: log.Kind,
		Line: log.Line,
		Ts:   log.TS,
	}
}

func taskEventMessage(event core.TaskEvent) (*TaskEventMessage, error) {
	switch payload := event.(type) {
	case core.TaskState:
		return NewTaskEventMessageState(taskStateDTO(payload))
	case core.TaskProgress:
		return NewTaskEventMessageProgress(taskProgressDTO(payload))
	case core.TaskLog:
		return NewTaskEventMessageLog(taskLogDTO(payload))
	default:
		return nil, errors.New("unsupported task event payload")
	}
}

func abortTaskEvents(
	stream providers.Stream[OPEN_TaskEvents, TaskEventMessage, CLOSE_TaskEvents],
	code int,
	reason string,
) error {
	if err := stream.Abort(code, reason); err != nil {
		return err
	}
	return nil
}

func taskInterfaceRefCore(info *apitypes.InterfaceRef) core.InterfaceInfo {
	return sharedroute.InterfaceRefCore(info)
}
