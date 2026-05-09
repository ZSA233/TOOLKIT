import type {
  ClashTarget,
  InterfaceInfo,
  SavedSettings,
  SystemStatus,
  TaskLog,
  TaskProgress,
  TaskState,
  TestTarget,
} from "../../lib/api/api/runtime/models";

export type NavKey = "overview" | "route" | "scan" | "logs";
export type InterfaceSelectionMode = "auto" | "manual";
export type DashboardNoticeTone = "info" | "ok" | "warn";
export type TestTargetProfile = "browser" | "stress" | "quick" | "chrome";
export type DashboardPendingAction =
  | ""
  | "save-settings"
  | "reload-settings"
  | "resolve-clash-target"
  | "set-active-mtu"
  | "restore-mtu"
  | "set-persistent-mtu"
  | "run-test"
  | "run-sweep"
  | "cancel-task"
  | "pick-clash-config"
  | "pick-browser-path";
export type DashboardInterfaceAction =
  | ""
  | "auto-detect"
  | "detect-clash-current"
  | "refresh-interface";

export interface TestTargetSettings extends Omit<TestTarget, "profiles"> {
  profiles: TestTargetProfile[];
}

export interface SessionSettings extends SavedSettings {
  clash_secret: string;
  rounds: string;
  concurrency: string;
  test_targets: TestTargetSettings[];
}

export type TaskProgressEvent = TaskProgress;

export type TaskLogEvent = TaskLog;

export interface DashboardState {
  booting: boolean;
  ready: boolean;
  pendingAction: DashboardPendingAction;
  error: string;
  notice: string;
  noticeSerial: number;
  noticeTone: DashboardNoticeTone;
  warning: string;
  status: SystemStatus | null;
  taskState: TaskState;
  taskProgress: TaskProgressEvent;
  settings: SessionSettings;
  interfaceCandidates: InterfaceInfo[];
  selectedInterface: InterfaceInfo | null;
  interfaceHint: string;
  selectionMode: InterfaceSelectionMode;
  detectingInterface: boolean;
  interfaceAction: DashboardInterfaceAction;
  originalMtu: number | null;
  clashTarget: ClashTarget | null;
  logs: TaskLogEvent[];
  autoScrollLogs: boolean;
}

export interface DashboardDerivedState {
  busy: boolean;
  interfaceBusy: boolean;
  taskBusy: boolean;
  currentTaskKind: string;
  canCancel: boolean;
  canRunTest: boolean;
  canRunSweep: boolean;
  canPersist: boolean;
  canEditInterface: boolean;
  canEditConfig: boolean;
  canSelectInterface: boolean;
  progressRatio: number;
  testButtonLabel: string;
  sweepButtonLabel: string;
  selectedInterfaceKey: string;
  canPromptAdminRelaunch: boolean;
  adminWarning: string;
}

export interface DashboardActions {
  updateSetting<K extends keyof SessionSettings>(key: K, value: SessionSettings[K]): void;
  saveSettings(): Promise<void>;
  reloadSettings(): Promise<void>;
  autoDetectInterface(clashCurrent?: boolean): Promise<void>;
  selectInterface(key: string): Promise<void>;
  resolveClashTarget(): Promise<void>;
  refreshInterface(): Promise<void>;
  setActiveMtu(): Promise<void>;
  restoreMtu(): Promise<void>;
  setPersistentMtu(): Promise<void>;
  runTest(): Promise<void>;
  runSweep(): Promise<void>;
  cancelTask(): Promise<void>;
  clearLogs(): void;
  toggleAutoScrollLogs(): void;
  promptAdminRelaunch(reason?: string): Promise<boolean>;
  pickClashConfigPath(): Promise<void>;
  pickBrowserPath(): Promise<void>;
}
