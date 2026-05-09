package network

import (
	projectapp "mtu-tuner/internal/app"
	"mtu-tuner/internal/core"
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

func (router *Router) ListInterfaces(
	ctx *CTX_ListInterfaces, req *REQ_ListInterfaces,
) (rsp *RSP_ListInterfaces, err error) {
	interfaces, err := router.service.ListInterfaces(ctx)
	if err != nil {
		return nil, err
	}
	return &RSP_ListInterfaces{
		Interfaces: interfaceInfoListProto(interfaces),
	}, nil
}

func (router *Router) DetectInterface(
	ctx *CTX_DetectInterface, req *REQ_DetectInterface,
) (rsp *RSP_DetectInterface, err error) {
	if req == nil || req.B == nil {
		return nil, nil
	}
	result, err := router.service.DetectInterface(ctx, core.DetectRequest{
		Probe:         req.B.Probe,
		FallbackProbe: req.B.FallbackProbe,
		Controller:    req.B.Controller,
		Secret:        req.B.Secret,
		Group:         req.B.Group,
		ConfigPath:    req.B.ConfigPath,
		ClashCurrent:  req.B.ClashCurrent,
	})
	if err != nil {
		return nil, err
	}
	return &RSP_DetectInterface{
		Selection:   probeSelectionProto(result.Selection),
		Interface:   interfaceInfoProto(result.Interface),
		OriginalMtu: result.OriginalMTU,
		Candidates:  interfaceInfoListProto(result.Candidates),
	}, nil
}

func (router *Router) ResolveClashTarget(
	ctx *CTX_ResolveClashTarget, req *REQ_ResolveClashTarget,
) (rsp *RSP_ResolveClashTarget, err error) {
	if req == nil || req.B == nil {
		return nil, nil
	}
	target, err := router.service.ResolveClashTarget(ctx, core.ResolveTargetRequest{
		Controller: req.B.Controller,
		Secret:     req.B.Secret,
		Group:      req.B.Group,
		ConfigPath: req.B.ConfigPath,
	})
	if err != nil {
		return nil, err
	}
	return clashTargetProto(target), nil
}

func (router *Router) RefreshInterface(
	ctx *CTX_RefreshInterface, req *REQ_RefreshInterface,
) (rsp *RSP_RefreshInterface, err error) {
	if req == nil || req.B == nil || req.B.Interface == nil {
		return nil, nil
	}
	result, err := router.service.RefreshInterface(ctx, interfaceRefCore(req.B.Interface))
	if err != nil {
		return nil, err
	}
	return interfaceCommandResultProto(result), nil
}

func (router *Router) ApplyInterfaceMtu(
	ctx *CTX_ApplyInterfaceMtu, req *REQ_ApplyInterfaceMtu,
) (rsp *RSP_ApplyInterfaceMtu, err error) {
	if req == nil || req.B == nil || req.B.Interface == nil {
		return nil, nil
	}
	result, err := router.service.SetActiveMTU(ctx, interfaceRefCore(req.B.Interface), req.B.Mtu)
	if err != nil {
		return nil, err
	}
	return interfaceCommandResultProto(result), nil
}

func (router *Router) RestoreInterfaceMtu(
	ctx *CTX_RestoreInterfaceMtu, req *REQ_RestoreInterfaceMtu,
) (rsp *RSP_RestoreInterfaceMtu, err error) {
	if req == nil || req.B == nil || req.B.Interface == nil {
		return nil, nil
	}
	result, err := router.service.RestoreMTU(ctx, interfaceRefCore(req.B.Interface))
	if err != nil {
		return nil, err
	}
	return interfaceCommandResultProto(result), nil
}

func (router *Router) PersistInterfaceMtu(
	ctx *CTX_PersistInterfaceMtu, req *REQ_PersistInterfaceMtu,
) (rsp *RSP_PersistInterfaceMtu, err error) {
	if req == nil || req.B == nil || req.B.Interface == nil {
		return nil, nil
	}
	result, err := router.service.SetPersistentMTU(ctx, interfaceRefCore(req.B.Interface), req.B.Mtu)
	if err != nil {
		return nil, err
	}
	return interfaceCommandResultProto(result), nil
}
