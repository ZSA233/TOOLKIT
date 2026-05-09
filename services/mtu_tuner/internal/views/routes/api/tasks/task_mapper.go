package tasks

import (
	"errors"

	"mtu-tuner/internal/core"
	providers "mtu-tuner/internal/views/providers"
	protos "mtu-tuner/internal/views/routes/api/_gen_protos"
	sharedroute "mtu-tuner/internal/views/routes/api/shared"
)

func taskStateProto(state core.TaskState) *protos.TaskState {
	return &protos.TaskState{
		Kind:            state.Kind,
		Status:          state.Status,
		CancelRequested: state.CancelRequested,
	}
}

func taskProgressProto(progress core.TaskProgress) *protos.TaskProgress {
	return &protos.TaskProgress{
		Kind:  progress.Kind,
		Done:  progress.Done,
		Total: progress.Total,
		Label: progress.Label,
	}
}

func taskLogProto(log core.TaskLog) *protos.TaskLog {
	return &protos.TaskLog{
		Kind: log.Kind,
		Line: log.Line,
		Ts:   log.TS,
	}
}

func taskEventMessage(event core.TaskEvent) (*TaskEventMessage, error) {
	switch payload := event.(type) {
	case core.TaskState:
		return NewTaskEventMessageState(taskStateProto(payload))
	case core.TaskProgress:
		return NewTaskEventMessageProgress(taskProgressProto(payload))
	case core.TaskLog:
		return NewTaskEventMessageLog(taskLogProto(payload))
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

func taskInterfaceRefCore(info *protos.InterfaceRef) core.InterfaceInfo {
	return sharedroute.InterfaceRefCore(info)
}
