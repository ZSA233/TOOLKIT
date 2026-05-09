import {
  DEFAULT_SESSION_SETTINGS,
  DEFAULT_TASK_PROGRESS,
  DEFAULT_TASK_STATE,
} from "../constants";
import type { DashboardNoticeTone, DashboardState } from "../types";
import type {
  DefaultConnectionClose,
  SystemStatus,
} from "../../../lib/api/api/runtime/models";

const TASK_EVENTS_WARNING_PREFIX = "Task event stream";

export const TASK_EVENTS_RECONNECT_DELAY_MS = 300;
export const NOTICE_TOAST_DURATION_MS = 2400;

export function createInitialState(): DashboardState {
  return {
    booting: true,
    ready: false,
    pendingAction: "",
    error: "",
    notice: "",
    noticeSerial: 0,
    noticeTone: "info",
    warning: "",
    status: null,
    taskState: DEFAULT_TASK_STATE,
    taskProgress: DEFAULT_TASK_PROGRESS,
    settings: {
      ...DEFAULT_SESSION_SETTINGS,
      test_targets: DEFAULT_SESSION_SETTINGS.test_targets.map((target) => ({
        ...target,
        profiles: [...target.profiles],
      })),
    },
    interfaceCandidates: [],
    selectedInterface: null,
    interfaceHint: "",
    selectionMode: "auto",
    detectingInterface: false,
    interfaceAction: "",
    originalMtu: null,
    clashTarget: null,
    logs: [],
    autoScrollLogs: true,
  };
}

export function isTaskStreamWarning(message: string): boolean {
  return message.startsWith(TASK_EVENTS_WARNING_PREFIX);
}

export function taskStreamWarningMessage(reason: string): string {
  return `${TASK_EVENTS_WARNING_PREFIX} unavailable: ${reason}. Reconnecting…`;
}

export function taskStreamCloseReason(info: DefaultConnectionClose | undefined): string {
  if (!info) {
    return "connection closed";
  }
  if (typeof info.reason === "string" && info.reason.trim() !== "") {
    return info.reason.trim();
  }
  if (typeof info.error === "string" && info.error.trim() !== "") {
    return info.error.trim();
  }
  if (typeof info.code === "number" && info.code > 0) {
    return `closed with code ${info.code}`;
  }
  return "connection closed";
}

export function errorMessage(error: unknown): string {
  if (error instanceof Error && error.message.trim() !== "") {
    return error.message;
  }
  return String(error);
}

export function noticePatch(
  current: Pick<DashboardState, "notice" | "noticeSerial" | "noticeTone">,
  notice: string,
  tone: DashboardNoticeTone = "info",
): Pick<DashboardState, "notice" | "noticeSerial" | "noticeTone"> {
  return {
    notice,
    noticeSerial: current.noticeSerial + 1,
    noticeTone: tone,
  };
}

export function clearNoticePatch(
  current: Pick<DashboardState, "notice" | "noticeSerial" | "noticeTone">,
): Pick<DashboardState, "notice" | "noticeSerial" | "noticeTone"> {
  if (current.notice === "") {
    return {
      notice: "",
      noticeSerial: current.noticeSerial,
      noticeTone: current.noticeTone,
    };
  }
  return noticePatch(current, "");
}

export function preserveNoticePatch(
  current: Pick<DashboardState, "notice" | "noticeSerial" | "noticeTone">,
): Pick<DashboardState, "notice" | "noticeSerial" | "noticeTone"> {
  return {
    notice: current.notice,
    noticeSerial: current.noticeSerial,
    noticeTone: current.noticeTone,
  };
}

export function platformName(value: DashboardState["status"] | SystemStatus | null): string {
  return (value?.platform_name ?? "").trim().toLowerCase();
}

export function buildAdminWarning(status: DashboardState["status"] | SystemStatus | null): string {
  if (!status || status.is_admin) {
    return "";
  }
  if (platformName(status) === "windows") {
    return "Administrator privileges are required for Apply MTU, Restore, Persist, and Sweep.";
  }
  return "Root privileges are required for Apply MTU, Restore, Persist, and Sweep.";
}
