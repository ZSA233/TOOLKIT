import type { SavedSettings } from "../../../lib/api/api/runtime/models";
import { DEFAULT_SESSION_SETTINGS } from "../constants";
import type { SessionSettings, TestTargetSettings } from "../types";

function cloneTestTargets(targets: TestTargetSettings[]): TestTargetSettings[] {
  return targets
    .map((target) => ({
      ...target,
      profiles: [...target.profiles],
    }))
    .sort((left, right) => left.order - right.order);
}

export function toDashboardSessionSettings(
  settings: Partial<SessionSettings> | SavedSettings,
): SessionSettings {
  const sessionFields = settings as Partial<SessionSettings>;
  return {
    ...DEFAULT_SESSION_SETTINGS,
    ...settings,
    clash_secret: sessionFields.clash_secret ?? "",
    rounds: sessionFields.rounds ?? "",
    concurrency: sessionFields.concurrency ?? "",
    test_targets: cloneTestTargets(
      sessionFields.test_targets ?? DEFAULT_SESSION_SETTINGS.test_targets,
    ),
  };
}

export function toPersistedSavedSettings(settings: SessionSettings): SavedSettings {
  return {
    version: settings.version,
    route_probe: settings.route_probe,
    fallback_probe: settings.fallback_probe,
    http_proxy: settings.http_proxy,
    clash_api: settings.clash_api,
    proxy_group: settings.proxy_group,
    config_path: settings.config_path,
    browser_path: settings.browser_path,
    test_profile: settings.test_profile,
    sweep_mtus: settings.sweep_mtus,
    target_mtu: settings.target_mtu,
    test_targets: cloneTestTargets(settings.test_targets),
  };
}

export function parseOptionalPositiveInt(value: string): number | undefined {
  const parsed = Number.parseInt(value.trim(), 10);
  if (!Number.isFinite(parsed) || parsed <= 0) {
    return undefined;
  }
  return parsed;
}

export function trimOptional(value: string): string | undefined {
  const trimmed = value.trim();
  return trimmed === "" ? undefined : trimmed;
}
