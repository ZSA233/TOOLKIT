import type { DashboardPendingAction } from "../types";

const CONFIG_PENDING_ACTIONS: DashboardPendingAction[] = [
  "save-settings",
  "reload-settings",
  "pick-clash-config",
  "pick-browser-path",
];

const INTERFACE_MUTATION_ACTIONS: DashboardPendingAction[] = [
  "set-active-mtu",
  "restore-mtu",
  "set-persistent-mtu",
];

const TASK_PENDING_ACTIONS: DashboardPendingAction[] = [
  "run-test",
  "run-sweep",
  "cancel-task",
];

export function hasPendingAction(action: DashboardPendingAction): boolean {
  return action !== "";
}

export function isConfigPendingAction(action: DashboardPendingAction): boolean {
  return CONFIG_PENDING_ACTIONS.includes(action);
}

export function isInterfaceMutationPendingAction(action: DashboardPendingAction): boolean {
  return INTERFACE_MUTATION_ACTIONS.includes(action);
}

export function isTaskPendingAction(action: DashboardPendingAction): boolean {
  return TASK_PENDING_ACTIONS.includes(action);
}

export function pendingActionStatusLabel(action: DashboardPendingAction): string {
  switch (action) {
    case "":
      return "none";
    case "save-settings":
      return "saving-settings";
    case "reload-settings":
      return "reloading-settings";
    case "resolve-clash-target":
      return "resolving-clash";
    case "set-active-mtu":
      return "applying-mtu";
    case "restore-mtu":
      return "restoring-mtu";
    case "set-persistent-mtu":
      return "persisting-mtu";
    case "run-test":
      return "starting-test";
    case "run-sweep":
      return "starting-sweep";
    case "cancel-task":
      return "cancelling-task";
    case "pick-clash-config":
      return "picking-clash-config";
    case "pick-browser-path":
      return "picking-browser-path";
    default:
      return action;
  }
}
