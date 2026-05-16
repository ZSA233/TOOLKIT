package system

import (
	projectapp "mtu-tuner/internal/app"
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

func (impl *Router) GetSystemStatus(ctx *CTX_GetSystemStatus, req *REQ_GetSystemStatus) (rsp *RSP_GetSystemStatus, err error) {
	return systemStatusDTO(impl.service.Status(ctx)), nil
}
