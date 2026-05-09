package settings

import (
	"errors"

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

func (router *Router) GetCurrentSettings(
	ctx *CTX_GetCurrentSettings, req *REQ_GetCurrentSettings,
) (rsp *RSP_GetCurrentSettings, err error) {
	settings, err := router.service.LoadSavedSettings(ctx)
	if err != nil {
		return nil, err
	}
	return savedSettingsProto(settings), nil
}

func (router *Router) SaveCurrentSettings(
	ctx *CTX_SaveCurrentSettings, req *REQ_SaveCurrentSettings,
) (rsp *RSP_SaveCurrentSettings, err error) {
	if req == nil || req.B == nil || req.B.Settings == nil {
		return nil, errors.New("settings payload is required")
	}
	settings, err := router.service.SaveSavedSettings(ctx, savedSettingsCore(req.B.Settings))
	if err != nil {
		return nil, err
	}
	return savedSettingsProto(settings), nil
}
