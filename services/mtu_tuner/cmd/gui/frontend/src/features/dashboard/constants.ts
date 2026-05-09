import type { SavedSettings, TaskState } from "../../lib/api/api/runtime/models";

import type {
  SessionSettings,
  TaskLogEvent,
  TaskProgressEvent,
  TestTargetSettings,
} from "./types";

export const QUICK_MTU_VALUES = [1500, 1460, 1440, 1420, 1400, 1380, 1360];
export const APP_DISPLAY_NAME = "mtu-tuner";
export const TEST_PROFILES = ["browser", "stress", "quick", "chrome"] as const;
export const DEFAULT_TEST_TARGETS: TestTargetSettings[] = [
  {
    name: "yt_page",
    url: "https://www.youtube.com/",
    enabled: true,
    profiles: ["browser", "stress", "quick"],
    order: 10,
  },
  {
    name: "yt_204",
    url: "https://www.youtube.com/generate_204",
    enabled: true,
    profiles: ["browser", "stress", "chrome"],
    order: 20,
  },
  {
    name: "yt_thumb",
    url: "https://i.ytimg.com/vi/dQw4w9WgXcQ/hqdefault.jpg",
    enabled: true,
    profiles: ["browser", "stress", "chrome"],
    order: 30,
  },
  {
    name: "google_page",
    url: "https://www.google.com/",
    enabled: true,
    profiles: ["browser", "stress", "quick"],
    order: 40,
  },
  {
    name: "google_204",
    url: "https://www.google.com/generate_204",
    enabled: true,
    profiles: ["browser", "stress", "chrome"],
    order: 50,
  },
  {
    name: "gstatic_204",
    url: "https://www.gstatic.com/generate_204",
    enabled: true,
    profiles: ["browser", "stress", "chrome"],
    order: 60,
  },
  {
    name: "google_logo",
    url: "https://www.google.com/images/branding/googlelogo/2x/googlelogo_color_272x92dp.png",
    enabled: true,
    profiles: ["chrome"],
    order: 65,
  },
  {
    name: "connectivity_204",
    url: "https://connectivitycheck.gstatic.com/generate_204",
    enabled: true,
    profiles: ["browser", "stress"],
    order: 70,
  },
  {
    name: "yt_page_2",
    url: "https://www.youtube.com/results?search_query=mtu",
    enabled: true,
    profiles: ["stress"],
    order: 80,
  },
  {
    name: "google_search",
    url: "https://www.google.com/search?q=mtu",
    enabled: true,
    profiles: ["stress"],
    order: 90,
  },
];

export const DEFAULT_SAVED_SETTINGS: SavedSettings = {
  version: 2,
  route_probe: "auto",
  fallback_probe: "1.1.1.1",
  http_proxy: "http://127.0.0.1:7890",
  clash_api: "http://127.0.0.1:9097",
  proxy_group: "auto",
  config_path: "",
  browser_path: "",
  test_profile: "chrome",
  sweep_mtus: "1500,1480,1460,1440,1420,1400,1380,1360",
  target_mtu: 1400,
  test_targets: DEFAULT_TEST_TARGETS,
};

export const DEFAULT_SESSION_SETTINGS: SessionSettings = {
  ...DEFAULT_SAVED_SETTINGS,
  clash_secret: "",
  rounds: "1",
  concurrency: "1",
  test_targets: DEFAULT_TEST_TARGETS,
};

export const DEFAULT_TASK_STATE: TaskState = {
  kind: "",
  status: "idle",
  cancel_requested: false,
};

export const DEFAULT_TASK_PROGRESS: TaskProgressEvent = {
  kind: "",
  done: 0,
  total: 1,
  label: "Idle",
};

export function createBootLog(line: string): TaskLogEvent {
  return {
    kind: "ui",
    line,
    ts: new Date().toISOString(),
  };
}
