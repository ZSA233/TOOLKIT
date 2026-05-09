package tasks

import (
	"errors"
	projectapp "mtu-tuner/internal/app"
	"mtu-tuner/internal/core"
	"mtu-tuner/internal/tasks"
	providers "mtu-tuner/internal/views/providers"
)

type Router struct {
	_GenRouter
	service *projectapp.Service
}

func NewRouter() *Router {
	return &Router{
		service: projectapp.MustDefaultService(),
	}
}

func (impl *Router) GetCurrentTask(ctx *CTX_GetCurrentTask, req *REQ_GetCurrentTask) (rsp *RSP_GetCurrentTask, err error) {
	return taskStateProto(impl.service.TaskState(ctx)), nil
}

func (impl *Router) TaskEvents(
	ctx *CTX_TaskEvents,
	stream providers.Stream[OPEN_TaskEvents, TaskEventMessage, CLOSE_TaskEvents],
) error {
	subscription := impl.service.SubscribeTaskEvents(tasks.DefaultSubscriptionBuffer)
	defer subscription.Close()

	initialMessage, err := taskEventMessage(impl.service.TaskState(ctx))
	if err != nil {
		return abortTaskEvents(stream, 1011, err.Error())
	}
	if err := stream.Send(initialMessage); err != nil {
		return abortTaskEvents(stream, 1011, err.Error())
	}

	for {
		select {
		case <-stream.Done():
			return nil
		case event, ok := <-subscription.Events():
			if !ok {
				err := subscription.Err()
				if err == nil {
					return nil
				}
				code := 1011
				if errors.Is(err, tasks.ErrSubscriptionOverflow) {
					code = 1013
				}
				return abortTaskEvents(stream, code, err.Error())
			}

			message, err := taskEventMessage(event)
			if err != nil {
				return abortTaskEvents(stream, 1011, err.Error())
			}
			if err := stream.Send(message); err != nil {
				return abortTaskEvents(stream, 1011, err.Error())
			}
		}
	}
}

func (impl *Router) StartConnectivityTest(ctx *CTX_StartConnectivityTest, req *REQ_StartConnectivityTest) (rsp *RSP_StartConnectivityTest, err error) {
	if req == nil || req.B == nil || req.B.Interface == nil {
		return nil, nil
	}
	state, err := impl.service.StartTest(core.TestRunRequest{
		Interface:   taskInterfaceRefCore(req.B.Interface),
		HTTPProxy:   req.B.HttpProxy,
		TestProfile: req.B.TestProfile,
		BrowserPath: req.B.BrowserPath,
		Rounds:      req.B.Rounds,
		Concurrency: req.B.Concurrency,
	})
	if err != nil {
		return nil, err
	}
	return &RSP_StartConnectivityTest{
		State: taskStateProto(state),
	}, nil
}

func (impl *Router) StartMtuSweep(ctx *CTX_StartMtuSweep, req *REQ_StartMtuSweep) (rsp *RSP_StartMtuSweep, err error) {
	if req == nil || req.B == nil || req.B.Interface == nil {
		return nil, nil
	}
	state, err := impl.service.StartSweep(core.SweepRunRequest{
		Interface:   taskInterfaceRefCore(req.B.Interface),
		HTTPProxy:   req.B.HttpProxy,
		TestProfile: req.B.TestProfile,
		BrowserPath: req.B.BrowserPath,
		SweepMTUs:   req.B.SweepMtus,
		Rounds:      req.B.Rounds,
		Concurrency: req.B.Concurrency,
	})
	if err != nil {
		return nil, err
	}
	return &RSP_StartMtuSweep{
		State: taskStateProto(state),
	}, nil
}

func (impl *Router) CancelCurrentTask(ctx *CTX_CancelCurrentTask, req *REQ_CancelCurrentTask) (rsp *RSP_CancelCurrentTask, err error) {
	return &RSP_CancelCurrentTask{
		State: taskStateProto(impl.service.CancelTask()),
	}, nil
}
