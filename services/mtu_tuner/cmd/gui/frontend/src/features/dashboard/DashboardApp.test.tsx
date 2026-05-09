import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import type { ApiStreamBridge } from "../../lib/api/api/runtime/client";
import type { TaskEventMessage } from "../../lib/api/api/routes/api/tasks/models";
import type {
  DefaultConnectionClose,
  InterfaceInfo,
  InterfaceRef,
} from "../../lib/api/api/runtime/models";
import type { DashboardDeps } from "./deps";
import { DEFAULT_SAVED_SETTINGS } from "./constants";
import { DashboardApp } from "./DashboardApp";

afterEach(() => {
  cleanup();
});

function mockViewportScrollbarWidth(scrollbarWidth: number) {
  const originalInnerWidth = Object.getOwnPropertyDescriptor(window, "innerWidth");
  const originalClientWidth = Object.getOwnPropertyDescriptor(
    document.documentElement,
    "clientWidth",
  );
  const viewportWidth = 1440;

  Object.defineProperty(window, "innerWidth", {
    configurable: true,
    value: viewportWidth,
  });
  Object.defineProperty(document.documentElement, "clientWidth", {
    configurable: true,
    value: viewportWidth - scrollbarWidth,
  });

  return () => {
    if (originalInnerWidth) {
      Object.defineProperty(window, "innerWidth", originalInnerWidth);
    }
    if (originalClientWidth) {
      Object.defineProperty(document.documentElement, "clientWidth", originalClientWidth);
    }
  };
}

const INTERFACE_EN0: InterfaceInfo = {
  platform_name: "Darwin",
  name: "en0",
  index: "7",
  mtu: 1500,
  gateway: "192.168.0.1",
  local_address: "192.168.0.23",
  description: "Wi-Fi",
};

const INTERFACE_EN1: InterfaceInfo = {
  platform_name: "Darwin",
  name: "en1",
  index: "8",
  mtu: 1460,
  gateway: "10.0.0.1",
  local_address: "10.0.0.12",
  description: "USB Ethernet",
};

function deferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

type StatusPayload = {
  platform_name: string;
  is_admin: boolean;
  supports_persistent_mtu: boolean;
  busy: boolean;
  current_task_kind: string;
  current_task_status: string;
};

class MockTaskEventsStream
  implements ApiStreamBridge<TaskEventMessage, DefaultConnectionClose>
{
  readonly mode = "mock";
  readonly routeId = "api.tasks.stream.events";
  readonly ready: Promise<void>;
  readonly closeCalls: Array<{ code?: number; reason?: string }> = [];

  private readonly messageListeners = new Set<(message: TaskEventMessage) => void>();
  private readonly closeListeners = new Set<(info: DefaultConnectionClose) => void>();
  private resolveReady!: () => void;
  private rejectReady!: (error?: unknown) => void;

  constructor(
    options: {
      readyState?: "resolved" | "pending" | "rejected";
      readyError?: Error;
    } = {},
  ) {
    this.ready = new Promise<void>((resolve, reject) => {
      this.resolveReady = resolve;
      this.rejectReady = reject;
    });

    const readyState = options.readyState ?? "resolved";
    Promise.resolve().then(() => {
      if (readyState === "rejected") {
        this.rejectReady(options.readyError ?? new Error("task event stream connect failed"));
        return;
      }
      if (readyState === "resolved") {
        this.resolveReady();
      }
    });
  }

  onMessage(listener: (message: TaskEventMessage) => void) {
    this.messageListeners.add(listener);
    return () => {
      this.messageListeners.delete(listener);
    };
  }

  onClose(listener: (info: DefaultConnectionClose) => void) {
    this.closeListeners.add(listener);
    return () => {
      this.closeListeners.delete(listener);
    };
  }

  async close(code?: number, reason?: string) {
    this.closeCalls.push({ code, reason });
  }

  emitMessage(message: TaskEventMessage) {
    this.messageListeners.forEach((listener) => listener(message));
  }

  emitClose(info: DefaultConnectionClose = {}) {
    this.closeListeners.forEach((listener) => listener(info));
  }

  resolve() {
    this.resolveReady();
  }

  reject(error?: Error) {
    this.rejectReady(error ?? new Error("task event stream connect failed"));
  }
}

function createMockDeps(): DashboardDeps & {
  emitTaskMessage(message: TaskEventMessage): void;
  emitTaskClose(info?: DefaultConnectionClose): void;
  queueTaskStream(stream?: MockTaskEventsStream): MockTaskEventsStream;
  latestTaskStream(): MockTaskEventsStream;
  getSavedPayload(): Record<string, unknown> | undefined;
  setStatusPayload(payload: StatusPayload): void;
} {
  let storedSettings = { ...DEFAULT_SAVED_SETTINGS };
  let lastSavedPayload: Record<string, unknown> | undefined;
  let statusPayload: StatusPayload = {
    platform_name: "darwin",
    is_admin: false,
    supports_persistent_mtu: false,
    busy: false,
    current_task_kind: "",
    current_task_status: "idle",
  };
  const queuedTaskStreams: MockTaskEventsStream[] = [new MockTaskEventsStream()];
  const taskStreams: MockTaskEventsStream[] = [];

  const deps: DashboardDeps = {
    ensureRuntime: vi.fn(async () => {}),
    pickClashConfigPath: vi.fn(async () => "/tmp/clash.yaml"),
    pickBrowserPath: vi.fn(async () => "/Applications/Chromium.app"),
    promptAdminRelaunch: vi.fn(async () => false),
    api: {
      systemClient: {
        getSystemStatus: vi.fn(async () => ({ ...statusPayload })),
      },
      settingsClient: {
        getCurrentSettings: vi.fn(async () => ({ ...storedSettings })),
        saveCurrentSettings: vi.fn(async ({ json }: { json?: { settings: typeof storedSettings } }) => {
          storedSettings = { ...(json?.settings ?? storedSettings) };
          lastSavedPayload = storedSettings as Record<string, unknown>;
          return { ...storedSettings };
        }),
      },
      networkClient: {
        listInterfaces: vi.fn(async () => ({
          interfaces: [{ ...INTERFACE_EN0 }, { ...INTERFACE_EN1 }],
        })),
        detectInterface: vi.fn(async () => ({
          selection: {
            probe_ip: "1.1.1.1",
            warning: "",
          },
          interface: { ...INTERFACE_EN0 },
          original_mtu: 1500,
          candidates: [{ ...INTERFACE_EN0 }, { ...INTERFACE_EN1 }],
        })),
        resolveClashTarget: vi.fn(async () => ({
          group: "PROXY -> HK",
          leaf: "HK",
          server: "example.com",
          port: 443,
          resolved_ip: "104.16.0.1",
          config_path: "/tmp/clash.yaml",
          source: "system-dns",
        })),
        refreshInterface: vi.fn(async ({ json }: { json: { interface: InterfaceRef } }) => ({
          interface: {
            ...json.interface,
            mtu: json.interface.name === "en1" ? 1460 : 1500,
            gateway: json.interface.name === "en1" ? "10.0.0.1" : "192.168.0.1",
            local_address: json.interface.name === "en1" ? "10.0.0.12" : "192.168.0.23",
            description: json.interface.name === "en1" ? "USB Ethernet" : "Wi-Fi",
          },
          output: "",
          original_mtu: json.interface.name === "en1" ? 1460 : 1500,
        })),
        applyInterfaceMtu: vi.fn(async ({ json }: { json: { interface: InterfaceRef; mtu: number } }) => ({
          interface: {
            ...json.interface,
            mtu: json.mtu,
            gateway: json.interface.name === "en1" ? "10.0.0.1" : "192.168.0.1",
            local_address: json.interface.name === "en1" ? "10.0.0.12" : "192.168.0.23",
            description: json.interface.name === "en1" ? "USB Ethernet" : "Wi-Fi",
          },
          output: "",
          original_mtu: json.interface.name === "en1" ? 1460 : 1500,
        })),
        restoreInterfaceMtu: vi.fn(async ({ json }: { json: { interface: InterfaceRef } }) => ({
          interface: {
            ...json.interface,
            mtu: json.interface.name === "en1" ? 1460 : 1500,
            gateway: json.interface.name === "en1" ? "10.0.0.1" : "192.168.0.1",
            local_address: json.interface.name === "en1" ? "10.0.0.12" : "192.168.0.23",
            description: json.interface.name === "en1" ? "USB Ethernet" : "Wi-Fi",
          },
          output: "",
          original_mtu: json.interface.name === "en1" ? 1460 : 1500,
        })),
        persistInterfaceMtu: vi.fn(async ({ json }: { json: { interface: InterfaceRef; mtu: number } }) => ({
          interface: {
            ...json.interface,
            mtu: json.mtu,
            gateway: json.interface.name === "en1" ? "10.0.0.1" : "192.168.0.1",
            local_address: json.interface.name === "en1" ? "10.0.0.12" : "192.168.0.23",
            description: json.interface.name === "en1" ? "USB Ethernet" : "Wi-Fi",
          },
          output: "",
          original_mtu: json.interface.name === "en1" ? 1460 : 1500,
        })),
      },
      tasksClient: {
        getCurrentTask: vi.fn(async () => ({
          kind: "",
          status: "idle",
          cancel_requested: false,
        })),
        subscribeTaskEvents: vi.fn(() => {
          const stream = queuedTaskStreams.shift() ?? new MockTaskEventsStream();
          taskStreams.push(stream);
          return stream;
        }),
        startConnectivityTest: vi.fn(async () => ({
          state: {
            kind: "connectivity_test",
            status: "running",
            cancel_requested: false,
          },
        })),
        startMtuSweep: vi.fn(async () => ({
          state: {
            kind: "mtu_sweep",
            status: "running",
            cancel_requested: false,
          },
        })),
        cancelCurrentTask: vi.fn(async () => ({
          state: {
            kind: "connectivity_test",
            status: "stopping",
            cancel_requested: true,
          },
        })),
      },
    } as DashboardDeps["api"],
  };

  return Object.assign(deps, {
    emitTaskMessage(message: TaskEventMessage) {
      const stream = taskStreams.at(-1);
      if (!stream) {
        throw new Error("no task stream has been opened");
      }
      stream.emitMessage(message);
    },
    emitTaskClose(info: DefaultConnectionClose = {}) {
      const stream = taskStreams.at(-1);
      if (!stream) {
        throw new Error("no task stream has been opened");
      }
      stream.emitClose(info);
    },
    queueTaskStream(stream = new MockTaskEventsStream()) {
      queuedTaskStreams.push(stream);
      return stream;
    },
    latestTaskStream() {
      const stream = taskStreams.at(-1);
      if (!stream) {
        throw new Error("no task stream has been opened");
      }
      return stream;
    },
    getSavedPayload() {
      return lastSavedPayload;
    },
    setStatusPayload(payload: StatusPayload) {
      statusPayload = { ...payload };
    },
  });
}

describe("DashboardApp", () => {
  it("bootstraps with automatic interface detection and no startup success log", async () => {
    const deps = createMockDeps();
    const user = userEvent.setup();
    render(<DashboardApp deps={deps} />);

    await screen.findByTestId("boot-state");

    await waitFor(() => {
      expect(deps.api.networkClient.detectInterface).toHaveBeenCalledTimes(1);
      expect(deps.api.tasksClient.subscribeTaskEvents).toHaveBeenCalledTimes(1);
    });

    expect((screen.getByLabelText("Interface") as HTMLSelectElement).value).toBe("darwin|7");
    expect(
      screen.queryByText("Cross-platform MTU tuning without a local HTTP server"),
    ).toBeNull();

    await user.click(screen.getByRole("button", { name: "Logs" }));
    expect(screen.getByTestId("task-log-panel").textContent ?? "").not.toContain(
      "Wails runtime ready. Local settings and task state loaded.",
    );
  });

  it("falls back to candidate interfaces when automatic detection fails", async () => {
    const deps = createMockDeps();
    deps.api.networkClient.detectInterface = vi.fn(async () => {
      throw new Error("route lookup failed");
    });

    render(<DashboardApp deps={deps} />);

    await screen.findByTestId("boot-state");

    await waitFor(() => {
      expect(screen.getByText("Auto detect failed: route lookup failed")).toBeTruthy();
    });

    const select = screen.getByLabelText("Interface") as HTMLSelectElement;
    expect(select.value).toBe("");
    expect(Array.from(select.options).map((option) => option.textContent)).toContain(
      "en1 · 10.0.0.12 · MTU 1460",
    );
  });

  it("shows virtual-route interface hints beside the selector instead of as a global warning", async () => {
    const deps = createMockDeps();
    deps.api.networkClient.detectInterface = vi.fn(async () => ({
      selection: {
        probe_ip: "1.1.1.1",
        warning:
          "Current route resolves to virtual interface Meta. Selected underlying interface WLAN instead.",
      },
      interface: { ...INTERFACE_EN0 },
      original_mtu: 1500,
      candidates: [{ ...INTERFACE_EN0 }, { ...INTERFACE_EN1 }],
    }));

    render(<DashboardApp deps={deps} />);

    await screen.findByTestId("boot-state");

    await waitFor(() => {
      expect(
        screen.getByText(
          "Current route resolves to virtual interface Meta. Selected underlying interface WLAN instead.",
        ),
      ).toBeTruthy();
    });
  });

  it("uses manual interface selection for later MTU actions until auto detect runs again", async () => {
    const deps = createMockDeps();
    deps.setStatusPayload({
      platform_name: "darwin",
      is_admin: true,
      supports_persistent_mtu: false,
      busy: false,
      current_task_kind: "",
      current_task_status: "idle",
    });
    const user = userEvent.setup();
    render(<DashboardApp deps={deps} />);

    await screen.findByTestId("boot-state");
    await waitFor(() => {
      expect(deps.api.networkClient.detectInterface).toHaveBeenCalledTimes(1);
    });
    await screen.findByRole("option", { name: "en1 · 10.0.0.12 · MTU 1460" });

    await user.selectOptions(screen.getByLabelText("Interface"), "darwin|8");
    await waitFor(() => {
      expect(deps.api.networkClient.refreshInterface).toHaveBeenLastCalledWith({
        json: {
          interface: {
            platform_name: "Darwin",
            name: "en1",
            index: "8",
          },
        },
      });
    });

    await user.click(screen.getByRole("button", { name: "Apply MTU" }));
    await waitFor(() => {
      expect(deps.api.networkClient.applyInterfaceMtu).toHaveBeenCalledWith({
        json: {
          interface: {
            platform_name: "Darwin",
            name: "en1",
            index: "8",
          },
          mtu: 1400,
        },
      });
    });

    await user.click(screen.getByRole("button", { name: "Auto Detect" }));
    await waitFor(() => {
      expect(deps.api.networkClient.detectInterface).toHaveBeenCalledTimes(2);
    });
    expect((screen.getByLabelText("Interface") as HTMLSelectElement).value).toBe("darwin|7");
  });

  it("keeps interface controls visually stable while auto detect is running", async () => {
    const deps = createMockDeps();
    const user = userEvent.setup();
    render(<DashboardApp deps={deps} />);

    await screen.findByTestId("boot-state");

    const bootDetect = deps.api.networkClient.detectInterface as ReturnType<typeof vi.fn>;
    await waitFor(() => {
      expect(bootDetect).toHaveBeenCalledTimes(1);
    });

    const nextDetect = deferred<{
      selection: { probe_ip: string; warning: string };
      interface: InterfaceInfo;
      original_mtu: number;
      candidates: InterfaceInfo[];
    }>();
    const interactiveDetect = vi.fn(() => nextDetect.promise);
    deps.api.networkClient.detectInterface = interactiveDetect;

    const select = screen.getByLabelText("Interface") as HTMLSelectElement;
    const applyButton = screen.getByRole("button", { name: "Apply MTU" }) as HTMLButtonElement;
    expect(select.disabled).toBe(false);
    expect(applyButton.disabled).toBe(false);
    expect(screen.queryByText("Detecting…")).toBeNull();

    await user.click(screen.getByRole("button", { name: "Auto Detect" }));

    await waitFor(() => {
      expect(interactiveDetect).toHaveBeenCalledTimes(1);
    });

    expect(screen.getByRole("button", { name: "Auto Detect" }).getAttribute("aria-busy")).toBe("true");
    expect(select.disabled).toBe(false);
    expect(applyButton.disabled).toBe(false);
    expect(screen.getByText("Selected")).toBeTruthy();

    nextDetect.resolve({
      selection: {
        probe_ip: "8.8.8.8",
        warning: "",
      },
      interface: { ...INTERFACE_EN0 },
      original_mtu: 1500,
      candidates: [{ ...INTERFACE_EN0 }, { ...INTERFACE_EN1 }],
    });

    await waitFor(() => {
      expect(select.value).toBe("darwin|7");
    });
  });

  it("shows explicit busy feedback for secondary action buttons", async () => {
    const deps = createMockDeps();
    const user = userEvent.setup();
    render(<DashboardApp deps={deps} />);

    await screen.findByTestId("boot-state");

    const nextResolve = deferred<{
      group: string;
      leaf: string;
      server: string;
      port: number;
      resolved_ip: string;
      config_path: string;
      source: string;
    }>();
    const interactiveResolve = vi.fn(() => nextResolve.promise);
    deps.api.networkClient.resolveClashTarget = interactiveResolve;

    await user.click(screen.getByRole("button", { name: "Route & Clash" }));
    await user.click(screen.getByRole("button", { name: "Resolve Clash" }));

    await waitFor(() => {
      expect(interactiveResolve).toHaveBeenCalledTimes(1);
    });

    const busyButton = screen.getByRole("button", { name: "Resolve Clash" });
    expect(busyButton.getAttribute("aria-busy")).toBe("true");
    expect(busyButton.className).toContain("action-button--busy");

    nextResolve.resolve({
      group: "PROXY -> HK",
      leaf: "HK",
      server: "example.com",
      port: 443,
      resolved_ip: "104.16.0.1",
      config_path: "/tmp/clash.yaml",
      source: "system-dns",
    });

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Resolve Clash" })).toBeTruthy();
    });
  });

  it("loads persisted settings and refreshes them on reload", async () => {
    const deps = createMockDeps();
    const user = userEvent.setup();
    await deps.api.settingsClient.saveCurrentSettings({
      json: {
        settings: {
          ...DEFAULT_SAVED_SETTINGS,
          route_probe: "8.8.8.8",
        },
      },
    });

    render(<DashboardApp deps={deps} />);

    await screen.findByTestId("boot-state");
    await user.click(screen.getByRole("button", { name: "Route & Clash" }));
    expect((screen.getByLabelText("Route Probe") as HTMLInputElement).value).toBe("8.8.8.8");

    await deps.api.settingsClient.saveCurrentSettings({
      json: {
        settings: {
          ...DEFAULT_SAVED_SETTINGS,
          route_probe: "9.9.9.9",
        },
      },
    });
    await user.click(screen.getByRole("button", { name: "Reload" }));

    await waitFor(() => {
      expect((screen.getByLabelText("Route Probe") as HTMLInputElement).value).toBe("9.9.9.9");
    });
  });

  it("switches test controls into cancel mode while busy", async () => {
    const deps = createMockDeps();
    render(<DashboardApp deps={deps} />);

    await screen.findByTestId("boot-state");

    deps.emitTaskMessage({
      type: "state",
      data: {
        kind: "connectivity_test",
        status: "running",
        cancel_requested: false,
      },
    });

    await waitFor(() => {
      expect((screen.getByTestId("detect-route") as HTMLButtonElement).disabled).toBe(true);
      expect(screen.getByTestId("test-action").textContent).toBe("Cancel Test");
    });

    deps.emitTaskMessage({
      type: "state",
      data: {
        kind: "connectivity_test",
        status: "stopping",
        cancel_requested: true,
      },
    });

    await waitFor(() => {
      expect(screen.getByTestId("test-action").textContent).toBe("Stopping Test…");
    });
  });

  it("updates progress and logs from runtime events", async () => {
    const deps = createMockDeps();
    const user = userEvent.setup();
    render(<DashboardApp deps={deps} />);

    await screen.findByTestId("boot-state");

    deps.emitTaskMessage({
      type: "progress",
      data: {
        kind: "mtu_sweep",
        done: 3,
        total: 8,
        label: "MTU 1440",
      },
    });
    deps.emitTaskMessage({
      type: "log",
      data: {
        kind: "mtu_sweep",
        line: "restoring original mtu",
        ts: "2026-05-05T12:00:00Z",
      },
    });

    await waitFor(() => {
      expect(screen.getByTestId("task-progress-label").textContent).toBe("MTU 1440");
    });

    await user.click(screen.getByRole("button", { name: "Logs" }));
    expect(screen.getByTestId("task-log-panel").textContent ?? "").toContain(
      "restoring original mtu",
    );
  });

  it("keeps the final sweep progress visible after the task returns to idle", async () => {
    const deps = createMockDeps();
    render(<DashboardApp deps={deps} />);

    await screen.findByTestId("boot-state");

    deps.emitTaskMessage({
      type: "progress",
      data: {
        kind: "mtu_sweep",
        done: 8,
        total: 8,
        label: "MTU 1360 complete",
      },
    });
    deps.emitTaskMessage({
      type: "state",
      data: {
        kind: "mtu_sweep",
        status: "completed",
        cancel_requested: false,
      },
    });
    deps.emitTaskMessage({
      type: "state",
      data: {
        kind: "",
        status: "idle",
        cancel_requested: false,
      },
    });

    await waitFor(() => {
      expect(screen.getByTestId("task-progress-label").textContent).toBe("MTU 1360 complete");
      expect(screen.getByText("8/8")).toBeTruthy();
      expect(screen.getByText("Task: idle")).toBeTruthy();
    });
  });

  it("reconnects the task stream after an unexpected close and resyncs task state", async () => {
    const deps = createMockDeps();
    const reconnectStream = deps.queueTaskStream();
    render(<DashboardApp deps={deps} />);

    await screen.findByTestId("boot-state");

    const taskState = deps.api.tasksClient.getCurrentTask as ReturnType<typeof vi.fn>;
    await waitFor(() => {
      expect(deps.api.tasksClient.subscribeTaskEvents).toHaveBeenCalledTimes(1);
      expect(taskState).toHaveBeenCalledTimes(1);
    });

    deps.emitTaskClose({ code: 1011, reason: "backend reset" });

    await waitFor(() => {
      expect(screen.getByText("Task event stream unavailable: backend reset. Reconnecting…")).toBeTruthy();
    });

    await waitFor(() => {
      expect(deps.api.tasksClient.subscribeTaskEvents).toHaveBeenCalledTimes(2);
    });

    reconnectStream.emitMessage({
      type: "state",
      data: {
        kind: "mtu_sweep",
        status: "running",
        cancel_requested: false,
      },
    });

    await waitFor(() => {
      expect(taskState).toHaveBeenCalledTimes(2);
      expect(screen.getByRole("button", { name: "Cancel Sweep" })).toBeTruthy();
      expect(screen.queryByText("Task event stream unavailable: backend reset. Reconnecting…")).toBeNull();
    });
  });

  it("defaults rounds and concurrency to compact sweep-friendly values", async () => {
    const deps = createMockDeps();
    const user = userEvent.setup();
    render(<DashboardApp deps={deps} />);

    await screen.findByTestId("boot-state");
    await user.click(screen.getByRole("button", { name: "Scan" }));

    expect((screen.getByLabelText("Rounds") as HTMLInputElement).value).toBe("1");
    expect((screen.getByLabelText("Concurrency") as HTMLInputElement).value).toBe("1");
  });

  it("passes rounds and concurrency to sweep requests", async () => {
    const deps = createMockDeps();
    deps.setStatusPayload({
      platform_name: "windows",
      is_admin: true,
      supports_persistent_mtu: true,
      busy: false,
      current_task_kind: "",
      current_task_status: "idle",
    });
    const user = userEvent.setup();
    render(<DashboardApp deps={deps} />);

    await screen.findByTestId("boot-state");
    await waitFor(() => {
      expect(deps.api.networkClient.detectInterface).toHaveBeenCalledTimes(1);
    });

    await user.click(screen.getByRole("button", { name: "Run Sweep" }));

    await waitFor(() => {
      expect(deps.api.tasksClient.startMtuSweep).toHaveBeenCalledWith({
        json: expect.objectContaining({
          rounds: 1,
          concurrency: 1,
        }),
      });
    });
  });

  it("prompts for administrator relaunch before sweep when running on Windows without elevation", async () => {
    const deps = createMockDeps();
    deps.setStatusPayload({
      platform_name: "windows",
      is_admin: false,
      supports_persistent_mtu: true,
      busy: false,
      current_task_kind: "",
      current_task_status: "idle",
    });
    const user = userEvent.setup();
    render(<DashboardApp deps={deps} />);

    await screen.findByTestId("boot-state");
    await waitFor(() => {
      expect(deps.api.networkClient.detectInterface).toHaveBeenCalledTimes(1);
    });

    await user.click(screen.getByRole("button", { name: "Run Sweep" }));

    await waitFor(() => {
      expect(deps.promptAdminRelaunch).toHaveBeenCalledWith("run MTU sweep tests");
      expect(deps.api.tasksClient.startMtuSweep).not.toHaveBeenCalled();
    });
  });

  it("toggles log auto-scroll from the logs section", async () => {
    const deps = createMockDeps();
    const user = userEvent.setup();
    render(<DashboardApp deps={deps} />);

    await screen.findByTestId("boot-state");
    await user.click(screen.getByRole("button", { name: "Logs" }));

    expect(screen.getByRole("button", { name: "Auto Scroll On" })).toBeTruthy();
    await user.click(screen.getByRole("button", { name: "Auto Scroll On" }));
    expect(screen.getByRole("button", { name: "Auto Scroll Off" })).toBeTruthy();
  });

  it("follows the current scroll section in the sidebar", async () => {
    const deps = createMockDeps();
    const view = render(<DashboardApp deps={deps} />);

    await screen.findByTestId("boot-state");

    Object.defineProperty(window, "innerHeight", {
      configurable: true,
      value: 900,
    });

    mockSectionRect(view.container, "overview", { top: -420, bottom: -24 });
    mockSectionRect(view.container, "route", { top: 40, bottom: 520 });
    mockSectionRect(view.container, "scan", { top: 560, bottom: 1120 });
    mockSectionRect(view.container, "logs", { top: 1180, bottom: 1700 });

    window.dispatchEvent(new Event("scroll"));

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Route & Clash" }).className).toContain(
        "nav-link--active",
      );
    });
  });

  it("keeps clash secret out of persisted payloads", async () => {
    const deps = createMockDeps();
    const user = userEvent.setup();
    render(<DashboardApp deps={deps} />);

    await screen.findByTestId("boot-state");
    await user.click(screen.getByRole("button", { name: "Route & Clash" }));

    await user.type(screen.getByLabelText("Clash Secret"), "top-secret");
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      const saved = deps.getSavedPayload();
      expect(saved).toBeDefined();
      expect(Object.prototype.hasOwnProperty.call(saved ?? {}, "clash_secret")).toBe(false);
    });
  });

  it("opens the advanced test-target drawer and persists edited targets", async () => {
    const deps = createMockDeps();
    const user = userEvent.setup();
    render(<DashboardApp deps={deps} />);

    await screen.findByTestId("boot-state");
    await user.click(screen.getByRole("button", { name: "Scan" }));
    await user.click(screen.getByRole("button", { name: "Test Targets" }));

    expect(
      screen.getByRole("dialog", { name: "Advanced Test Targets" }),
    ).toBeTruthy();

    expect(screen.queryByLabelText("Target Name")).toBeNull();
    await user.click(screen.getByRole("button", { name: "Edit yt_page" }));

    const nameInput = screen.getByLabelText("Target Name");
    await user.clear(nameInput);
    await user.type(nameInput, "YouTube Landing");
    await user.click(screen.getByRole("button", { name: "Done" }));

    await user.click(screen.getByRole("button", { name: "Route & Clash" }));
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      const saved = deps.getSavedPayload();
      expect(saved).toBeDefined();
      expect(Array.isArray(saved?.test_targets)).toBe(true);
      expect((saved?.test_targets as Array<{ name: string }>)[0]?.name).toBe(
        "YouTube Landing",
      );
    });
  });

  it("locks background scroll while the advanced test-target drawer is open", async () => {
    const deps = createMockDeps();
    const user = userEvent.setup();
    const restoreViewport = mockViewportScrollbarWidth(16);
    try {
      render(<DashboardApp deps={deps} />);

      expect(document.body.style.overflow).toBe("");

      await screen.findByTestId("boot-state");
      await user.click(screen.getByRole("button", { name: "Scan" }));
      await user.click(screen.getByRole("button", { name: "Test Targets" }));

      await waitFor(() => {
        expect(document.body.style.overflow).toBe("hidden");
        expect(document.body.style.overscrollBehavior).toBe("none");
        expect(document.body.style.paddingRight).toBe("16px");
      });

      await user.click(screen.getByRole("button", { name: "Done" }));

      expect(screen.getByRole("dialog", { name: "Advanced Test Targets" })).toBeTruthy();

      await waitFor(() => {
        expect(screen.queryByRole("dialog", { name: "Advanced Test Targets" })).toBeNull();
        expect(document.body.style.overflow).toBe("");
        expect(document.body.style.overscrollBehavior).toBe("");
        expect(document.body.style.paddingRight).toBe("");
      });
    } finally {
      restoreViewport();
    }
  });

  it("renders the advanced test-target drawer in a body-level portal", async () => {
    const deps = createMockDeps();
    const user = userEvent.setup();
    const { container } = render(<DashboardApp deps={deps} />);

    await screen.findByTestId("boot-state");
    await user.click(screen.getByRole("button", { name: "Scan" }));
    await user.click(screen.getByRole("button", { name: "Test Targets" }));

    const dialog = await screen.findByRole("dialog", { name: "Advanced Test Targets" });
    const layer = document.body.querySelector(".drawer-layer");
    const dashboardMain = container.querySelector(".dashboard-main");

    expect(layer).toBeTruthy();
    expect(layer?.parentElement).toBe(document.body);
    expect(dashboardMain?.contains(dialog)).toBe(false);
  });

  it("keeps the drawer mounted until the close animation finishes", async () => {
    const deps = createMockDeps();
    const user = userEvent.setup();
    render(<DashboardApp deps={deps} />);

    await screen.findByTestId("boot-state");
    await user.click(screen.getByRole("button", { name: "Scan" }));
    await user.click(screen.getByRole("button", { name: "Test Targets" }));

    const layer = document.body.querySelector(".drawer-layer");
    expect(layer?.getAttribute("data-state")).toBe("open");

    await user.click(screen.getByRole("button", { name: "Done" }));

    expect(screen.getByRole("dialog", { name: "Advanced Test Targets" })).toBeTruthy();
    await waitFor(() => {
      expect(layer?.getAttribute("data-state")).toBe("closing");
    });

    await waitFor(() => {
      expect(screen.queryByRole("dialog", { name: "Advanced Test Targets" })).toBeNull();
    });
  });

  it("prompts before closing the drawer when test target changes would be discarded", async () => {
    const deps = createMockDeps();
    const user = userEvent.setup();
    render(<DashboardApp deps={deps} />);

    await screen.findByTestId("boot-state");
    await user.click(screen.getByRole("button", { name: "Scan" }));
    await user.click(screen.getByRole("button", { name: "Test Targets" }));
    await user.click(screen.getByRole("switch", { name: "Enable yt_page" }));
    await user.click(screen.getByRole("button", { name: "Done" }));

    expect(screen.getByRole("dialog", { name: "Advanced Test Targets" })).toBeTruthy();
    expect(
      screen.getByRole("alertdialog", { name: "Discard unsaved test target changes?" }),
    ).toBeTruthy();

    await user.click(screen.getByRole("button", { name: "Keep Editing" }));
    expect(screen.queryByRole("alertdialog", { name: "Discard unsaved test target changes?" })).toBeNull();
    expect(screen.getByRole("dialog", { name: "Advanced Test Targets" })).toBeTruthy();
  });

  it("discards unsaved test target changes after confirmation", async () => {
    const deps = createMockDeps();
    const user = userEvent.setup();
    render(<DashboardApp deps={deps} />);

    await screen.findByTestId("boot-state");
    await user.click(screen.getByRole("button", { name: "Scan" }));
    await user.click(screen.getByRole("button", { name: "Test Targets" }));
    await user.click(screen.getByRole("switch", { name: "Enable yt_page" }));
    await user.click(screen.getByRole("button", { name: "Close test target configuration" }));

    expect(
      screen.getByRole("alertdialog", { name: "Discard unsaved test target changes?" }),
    ).toBeTruthy();

    await user.click(screen.getByRole("button", { name: "Discard Changes" }));

    await waitFor(() => {
      expect(screen.queryByRole("dialog", { name: "Advanced Test Targets" })).toBeNull();
    });

    await user.click(screen.getByRole("button", { name: "Route & Clash" }));
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      const saved = deps.getSavedPayload();
      expect(saved).toBeDefined();
      expect((saved?.test_targets as Array<{ name: string; enabled: boolean }>)[0]).toEqual(
        expect.objectContaining({
          name: "yt_page",
          enabled: true,
        }),
      );
    });
  });

  it("renders a stable clash placeholder before target details are available", async () => {
    const deps = createMockDeps();
    const user = userEvent.setup();
    render(<DashboardApp deps={deps} />);

    await screen.findByTestId("boot-state");
    await user.click(screen.getByRole("button", { name: "Route & Clash" }));

    expect(screen.getByTestId("clash-target-placeholder")).toBeTruthy();
    expect(screen.getByText("No Clash target resolved yet.")).toBeTruthy();
  });

  it("supports adding and reordering advanced test targets", async () => {
    const deps = createMockDeps();
    const user = userEvent.setup();
    render(<DashboardApp deps={deps} />);

    await screen.findByTestId("boot-state");
    await user.click(screen.getByRole("button", { name: "Scan" }));
    await user.click(screen.getByRole("button", { name: "Test Targets" }));

    expect(screen.queryByLabelText("Target Name")).toBeNull();
    await user.click(screen.getByRole("button", { name: "Add Target" }));

    const expandedNameInputs = screen.getAllByLabelText("Target Name");
    expect(expandedNameInputs).toHaveLength(1);

    const customInput = expandedNameInputs.at(-1) as HTMLInputElement;
    await user.clear(customInput);
    await user.type(customInput, "Custom Probe");
    expect(screen.queryByRole("button", { name: "Move Up" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Move Down" })).toBeNull();
    expect(screen.getByRole("button", { name: "Reorder yt_page" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Reorder Custom Probe" })).toBeTruthy();
  });

  it("uses compact switches and profile menus inside the advanced target drawer", async () => {
    const deps = createMockDeps();
    const user = userEvent.setup();
    render(<DashboardApp deps={deps} />);

    await screen.findByTestId("boot-state");
    await user.click(screen.getByRole("button", { name: "Scan" }));
    await user.click(screen.getByRole("button", { name: "Test Targets" }));

    expect(screen.queryByText("Target 1")).toBeNull();
    expect(screen.queryByLabelText("Target Name")).toBeNull();

    const enabledSwitch = screen.getByRole("switch", { name: "Enable yt_page" });
    expect(enabledSwitch.getAttribute("aria-checked")).toBe("true");

    await user.click(enabledSwitch);
    expect(enabledSwitch.getAttribute("aria-checked")).toBe("false");

    await user.click(screen.getByRole("button", { name: "Profiles for yt_page" }));
    await user.click(screen.getByRole("menuitemcheckbox", { name: "stress" }));
    await user.click(screen.getByRole("button", { name: "Done" }));

    await user.click(screen.getByRole("button", { name: "Route & Clash" }));
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      const saved = deps.getSavedPayload();
      expect(saved).toBeDefined();
      expect((saved?.test_targets as Array<{ name: string; enabled: boolean }>)[0]?.enabled).toBe(
        false,
      );
      expect(
        (saved?.test_targets as Array<{ name: string; profiles: string[] }>)[0]?.profiles.includes(
          "stress",
        ),
      ).toBe(false);
    });
  });

  it("allows compact rows to remove a target without entering edit mode", async () => {
    const deps = createMockDeps();
    const user = userEvent.setup();
    render(<DashboardApp deps={deps} />);

    await screen.findByTestId("boot-state");
    await user.click(screen.getByRole("button", { name: "Scan" }));
    await user.click(screen.getByRole("button", { name: "Test Targets" }));

    expect(screen.queryByLabelText("Target Name")).toBeNull();
    await user.click(screen.getByRole("button", { name: "Remove yt_204" }));
    expect(screen.queryByRole("button", { name: "Edit yt_204" })).toBeNull();

    await user.click(screen.getByRole("button", { name: "Done" }));
    await user.click(screen.getByRole("button", { name: "Route & Clash" }));
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      const saved = deps.getSavedPayload();
      expect(saved).toBeDefined();
      expect(
        (saved?.test_targets as Array<{ name: string }>).some((target) => target.name === "yt_204"),
      ).toBe(false);
    });
  });

  it("marks changed target rows with a dirty accent bar", async () => {
    const deps = createMockDeps();
    const user = userEvent.setup();
    render(<DashboardApp deps={deps} />);

    await screen.findByTestId("boot-state");
    await user.click(screen.getByRole("button", { name: "Scan" }));
    await user.click(screen.getByRole("button", { name: "Test Targets" }));

    const ytPageEdit = screen.getByRole("button", { name: "Edit yt_page" });
    const ytPageRow = ytPageEdit.closest(".drawer-target-row");
    const yt204Row = screen
      .getByRole("button", { name: "Edit yt_204" })
      .closest(".drawer-target-row");

    expect(ytPageRow?.className.includes("drawer-target-row--dirty")).toBe(false);
    expect(yt204Row?.className.includes("drawer-target-row--dirty")).toBe(false);

    await user.click(screen.getByRole("switch", { name: "Enable yt_page" }));
    expect(ytPageRow?.className.includes("drawer-target-row--dirty")).toBe(true);
    expect(yt204Row?.className.includes("drawer-target-row--dirty")).toBe(false);
  });


  it("keeps the drawer in compact view until a row is focused for editing", async () => {
    const deps = createMockDeps();
    const user = userEvent.setup();
    render(<DashboardApp deps={deps} />);

    await screen.findByTestId("boot-state");
    await user.click(screen.getByRole("button", { name: "Scan" }));
    await user.click(screen.getByRole("button", { name: "Test Targets" }));

    expect(screen.queryByLabelText("Target Name")).toBeNull();
    expect(screen.getByRole("button", { name: "Edit yt_page" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Edit yt_204" })).toBeTruthy();

    await user.click(screen.getByRole("button", { name: "Edit yt_page" }));
    expect(screen.getAllByLabelText("Target Name")).toHaveLength(1);

    await user.click(screen.getByRole("button", { name: "Edit yt_204" }));
    expect(screen.getAllByLabelText("Target Name")).toHaveLength(1);
    expect((screen.getByLabelText("Target Name") as HTMLInputElement).value).toBe("yt_204");
  });
});

function mockSectionRect(
  container: HTMLElement,
  key: "overview" | "route" | "scan" | "logs",
  rect: { top: number; bottom: number },
) {
  const node = container.querySelector(`[data-section="${key}"]`);
  if (!(node instanceof HTMLDivElement)) {
    throw new Error(`section ${key} not found`);
  }
  Object.defineProperty(node, "getBoundingClientRect", {
    configurable: true,
    value: () => ({
      x: 0,
      y: rect.top,
      width: 0,
      height: Math.max(0, rect.bottom - rect.top),
      top: rect.top,
      right: 0,
      bottom: rect.bottom,
      left: 0,
      toJSON: () => ({}),
    }),
  });
}
