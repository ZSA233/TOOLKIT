import { Api as WailsApi, ensureWailsRuntime } from "../../lib/api/api/transports/wailsv3";

type GeneratedClients = ReturnType<typeof WailsApi.createClients>;
type WailsRuntimeHandler = (payload?: unknown) => void;

type WailsRuntime = {
  Call?: {
    ByName?: (name: string, ...args: unknown[]) => Promise<unknown>;
  };
  Events?: {
    On?: (name: string, handler: WailsRuntimeHandler) => void;
    Off?: (name: string, handler?: WailsRuntimeHandler) => void;
  };
};

export interface DashboardApi {
  tasksClient: Pick<
    GeneratedClients["tasksClient"],
    | "getCurrentTask"
    | "subscribeTaskEvents"
    | "startConnectivityTest"
    | "startMtuSweep"
    | "cancelCurrentTask"
  >;
  networkClient: Pick<
    GeneratedClients["networkClient"],
    | "listInterfaces"
    | "detectInterface"
    | "resolveClashTarget"
    | "refreshInterface"
    | "applyInterfaceMtu"
    | "restoreInterfaceMtu"
    | "persistInterfaceMtu"
  >;
  settingsClient: Pick<
    GeneratedClients["settingsClient"],
    "getCurrentSettings" | "saveCurrentSettings"
  >;
  systemClient: Pick<GeneratedClients["systemClient"], "getSystemStatus">;
}

export interface DashboardDeps {
  api: DashboardApi;
  ensureRuntime(): Promise<void>;
  pickClashConfigPath(): Promise<string>;
  pickBrowserPath(): Promise<string>;
  promptAdminRelaunch(reason: string): Promise<boolean>;
}

const SHELL_BINDINGS = {
  clash: [
    "main.ShellService.PickClashConfigPath",
    "mtu-tuner/cmd/gui.ShellService.PickClashConfigPath",
  ],
  browser: [
    "main.ShellService.PickBrowserPath",
    "mtu-tuner/cmd/gui.ShellService.PickBrowserPath",
  ],
  adminRelaunch: [
    "main.ShellService.PromptAdminRelaunch",
    "mtu-tuner/cmd/gui.ShellService.PromptAdminRelaunch",
  ],
} as const;

let importedWailsRuntime: WailsRuntime | null = null;
let importedWailsRuntimePromise: Promise<WailsRuntime> | null = null;

function windowWailsRuntime(): WailsRuntime | null {
  if (typeof window === "undefined") {
    return null;
  }
  return (window as typeof window & { wails?: WailsRuntime }).wails ?? null;
}

function hasCallRuntime(runtime: WailsRuntime | null | undefined): runtime is WailsRuntime {
  return typeof runtime?.Call?.ByName === "function";
}

function hasEventRuntime(runtime: WailsRuntime | null | undefined): runtime is WailsRuntime {
  return typeof runtime?.Events?.On === "function";
}

function resolveRuntimeWith(
  predicate: (runtime: WailsRuntime | null | undefined) => boolean,
): WailsRuntime | null {
  const windowRuntime = windowWailsRuntime();
  if (predicate(windowRuntime)) {
    return windowRuntime;
  }
  if (predicate(importedWailsRuntime)) {
    return importedWailsRuntime;
  }
  return null;
}

async function importWailsRuntime(): Promise<WailsRuntime> {
  if (importedWailsRuntimePromise == null) {
    const dynamicImport = new Function("specifier", "return import(specifier)") as (
      specifier: string,
    ) => Promise<unknown>;
    importedWailsRuntimePromise = dynamicImport("/wails/runtime.js") as Promise<WailsRuntime>;
  }
  importedWailsRuntime = await importedWailsRuntimePromise;
  return importedWailsRuntime;
}

async function ensureDashboardWailsRuntime(): Promise<WailsRuntime> {
  const readyRuntime = resolveRuntimeWith(hasCallRuntime);
  if (readyRuntime != null) {
    return readyRuntime;
  }
  try {
    await ensureWailsRuntime();
  } catch {
    // Fall through to a direct runtime import. Some Wails setups expose window.wails
    // before the full runtime module is loaded, which leaves Call/Events incomplete.
  }
  const runtimeAfterEnsure = resolveRuntimeWith(hasCallRuntime);
  if (runtimeAfterEnsure != null) {
    return runtimeAfterEnsure;
  }
  await importWailsRuntime();
  const runtimeAfterImport = resolveRuntimeWith(hasCallRuntime);
  if (runtimeAfterImport != null) {
    return runtimeAfterImport;
  }
  await ensureWailsRuntime();
  const runtime = resolveRuntimeWith(hasCallRuntime);
  if (runtime != null) {
    return runtime;
  }
  throw new Error("Wails runtime Call.ByName is unavailable.");
}

async function ensureDashboardEventRuntime(): Promise<WailsRuntime> {
  const readyRuntime = resolveRuntimeWith(hasEventRuntime);
  if (readyRuntime != null) {
    return readyRuntime;
  }
  await importWailsRuntime();
  const runtime = resolveRuntimeWith(hasEventRuntime);
  if (runtime != null) {
    return runtime;
  }
  throw new Error("Wails runtime Events.On is unavailable.");
}

async function callStringByCandidates(bindingNames: readonly string[]): Promise<string> {
  const runtime = await ensureDashboardWailsRuntime();
  const caller = runtime.Call?.ByName;
  if (typeof caller !== "function") {
    throw new Error("Wails runtime Call.ByName is unavailable.");
  }

  let lastError: unknown = null;
  for (const name of bindingNames) {
    try {
      const result = await caller(name);
      return typeof result === "string" ? result : "";
    } catch (error) {
      lastError = error;
    }
  }

  if (lastError instanceof Error) {
    throw lastError;
  }
  throw new Error(`Wails binding not found: ${bindingNames[0]}`);
}

async function callBooleanByCandidates(bindingNames: readonly string[], ...args: unknown[]): Promise<boolean> {
  const runtime = await ensureDashboardWailsRuntime();
  const caller = runtime.Call?.ByName;
  if (typeof caller !== "function") {
    throw new Error("Wails runtime Call.ByName is unavailable.");
  }

  let lastError: unknown = null;
  for (const name of bindingNames) {
    try {
      const result = await caller(name, ...args);
      return result === true;
    } catch (error) {
      lastError = error;
    }
  }

  if (lastError instanceof Error) {
    throw lastError;
  }
  throw new Error(`Wails binding not found: ${bindingNames[0]}`);
}

export function createDashboardDeps(): DashboardDeps {
  return {
    api: WailsApi.createClients(),
    ensureRuntime() {
      return Promise.all([
        ensureDashboardWailsRuntime(),
        ensureDashboardEventRuntime(),
      ]).then(() => undefined);
    },
    pickClashConfigPath() {
      return callStringByCandidates(SHELL_BINDINGS.clash);
    },
    pickBrowserPath() {
      return callStringByCandidates(SHELL_BINDINGS.browser);
    },
    promptAdminRelaunch(reason) {
      return callBooleanByCandidates(SHELL_BINDINGS.adminRelaunch, reason);
    },
  };
}
