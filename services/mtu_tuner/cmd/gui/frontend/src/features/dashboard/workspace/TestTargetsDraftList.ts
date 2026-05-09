import type { TestTargetProfile, TestTargetSettings } from "../types";
import { TEST_PROFILES } from "../constants";

export function cloneTarget(target: TestTargetSettings): TestTargetSettings {
  return {
    ...target,
    profiles: [...target.profiles],
  };
}

export function cloneTargets(targets: TestTargetSettings[]): TestTargetSettings[] {
  return targets.map(cloneTarget).sort((left, right) => left.order - right.order);
}

export function targetSettingsEqual(
  left: TestTargetSettings,
  right: TestTargetSettings,
): boolean {
  return (
    left.name === right.name &&
    left.url === right.url &&
    left.enabled === right.enabled &&
    left.order === right.order &&
    left.profiles.length === right.profiles.length &&
    left.profiles.every((profile, index) => profile === right.profiles[index])
  );
}

export function createTargetDraft(targets: TestTargetSettings[]): TestTargetSettings {
  const nextOrder = Math.max(0, ...targets.map((target) => target.order)) + 10;
  return {
    name: "",
    url: "https://",
    enabled: true,
    profiles: ["browser"],
    order: nextOrder,
  };
}

export function toggleProfile(
  target: TestTargetSettings,
  profile: TestTargetProfile,
): TestTargetProfile[] {
  if (target.profiles.includes(profile)) {
    const nextProfiles = target.profiles.filter((item) => item !== profile);
    return nextProfiles.length > 0 ? nextProfiles : [profile];
  }

  return [...target.profiles, profile];
}

export function summarizeProfiles(profiles: TestTargetProfile[]): string {
  const orderedProfiles = TEST_PROFILES.filter((profile) => profiles.includes(profile));
  if (orderedProfiles.length === 0) {
    return "Profiles";
  }
  if (orderedProfiles.length === 1) {
    return orderedProfiles[0];
  }
  return `${orderedProfiles[0]} +${orderedProfiles.length - 1}`;
}

export function targetDisplayLabel(target: TestTargetSettings, index: number): string {
  const trimmedName = target.name.trim();
  if (trimmedName !== "") {
    return trimmedName;
  }

  const trimmedURL = target.url.trim();
  if (trimmedURL !== "") {
    return trimmedURL;
  }

  return `target ${index + 1}`;
}

export function assignMovedTargetOrder(input: {
  previousOrder: number | null;
  nextOrder: number | null;
}): number | null {
  return computeInsertedOrder(input.previousOrder, input.nextOrder);
}

function computeInsertedOrder(
  previousOrder: number | null,
  nextOrder: number | null,
): number | null {
  if (previousOrder == null && nextOrder == null) {
    return 10;
  }

  if (previousOrder == null) {
    const candidate = Math.floor(nextOrder! / 2);
    return candidate > 0 ? candidate : null;
  }

  if (nextOrder == null) {
    return previousOrder + 10;
  }

  if (nextOrder - previousOrder <= 1) {
    return null;
  }

  return previousOrder + Math.floor((nextOrder - previousOrder) / 2);
}
