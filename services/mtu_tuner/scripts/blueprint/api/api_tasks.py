from api_blueprint.includes import *
from blueprint.app import bp
from blueprint.api.api_network import InterfaceRef


class TaskState(Model):
    kind = String(description="task kind", optional=True)
    status = String(description="task status")
    cancel_requested = Bool(description="whether cancellation was requested")


class TaskProgress(Model):
    kind = String(description="task kind")
    done = Int(description="completed steps")
    total = Int(description="total steps")
    label = String(description="current step label")


class TaskLog(Model):
    kind = String(description="task kind")
    line = String(description="log line")
    ts = String(description="event timestamp")


class TaskEventsOpen(Model):
    pass


class ConnectivityTestRequest(Model):
    interface = InterfaceRef(description="interface reference")
    http_proxy = String(description="http proxy")
    test_profile = String(description="test profile")
    browser_path = String(description="browser path", optional=True)
    rounds = Int(description="round override", optional=True)
    concurrency = Int(description="concurrency override", optional=True)


class MtuSweepRequest(Model):
    interface = InterfaceRef(description="interface reference")
    http_proxy = String(description="http proxy")
    test_profile = String(description="test profile")
    browser_path = String(description="browser path", optional=True)
    sweep_mtus = String(description="comma separated sweep mtus")
    rounds = Int(description="round override", optional=True)
    concurrency = Int(description="concurrency override", optional=True)


class StartTaskResponse(Model):
    state = TaskState(description="task state")


class CancelTaskResponse(Model):
    state = TaskState(description="task state")


with bp.group("/tasks") as views:
    views.GET(
        "/current",
        summary="Get long task state",
        operation_id="GetCurrentTask",
    ).RSP(TaskState)

    views.STREAM(
        "/events",
        summary="Subscribe task event stream",
        scope=ConnectionScope.SESSION,
        operation_id="TaskEvents",
    ).OPEN(
        TaskEventsOpen
    ).SERVER_MESSAGE(
        "TaskEventMessage",
        state=TaskState,
        progress=TaskProgress,
        log=TaskLog,
    )

    views.POST(
        "/connectivity-test",
        summary="Run quick test task",
        operation_id="StartConnectivityTest",
    ).REQ(ConnectivityTestRequest).RSP(StartTaskResponse)

    views.POST(
        "/mtu-sweep",
        summary="Run mtu sweep task",
        operation_id="StartMtuSweep",
    ).REQ(MtuSweepRequest).RSP(StartTaskResponse)

    views.POST(
        "/current/cancel",
        summary="Cancel current task",
        operation_id="CancelCurrentTask",
    ).RSP(CancelTaskResponse)
