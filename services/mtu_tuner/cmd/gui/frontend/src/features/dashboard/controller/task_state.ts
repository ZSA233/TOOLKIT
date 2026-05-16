import { DEFAULT_TASK_PROGRESS } from "../constants";
import type { TaskLogEvent, TaskProgressEvent } from "../types";
import type { TaskState } from "../../../lib/api/api/runtime/types";

export function isTaskBusy(status: string): boolean {
  return status === "running" || status === "stopping";
}

export function defaultTaskProgressForState(state: TaskState): TaskProgressEvent {
  if (!isTaskBusy(state.status)) {
    return DEFAULT_TASK_PROGRESS;
  }
  return {
    kind: normalizeTaskKind(state.kind ?? ""),
    done: 0,
    total: 1,
    label: state.cancel_requested ? "Stopping…" : "Running…",
  };
}

export function nextTaskProgressFromState(
  currentProgress: TaskProgressEvent,
  currentState: TaskState,
  nextState: TaskState,
): TaskProgressEvent {
  if (nextState.status === "idle") {
    if (
      currentProgress.kind !== "" &&
      (currentProgress.done > 0 || currentProgress.label !== DEFAULT_TASK_PROGRESS.label)
    ) {
      return currentProgress;
    }
    return DEFAULT_TASK_PROGRESS;
  }
  if (!isTaskBusy(nextState.status)) {
    return currentProgress;
  }
  const nextKind = normalizeTaskKind(nextState.kind ?? "");
  const currentKind = normalizeTaskKind(currentProgress.kind ?? "");
  if (!isTaskBusy(currentState.status) || currentKind !== nextKind) {
    return defaultTaskProgressForState(nextState);
  }
  return currentProgress;
}

export function maybeTaskLog(entry: TaskLogEvent): TaskLogEvent | null {
  if (entry.line.trim() === "") {
    return null;
  }
  return {
    kind: entry.kind ?? "",
    line: entry.line,
    ts: entry.ts || new Date().toISOString(),
  };
}

export function appendLogEntry(logs: TaskLogEvent[], entry: TaskLogEvent): TaskLogEvent[] {
  const next = [...logs, entry];
  if (next.length <= 400) {
    return next;
  }
  return next.slice(next.length - 400);
}

export function taskKindLabel(kind: string): string {
  switch (normalizeTaskKind(kind)) {
    case "connectivity_test":
      return "Connectivity test";
    case "mtu_sweep":
      return "MTU sweep";
    default:
      return "Task";
  }
}

export function normalizeTaskKind(kind: string): string {
  switch (kind) {
    case "test":
    case "connectivity_test":
      return "connectivity_test";
    case "sweep":
    case "mtu_sweep":
      return "mtu_sweep";
    default:
      return kind;
  }
}

export function isConnectivityTestKind(kind: string): boolean {
  return normalizeTaskKind(kind) === "connectivity_test";
}

export function isMtuSweepKind(kind: string): boolean {
  return normalizeTaskKind(kind) === "mtu_sweep";
}
