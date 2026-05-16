import {
  startTransition,
  useEffect,
  useEffectEvent,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import type { DashboardDeps } from "./deps";
import { createBootLog } from "./constants";
import {
  findCandidateByKey,
  interfaceKey,
  mergeInterfaceCandidates,
  splitDetectSelectionWarning,
  toInterfaceRef,
  updateCandidateSnapshot,
} from "./controller/interface_state";
import {
  buildAdminWarning,
  clearNoticePatch,
  createInitialState,
  errorMessage,
  isTaskStreamWarning,
  NOTICE_TOAST_DURATION_MS,
  noticePatch,
  platformName,
  preserveNoticePatch,
  TASK_EVENTS_RECONNECT_DELAY_MS,
  taskStreamCloseReason,
  taskStreamWarningMessage,
} from "./controller/runtime_state";
import {
  hasPendingAction,
  isConfigPendingAction,
  isInterfaceMutationPendingAction,
  isTaskPendingAction,
} from "./controller/pending_state";
import {
  parseOptionalPositiveInt,
  toDashboardSessionSettings,
  toPersistedSavedSettings,
  trimOptional,
} from "./controller/settings_state";
import {
  appendLogEntry,
  isConnectivityTestKind,
  isMtuSweepKind,
  isTaskBusy,
  maybeTaskLog,
  nextTaskProgressFromState,
  normalizeTaskKind,
  taskKindLabel,
} from "./controller/task_state";
import type {
  DashboardActions,
  DashboardDerivedState,
  DashboardNoticeTone,
  DashboardState,
  SessionSettings,
  TaskLogEvent,
  TaskProgressEvent,
} from "./types";
import type { InterfaceInfo, TaskState } from "../../lib/api/api/runtime/types";
import type { TaskEventMessage } from "../../lib/api/api/routes/api/tasks/types";

export function useDashboardController(deps: DashboardDeps): {
  state: DashboardState;
  derived: DashboardDerivedState;
  actions: DashboardActions;
} {
  const [state, setState] = useState<DashboardState>(createInitialState);
  const stateRef = useRef(state);
  const bootstrapStartedRef = useRef(false);
  const autoDetectStartedRef = useRef(false);

  useLayoutEffect(() => {
    stateRef.current = state;
  }, [state]);

  useEffect(() => {
    if (state.notice.trim() === "") {
      return;
    }
    const serial = state.noticeSerial;
    const timer = window.setTimeout(() => {
      startTransition(() => {
        setState((current) =>
          current.noticeSerial !== serial
            ? current
            : {
                ...current,
                ...clearNoticePatch(current),
              },
        );
      });
    }, NOTICE_TOAST_DURATION_MS);
    return () => {
      window.clearTimeout(timer);
    };
  }, [state.notice, state.noticeSerial]);

  const pushLog = useEffectEvent((line: string) => {
    const entry = createBootLog(line);
    startTransition(() => {
      setState((current) => ({
        ...current,
        logs: appendLogEntry(current.logs, entry),
      }));
    });
  });

  const reportError = useEffectEvent((label: string, error: unknown) => {
    const message = error instanceof Error ? error.message : String(error);
    startTransition(() => {
        setState((current) => ({
          ...current,
          error: `${label}: ${message}`,
          ...clearNoticePatch(current),
          pendingAction: "",
          detectingInterface: false,
          interfaceAction: "",
          booting: false,
      }));
    });
    pushLog(`ERROR [${label}] ${message}`);
  });

  const applyRuntimeSnapshot = useEffectEvent(
    (
      status: Awaited<ReturnType<DashboardDeps["api"]["systemClient"]["getSystemStatus"]>>,
      settings: Awaited<ReturnType<DashboardDeps["api"]["settingsClient"]["getCurrentSettings"]>>,
      taskState: Awaited<ReturnType<DashboardDeps["api"]["tasksClient"]["getCurrentTask"]>>,
    ) => {
      startTransition(() => {
        setState((current) => ({
          ...current,
          booting: false,
          ready: true,
          pendingAction: "",
          error: "",
          status,
          settings: {
            ...toDashboardSessionSettings(settings),
            clash_secret: current.settings.clash_secret,
            rounds: current.settings.rounds,
            concurrency: current.settings.concurrency,
          },
          taskState,
          taskProgress: nextTaskProgressFromState(current.taskProgress, current.taskState, taskState),
        }));
      });
    },
  );

  const refreshSettings = useEffectEvent(async () => {
    const [status, settings, taskState] = await Promise.all([
      deps.api.systemClient.getSystemStatus(),
      deps.api.settingsClient.getCurrentSettings(),
      deps.api.tasksClient.getCurrentTask(),
    ]);
    applyRuntimeSnapshot(status, settings, taskState);
  });

  const loadInterfaceCandidates = useEffectEvent(
    async (
      warningMessage: string,
      options: {
        preserveSelection?: boolean;
        pushWarningLog?: boolean;
      } = {},
    ) => {
      const response = await deps.api.networkClient.listInterfaces();
      startTransition(() => {
        setState((current) => {
          const nextCandidates = mergeInterfaceCandidates(null, response.interfaces);
          const currentKey = interfaceKey(current.selectedInterface);
          const retained = options.preserveSelection
            ? findCandidateByKey(nextCandidates, currentKey)
            : null;
          const hasSelection = retained !== null;
          return {
            ...current,
            detectingInterface: false,
            interfaceAction: "",
            interfaceCandidates: nextCandidates,
            selectedInterface: retained,
            interfaceHint: "",
            selectionMode: hasSelection ? current.selectionMode : "auto",
            warning: warningMessage,
            ...(hasSelection ? preserveNoticePatch(current) : clearNoticePatch(current)),
            originalMtu: hasSelection ? current.originalMtu : null,
          };
        });
      });
      if (options.pushWarningLog && warningMessage) {
        pushLog(`WARNING ${warningMessage}`);
      }
    },
  );

  const autoDetectInterface = useEffectEvent(
    async (
      clashCurrent = false,
      options: {
        background?: boolean;
        action?: DashboardState["interfaceAction"];
      } = {},
    ) => {
      const current = stateRef.current;
      startTransition(() => {
        setState((snapshot) => ({
          ...snapshot,
          detectingInterface: true,
          interfaceAction: options.background ? "" : (options.action ?? ""),
          error: "",
          interfaceHint: "",
          warning: "",
          ...(options.background ? preserveNoticePatch(snapshot) : clearNoticePatch(snapshot)),
        }));
      });

      try {
        const result = await deps.api.networkClient.detectInterface({
          json: {
            probe: current.settings.route_probe,
            fallback_probe: current.settings.fallback_probe,
            controller: current.settings.clash_api,
            secret: trimOptional(current.settings.clash_secret),
            group: current.settings.proxy_group,
            config_path: trimOptional(current.settings.config_path),
            clash_current: clashCurrent,
          },
        });
        const detectWarning = splitDetectSelectionWarning(result.selection.warning ?? "");

        startTransition(() => {
          setState((snapshot) => ({
            ...snapshot,
            detectingInterface: false,
            interfaceAction: "",
            selectedInterface: result.interface,
            interfaceCandidates: mergeInterfaceCandidates(result.interface, result.candidates),
            interfaceHint: detectWarning.interfaceHint,
            selectionMode: "auto",
            originalMtu: result.original_mtu,
            clashTarget: result.selection.target ?? snapshot.clashTarget,
            warning: detectWarning.globalWarning,
            ...(options.background
              ? preserveNoticePatch(snapshot)
              : noticePatch(
                  snapshot,
                  clashCurrent ? "Updated from current Clash route." : `Detected ${result.interface.name}.`,
                  "info",
                )),
            settings: result.selection.target?.config_path
              ? {
                  ...snapshot.settings,
                  config_path: result.selection.target.config_path,
                }
              : snapshot.settings,
          }));
        });

        if (!options.background) {
          pushLog(
            `Detected ${result.interface.name} MTU=${result.interface.mtu ?? "-"} via ${result.selection.probe_ip}.`,
          );
        }
        if (detectWarning.globalWarning) {
          pushLog(`WARNING ${detectWarning.globalWarning}`);
        }
      } catch (error) {
        const message = error instanceof Error ? error.message : String(error);
        try {
          await loadInterfaceCandidates(`Auto detect failed: ${message}`, {
            preserveSelection: true,
            pushWarningLog: true,
          });
        } catch (listError) {
          reportError("auto-detect", listError);
        }
      }
    },
  );

  // Keep bootstrap tied to the stable deps object only. Making Effect Events reactive here
  // causes repeated startup runs and duplicate startup work after ordinary state updates.
  useEffect(() => {
    if (bootstrapStartedRef.current) {
      return;
    }
    bootstrapStartedRef.current = true;

    let cancelled = false;
    void (async () => {
      try {
        await deps.ensureRuntime();
        if (cancelled) {
          return;
        }
        await refreshSettings();
        if (cancelled || autoDetectStartedRef.current) {
          return;
        }
        autoDetectStartedRef.current = true;
        await autoDetectInterface(false, { background: true });
      } catch (error) {
        if (!cancelled) {
          reportError("bootstrap", error);
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [deps]);

  const applyTaskStateSnapshot = useEffectEvent(
    (next: TaskState, options: { announceTransitions?: boolean } = {}) => {
      const announceTransitions = options.announceTransitions === true;
      const taskLabel = taskKindLabel(next.kind ?? "");
      startTransition(() => {
        setState((current) => ({
          ...current,
          taskState: next,
          taskProgress: nextTaskProgressFromState(current.taskProgress, current.taskState, next),
          pendingAction: "",
          error:
            announceTransitions && next.status === "failed"
              ? `${taskLabel} failed. Check logs for details.`
              : current.error,
          ...(announceTransitions && next.status === "completed"
            ? noticePatch(current, `${taskLabel} completed.`, "ok")
            : preserveNoticePatch(current)),
          warning:
            announceTransitions && next.status === "cancelled"
              ? `${taskLabel} cancelled.`
              : current.warning,
          status: current.status
            ? {
                ...current.status,
                busy: isTaskBusy(next.status),
                current_task_kind: next.kind,
                current_task_status: next.status,
              }
            : current.status,
        }));
      });
    },
  );

  const setTaskStreamWarning = useEffectEvent((reason: string) => {
    const warning = taskStreamWarningMessage(reason);
    startTransition(() => {
      setState((current) => ({
        ...current,
        warning,
      }));
    });
    pushLog(`WARNING ${warning}`);
  });

  const clearTaskStreamWarning = useEffectEvent(() => {
    startTransition(() => {
      setState((current) =>
        isTaskStreamWarning(current.warning)
          ? {
              ...current,
              warning: "",
            }
          : current,
      );
    });
  });

  const syncTaskStateAfterReconnect = useEffectEvent(async () => {
    try {
      const taskState = await deps.api.tasksClient.getCurrentTask();
      applyTaskStateSnapshot(taskState);
      pushLog("Task event stream reconnected.");
    } catch (error) {
      pushLog(`WARNING task-state resync after reconnect failed: ${errorMessage(error)}`);
    }
  });

  const handleTaskStateEvent = useEffectEvent((next: TaskState) => {
    applyTaskStateSnapshot(next, { announceTransitions: true });
  });

  const handleTaskProgressEvent = useEffectEvent((next: TaskProgressEvent) => {
    startTransition(() => {
      setState((current) => ({
        ...current,
        taskProgress: {
          ...next,
          kind: next.kind ?? "",
          total: Math.max(1, next.total),
          label: next.label || "Idle",
        },
      }));
    });
  });

  const handleTaskLogEvent = useEffectEvent((entry: TaskLogEvent) => {
    const next = maybeTaskLog(entry);
    if (next == null) {
      return;
    }
    startTransition(() => {
      setState((current) => ({
        ...current,
        logs: appendLogEntry(current.logs, next),
      }));
    });
  });

  const handleTaskEventMessage = useEffectEvent((message: TaskEventMessage) => {
    switch (message.type) {
      case "state":
        handleTaskStateEvent(message.data);
        return;
      case "progress":
        handleTaskProgressEvent(message.data);
        return;
      case "log":
        handleTaskLogEvent(message.data);
        return;
      default:
        return;
    }
  });

  useEffect(() => {
    let disposed = false;
    let connectedOnce = false;
    let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
    let stream: ReturnType<DashboardDeps["api"]["tasksClient"]["subscribeTaskEvents"]> | null = null;

    const clearReconnectTimer = () => {
      if (reconnectTimer != null) {
        clearTimeout(reconnectTimer);
        reconnectTimer = null;
      }
    };

    const scheduleReconnect = (reason: string) => {
      if (disposed || reconnectTimer != null) {
        return;
      }
      setTaskStreamWarning(reason);
      reconnectTimer = setTimeout(() => {
        reconnectTimer = null;
        void connectTaskEvents(true);
      }, TASK_EVENTS_RECONNECT_DELAY_MS);
    };

    const closeCurrentStream = async () => {
      const activeStream = stream;
      stream = null;
      if (activeStream == null) {
        return;
      }
      try {
        await activeStream.close(1000, "dashboard dispose");
      } catch {
        // Ignore close failures while the dashboard is being torn down.
      }
    };

    const connectTaskEvents = async (isReconnect: boolean) => {
      try {
        await deps.ensureRuntime();
        if (disposed) {
          return;
        }

        const bridge = deps.api.tasksClient.subscribeTaskEvents();
        stream = bridge;

        let disconnected = false;
        const cleanupMessage = bridge.onMessage((message) => {
          if (stream !== bridge || disposed) {
            return;
          }
          handleTaskEventMessage(message);
        });
        const cleanupClose = bridge.onClose((info) => {
          cleanupMessage();
          cleanupClose();
          if (disconnected || disposed || stream !== bridge) {
            return;
          }
          disconnected = true;
          stream = null;
          scheduleReconnect(taskStreamCloseReason(info));
        });

        try {
          await bridge.ready;
        } catch (error) {
          cleanupMessage();
          cleanupClose();
          if (disconnected || disposed || stream !== bridge) {
            return;
          }
          disconnected = true;
          stream = null;
          scheduleReconnect(errorMessage(error));
          return;
        }

        if (disposed || stream !== bridge) {
          cleanupMessage();
          cleanupClose();
          try {
            await bridge.close(1000, "dashboard dispose");
          } catch {
            // Ignore close failures while another stream instance takes over.
          }
          return;
        }

        clearTaskStreamWarning();
        if (connectedOnce && isReconnect) {
          await syncTaskStateAfterReconnect();
        }
        connectedOnce = true;
      } catch (error) {
        if (!disposed) {
          scheduleReconnect(errorMessage(error));
        }
      }
    };

    void connectTaskEvents(false);

    return () => {
      disposed = true;
      clearReconnectTimer();
      void closeCurrentStream();
    };
  }, [deps]);

  const refreshSelectedInterface = useEffectEvent(
    async (
      info: InterfaceInfo,
      options: {
        mode?: DashboardState["selectionMode"];
        notice?: string;
        noticeTone?: DashboardNoticeTone;
        pushSuccessLog?: boolean;
        pushNoticeLog?: boolean;
        action?: DashboardState["interfaceAction"];
      } = {},
    ) => {
      startTransition(() => {
        setState((current) => ({
          ...current,
          detectingInterface: true,
          interfaceAction: options.action ?? "",
          error: "",
        }));
      });
      try {
        const result = await deps.api.networkClient.refreshInterface({
          json: {
            interface: toInterfaceRef(info),
          },
        });
        startTransition(() => {
          setState((current) => ({
            ...current,
            detectingInterface: false,
            interfaceAction: "",
            selectedInterface: result.interface,
            interfaceCandidates: updateCandidateSnapshot(current.interfaceCandidates, result.interface),
            interfaceHint: options.mode === "manual" ? "" : current.interfaceHint,
            selectionMode: options.mode ?? current.selectionMode,
            originalMtu: result.original_mtu ?? current.originalMtu,
            ...noticePatch(current, options.notice ?? "Interface refreshed.", options.noticeTone ?? "ok"),
          }));
        });
        if (options.pushSuccessLog) {
          pushLog(`Refreshed interface MTU. Effective MTU=${result.interface.mtu ?? "-"}.`);
        } else if (options.pushNoticeLog && options.notice) {
          pushLog(options.notice);
        }
      } catch (error) {
        reportError("refresh-interface", error);
      }
    },
  );

  const withPending = async (label: DashboardState["pendingAction"], work: () => Promise<void>) => {
    startTransition(() => {
      setState((current) => ({
        ...current,
        pendingAction: label,
        error: "",
        ...clearNoticePatch(current),
      }));
    });
    try {
      await work();
    } catch (error) {
      reportError(label, error);
      return;
    }
    startTransition(() => {
      setState((current) => ({
        ...current,
        pendingAction: "",
      }));
    });
  };

  const requestAdminRelaunch = useEffectEvent(async (reason: string) => {
    const warningMessage = `Administrator privileges are required to ${reason}.`;
    try {
      const launched = await deps.promptAdminRelaunch(reason);
      if (launched) {
        startTransition(() => {
          setState((snapshot) => ({
            ...snapshot,
            pendingAction: "",
            error: "",
            warning: "",
            ...noticePatch(snapshot, "Relaunching as Administrator…", "warn"),
          }));
        });
        pushLog(`Requested administrator relaunch to ${reason}.`);
        return true;
      }
      startTransition(() => {
          setState((snapshot) => ({
            ...snapshot,
            pendingAction: "",
            error: "",
            ...clearNoticePatch(snapshot),
            warning: warningMessage,
          }));
      });
      pushLog(`WARNING ${warningMessage}`);
      return false;
    } catch (error) {
      reportError("admin-relaunch", error);
      return false;
    }
  });

  const ensureAdminAccess = useEffectEvent(async (reason: string) => {
    const current = stateRef.current;
    if (current.status?.is_admin) {
      return true;
    }
    if (platformName(current.status) === "windows") {
      await requestAdminRelaunch(reason);
      return false;
    }
    const warningMessage = `Root privileges are required to ${reason}. Restart the app with elevated permissions.`;
    startTransition(() => {
      setState((snapshot) => ({
        ...snapshot,
        error: "",
        ...clearNoticePatch(snapshot),
        warning: warningMessage,
      }));
    });
    pushLog(`WARNING ${warningMessage}`);
    return false;
  });

  const actions: DashboardActions = {
    updateSetting(key, value) {
      setState((current) => {
        const next = {
          ...current,
          settings: {
            ...current.settings,
            [key]: value,
          },
        };
        stateRef.current = next;
        return next;
      });
    },
    async saveSettings() {
      let savedOk = false;
      await withPending("save-settings", async () => {
        const current = stateRef.current;
        const saved = await deps.api.settingsClient.saveCurrentSettings({
          json: {
            settings: toPersistedSavedSettings(current.settings),
          },
        });
        startTransition(() => {
          setState((snapshot) => ({
            ...snapshot,
            settings: {
              ...toDashboardSessionSettings(saved),
              clash_secret: snapshot.settings.clash_secret,
              rounds: snapshot.settings.rounds,
              concurrency: snapshot.settings.concurrency,
            },
            ...noticePatch(snapshot, "Saved local configuration.", "ok"),
          }));
        });
        pushLog("Saved local configuration. Clash secret was not persisted.");
        savedOk = true;
      });
      return savedOk;
    },
    async reloadSettings() {
      await withPending("reload-settings", async () => {
        await refreshSettings();
        startTransition(() => {
          setState((current) => ({
            ...current,
            ...noticePatch(current, "Reloaded local configuration.", "ok"),
          }));
        });
        pushLog("Reloaded saved settings from the local config file.");
      });
    },
    async autoDetectInterface(clashCurrent = false) {
      await autoDetectInterface(clashCurrent, {
        background: false,
        action: clashCurrent ? "detect-clash-current" : "auto-detect",
      });
    },
    async selectInterface(key: string) {
      const candidate = findCandidateByKey(stateRef.current.interfaceCandidates, key);
      if (!candidate) {
        return;
      }
      startTransition(() => {
        setState((current) => ({
          ...current,
          selectedInterface: candidate,
          interfaceHint: "",
          selectionMode: "manual",
          warning: "",
          ...noticePatch(current, `Selected ${candidate.name}.`, "info"),
        }));
      });
      await refreshSelectedInterface(candidate, {
        mode: "manual",
        notice: `Selected ${candidate.name}.`,
        noticeTone: "info",
      });
    },
    async resolveClashTarget() {
      await withPending("resolve-clash-target", async () => {
        const current = stateRef.current;
        const target = await deps.api.networkClient.resolveClashTarget({
          json: {
            controller: current.settings.clash_api,
            secret: trimOptional(current.settings.clash_secret),
            group: current.settings.proxy_group,
            config_path: trimOptional(current.settings.config_path),
          },
        });
        startTransition(() => {
          setState((snapshot) => ({
            ...snapshot,
            clashTarget: target,
            ...noticePatch(snapshot, `Resolved Clash leaf ${target.leaf}.`, "info"),
            settings: target.config_path
              ? {
                  ...snapshot.settings,
                  config_path: target.config_path,
                }
              : snapshot.settings,
          }));
        });
        pushLog(`Resolved Clash target ${target.group} -> ${target.leaf} (${target.resolved_ip}).`);
      });
    },
    async refreshInterface() {
      const selectedInterface = stateRef.current.selectedInterface;
      if (!selectedInterface) {
        return;
      }
      await refreshSelectedInterface(selectedInterface, {
        action: "refresh-interface",
        pushSuccessLog: true,
      });
    },
    async setActiveMtu() {
      const current = stateRef.current;
      const selectedInterface = current.selectedInterface;
      if (!selectedInterface) {
        return;
      }
      if (!(await ensureAdminAccess("change MTU values"))) {
        return;
      }
      await withPending("set-active-mtu", async () => {
        const result = await deps.api.networkClient.applyInterfaceMtu({
          json: {
            interface: toInterfaceRef(selectedInterface),
            mtu: current.settings.target_mtu,
          },
        });
        startTransition(() => {
          setState((snapshot) => ({
            ...snapshot,
            selectedInterface: result.interface,
            interfaceCandidates: updateCandidateSnapshot(snapshot.interfaceCandidates, result.interface),
            originalMtu: result.original_mtu ?? snapshot.originalMtu,
            ...noticePatch(snapshot, `Applied MTU ${result.interface.mtu ?? current.settings.target_mtu}.`, "ok"),
          }));
        });
        if (result.output) {
          pushLog(result.output);
        }
        pushLog(`Applied active MTU ${result.interface.mtu ?? current.settings.target_mtu}.`);
      });
    },
    async restoreMtu() {
      const selectedInterface = stateRef.current.selectedInterface;
      if (!selectedInterface) {
        return;
      }
      if (!(await ensureAdminAccess("restore the original MTU"))) {
        return;
      }
      await withPending("restore-mtu", async () => {
        const result = await deps.api.networkClient.restoreInterfaceMtu({
          json: {
            interface: toInterfaceRef(selectedInterface),
          },
        });
        startTransition(() => {
          setState((current) => ({
            ...current,
            selectedInterface: result.interface,
            interfaceCandidates: updateCandidateSnapshot(current.interfaceCandidates, result.interface),
            originalMtu: result.original_mtu ?? current.originalMtu,
            ...noticePatch(current, `Restored MTU to ${result.interface.mtu ?? "-"}.`, "ok"),
          }));
        });
        if (result.output) {
          pushLog(result.output);
        }
        pushLog(`Restored MTU to ${result.interface.mtu ?? "-"}.`);
      });
    },
    async setPersistentMtu() {
      const current = stateRef.current;
      const selectedInterface = current.selectedInterface;
      if (!selectedInterface) {
        return;
      }
      if (!(await ensureAdminAccess("persist MTU changes"))) {
        return;
      }
      await withPending("set-persistent-mtu", async () => {
        const result = await deps.api.networkClient.persistInterfaceMtu({
          json: {
            interface: toInterfaceRef(selectedInterface),
            mtu: current.settings.target_mtu,
          },
        });
        startTransition(() => {
          setState((snapshot) => ({
            ...snapshot,
            selectedInterface: result.interface,
            interfaceCandidates: updateCandidateSnapshot(snapshot.interfaceCandidates, result.interface),
            originalMtu: result.original_mtu ?? snapshot.originalMtu,
            ...noticePatch(snapshot, `Persisted MTU ${result.interface.mtu ?? current.settings.target_mtu}.`, "ok"),
          }));
        });
        if (result.output) {
          pushLog(result.output);
        }
        pushLog(`Persisted MTU ${result.interface.mtu ?? current.settings.target_mtu}.`);
      });
    },
    async runTest() {
      const current = stateRef.current;
      if (isTaskBusy(current.taskState.status) && isConnectivityTestKind(current.taskState.kind ?? "")) {
        await actions.cancelTask();
        return;
      }
      const selectedInterface = current.selectedInterface;
      if (!selectedInterface) {
        return;
      }
      await withPending("run-test", async () => {
        const result = await deps.api.tasksClient.startConnectivityTest({
          json: {
            interface: toInterfaceRef(selectedInterface),
            http_proxy: current.settings.http_proxy,
            test_profile: current.settings.test_profile,
            browser_path: trimOptional(current.settings.browser_path),
            rounds: parseOptionalPositiveInt(current.settings.rounds),
            concurrency: parseOptionalPositiveInt(current.settings.concurrency),
          },
        });
        startTransition(() => {
          setState((snapshot) => ({
            ...snapshot,
            taskState: result.state,
            taskProgress: {
              kind: normalizeTaskKind(result.state.kind ?? "connectivity_test"),
              done: 0,
              total: 1,
              label: "Queued test task",
            },
            ...noticePatch(snapshot, "Started connectivity test task.", "info"),
          }));
        });
        pushLog(`Started ${current.settings.test_profile} test task on ${selectedInterface.name}.`);
      });
    },
    async runSweep() {
      const current = stateRef.current;
      if (isTaskBusy(current.taskState.status) && isMtuSweepKind(current.taskState.kind ?? "")) {
        await actions.cancelTask();
        return;
      }
      const selectedInterface = current.selectedInterface;
      if (!selectedInterface) {
        return;
      }
      if (!(await ensureAdminAccess("run MTU sweep tests"))) {
        return;
      }
      await withPending("run-sweep", async () => {
        const result = await deps.api.tasksClient.startMtuSweep({
          json: {
            interface: toInterfaceRef(selectedInterface),
            http_proxy: current.settings.http_proxy,
            test_profile: current.settings.test_profile,
            browser_path: trimOptional(current.settings.browser_path),
            sweep_mtus: current.settings.sweep_mtus,
            rounds: parseOptionalPositiveInt(current.settings.rounds),
            concurrency: parseOptionalPositiveInt(current.settings.concurrency),
          },
        });
        startTransition(() => {
          setState((snapshot) => ({
            ...snapshot,
            taskState: result.state,
            taskProgress: {
              kind: normalizeTaskKind(result.state.kind ?? "mtu_sweep"),
              done: 0,
              total: 1,
              label: "Queued sweep task",
            },
            ...noticePatch(snapshot, "Started MTU sweep task.", "info"),
          }));
        });
        pushLog(`Started MTU sweep with profile ${current.settings.test_profile}.`);
      });
    },
    async cancelTask() {
      await withPending("cancel-task", async () => {
        const result = await deps.api.tasksClient.cancelCurrentTask();
        startTransition(() => {
          setState((current) => ({
            ...current,
            taskState: result.state,
            ...noticePatch(current, "Cancellation requested.", "warn"),
          }));
        });
        pushLog(`Cancellation requested for ${result.state.kind || "current"} task.`);
      });
    },
    clearLogs() {
      setState((current) => ({
        ...current,
        logs: [],
        ...noticePatch(current, "Cleared log panel.", "info"),
      }));
    },
    toggleAutoScrollLogs() {
      setState((current) => ({
        ...current,
        autoScrollLogs: !current.autoScrollLogs,
      }));
    },
    async promptAdminRelaunch(reason = "apply MTU changes and run MTU sweeps") {
      const current = stateRef.current;
      if (current.status?.is_admin) {
        return true;
      }
      if (platformName(current.status) !== "windows") {
        const warningMessage = `Root privileges are required to ${reason}. Restart the app with elevated permissions.`;
        startTransition(() => {
          setState((snapshot) => ({
            ...snapshot,
            error: "",
            ...clearNoticePatch(snapshot),
            warning: warningMessage,
          }));
        });
        pushLog(`WARNING ${warningMessage}`);
        return false;
      }
      return requestAdminRelaunch(reason);
    },
    async pickClashConfigPath() {
      await withPending("pick-clash-config", async () => {
        const path = await deps.pickClashConfigPath();
        if (!path) {
          return;
        }
        startTransition(() => {
          setState((current) => ({
            ...current,
            settings: {
              ...current.settings,
              config_path: path,
            },
          }));
        });
      });
    },
    async pickBrowserPath() {
      await withPending("pick-browser-path", async () => {
        const path = await deps.pickBrowserPath();
        if (!path) {
          return;
        }
        startTransition(() => {
          setState((current) => ({
            ...current,
            settings: {
              ...current.settings,
              browser_path: path,
            },
          }));
        });
      });
    },
  };

  const derived = useMemo<DashboardDerivedState>(() => {
    const taskBusy = isTaskBusy(state.taskState.status);
    const pendingBusy = hasPendingAction(state.pendingAction);
    const configBusy = isConfigPendingAction(state.pendingAction);
    const interfaceMutationBusy = isInterfaceMutationPendingAction(state.pendingAction);
    const taskActionBusy = isTaskPendingAction(state.pendingAction);
    const busy = taskBusy || state.booting;
    const interfaceBusy = state.booting || taskBusy || interfaceMutationBusy;
    const currentTaskKind = normalizeTaskKind(state.taskState.kind ?? "");
    return {
      busy,
      interfaceBusy,
      taskBusy,
      currentTaskKind,
      canCancel: taskBusy && (isConnectivityTestKind(currentTaskKind) || isMtuSweepKind(currentTaskKind)),
      canRunTest:
        !!state.selectedInterface &&
        !interfaceMutationBusy &&
        !taskActionBusy &&
        (!taskBusy || isConnectivityTestKind(currentTaskKind)),
      canRunSweep:
        !!state.selectedInterface &&
        !interfaceMutationBusy &&
        !taskActionBusy &&
        (!taskBusy || isMtuSweepKind(currentTaskKind)),
      canPersist:
        !!state.selectedInterface &&
        !!state.status?.supports_persistent_mtu &&
        !state.booting &&
        !taskBusy &&
        !interfaceMutationBusy,
      canEditInterface: !!state.selectedInterface,
      canEditConfig: !state.booting && !taskBusy && !configBusy,
      canSelectInterface: !state.booting && !taskBusy && !interfaceMutationBusy,
      progressRatio: state.taskProgress.total > 0 ? state.taskProgress.done / state.taskProgress.total : 0,
      testButtonLabel:
        taskBusy && isConnectivityTestKind(currentTaskKind)
          ? state.taskState.cancel_requested
            ? "Stopping Test…"
            : "Cancel Test"
          : "Run Test",
      sweepButtonLabel:
        taskBusy && isMtuSweepKind(currentTaskKind)
          ? state.taskState.cancel_requested
            ? "Stopping Sweep…"
            : "Cancel Sweep"
          : "Run Sweep",
      selectedInterfaceKey: interfaceKey(state.selectedInterface),
      canPromptAdminRelaunch:
        !state.status?.is_admin &&
        platformName(state.status) === "windows" &&
        !state.booting &&
        !pendingBusy &&
        !taskBusy,
      adminWarning: buildAdminWarning(state.status),
    };
  }, [state]);

  return { state, derived, actions };
}
