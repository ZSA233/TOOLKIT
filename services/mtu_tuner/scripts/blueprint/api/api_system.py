from api_blueprint.includes import *
from blueprint.app import bp


class SystemStatus(Model):
    platform_name = String(description="runtime platform")
    is_admin = Bool(description="whether current process has admin/root privilege")
    supports_persistent_mtu = Bool(description="whether persistent MTU is supported")
    busy = Bool(description="whether a long task is running")
    current_task_kind = String(description="current long task kind", optional=True)
    current_task_status = String(description="current long task status")


with bp.group("/system") as views:
    views.GET(
        "/status",
        summary="Get runtime status",
        operation_id="GetSystemStatus",
    ).RSP(SystemStatus)
