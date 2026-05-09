from api_blueprint.includes import *
from blueprint.app import bp


class InterfaceRef(Model):
    platform_name = String(description="platform name")
    name = String(description="interface name")
    index = String(description="interface index", optional=True)


class InterfaceInfo(Model):
    platform_name = String(description="platform name")
    name = String(description="interface name")
    index = String(description="interface index", optional=True)
    mtu = Int(description="current mtu", optional=True)
    gateway = String(description="gateway", optional=True)
    local_address = String(description="local address", optional=True)
    description = String(description="interface description", optional=True)


class ClashTarget(Model):
    group = String(description="group chain")
    leaf = String(description="leaf node")
    server = String(description="server host")
    port = Int(description="server port", optional=True)
    resolved_ip = String(description="resolved ip")
    config_path = String(description="config path", optional=True)
    source = String(description="resolution source", optional=True)


class ProbeSelection(Model):
    probe_ip = String(description="probe ip")
    warning = String(description="warning", optional=True)
    target = ClashTarget(description="resolved clash target", optional=True)


class DetectInterfaceRequest(Model):
    probe = String(description="route probe")
    fallback_probe = String(description="fallback route probe")
    controller = String(description="clash api controller")
    secret = String(description="clash api secret", optional=True)
    group = String(description="clash group")
    config_path = String(description="clash config path", optional=True)
    clash_current = Bool(description="require current clash node")


class DetectInterfaceResponse(Model):
    selection = ProbeSelection(description="probe selection")
    interface = InterfaceInfo(description="detected interface")
    original_mtu = Int(description="recorded original mtu")
    candidates = Array[InterfaceInfo](description="candidate interfaces")


class InterfaceListResponse(Model):
    interfaces = Array[InterfaceInfo](description="candidate interfaces")


class InterfaceRefreshRequest(Model):
    interface = InterfaceRef(description="interface reference")


class InterfaceMtuCommandRequest(Model):
    interface = InterfaceRef(description="interface reference")
    mtu = Int(description="target mtu")


class InterfaceCommandResult(Model):
    interface = InterfaceInfo(description="interface snapshot")
    output = String(description="command output", optional=True)
    original_mtu = Int(description="recorded original mtu", optional=True)


class InterfaceRestoreRequest(Model):
    interface = InterfaceRef(description="interface reference")


class ResolveClashTargetRequest(Model):
    controller = String(description="clash api controller")
    secret = String(description="clash api secret", optional=True)
    group = String(description="clash group")
    config_path = String(description="clash config path", optional=True)


with bp.group("/network") as views:
    views.GET(
        "/interfaces",
        summary="List available interfaces",
        operation_id="ListInterfaces",
    ).RSP(InterfaceListResponse)

    views.POST(
        "/detect",
        summary="Detect route interface",
        operation_id="DetectInterface",
    ).REQ(DetectInterfaceRequest).RSP(DetectInterfaceResponse)

    views.POST(
        "/clash-target/resolve",
        summary="Resolve current clash target",
        operation_id="ResolveClashTarget",
    ).REQ(ResolveClashTargetRequest).RSP(ClashTarget)

    views.POST(
        "/interface/refresh",
        summary="Refresh current interface mtu",
        operation_id="RefreshInterface",
    ).REQ(InterfaceRefreshRequest).RSP(InterfaceCommandResult)

    views.POST(
        "/interface/mtu/apply",
        summary="Set active mtu",
        operation_id="ApplyInterfaceMtu",
    ).REQ(InterfaceMtuCommandRequest).RSP(InterfaceCommandResult)

    views.POST(
        "/interface/mtu/restore",
        summary="Restore recorded mtu",
        operation_id="RestoreInterfaceMtu",
    ).REQ(InterfaceRestoreRequest).RSP(InterfaceCommandResult)

    views.POST(
        "/interface/mtu/persist",
        summary="Set persistent mtu",
        operation_id="PersistInterfaceMtu",
    ).REQ(InterfaceMtuCommandRequest).RSP(InterfaceCommandResult)
