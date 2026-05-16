import type { InterfaceInfo, InterfaceRef } from "../../../lib/api/api/runtime/types";

export function toInterfaceRef(info: InterfaceInfo): InterfaceRef {
  return {
    platform_name: info.platform_name,
    name: info.name,
    index: info.index,
  };
}

export function interfaceKey(info: InterfaceInfo | null | undefined): string {
  if (!info) {
    return "";
  }
  const primary = (info.index ?? "").trim() || info.name.trim();
  if (primary === "") {
    return "";
  }
  return `${info.platform_name.toLowerCase()}|${primary}`;
}

export function mergeInterfaceCandidates(
  primary: InterfaceInfo | null,
  candidates: Array<InterfaceInfo>,
): Array<InterfaceInfo> {
  const merged: Array<InterfaceInfo> = [];
  const seen = new Set<string>();

  const append = (value: InterfaceInfo | null) => {
    if (!value) {
      return;
    }
    const key = interfaceKey(value);
    if (key === "" || seen.has(key)) {
      return;
    }
    seen.add(key);
    merged.push({
      ...value,
      description: value.description?.trim() || value.name,
    });
  };

  append(primary);
  candidates.forEach(append);
  return merged;
}

export function updateCandidateSnapshot(
  candidates: Array<InterfaceInfo>,
  next: InterfaceInfo,
): Array<InterfaceInfo> {
  const targetKey = interfaceKey(next);
  const updated = candidates.map((candidate) =>
    interfaceKey(candidate) === targetKey
      ? {
          ...candidate,
          ...next,
          description: next.description?.trim() || next.name,
        }
      : candidate,
  );
  return mergeInterfaceCandidates(next, updated);
}

export function findCandidateByKey(
  candidates: Array<InterfaceInfo>,
  key: string,
): InterfaceInfo | null {
  return candidates.find((candidate) => interfaceKey(candidate) === key) ?? null;
}

export function splitDetectSelectionWarning(message: string): {
  globalWarning: string;
  interfaceHint: string;
} {
  const raw = message.trim();
  if (raw === "") {
    return {
      globalWarning: "",
      interfaceHint: "",
    };
  }

  const interfaceHintPattern =
    /Current route resolves to virtual interface .*?\.\s*Selected underlying interface .*? instead\./;
  const interfaceHint = raw.match(interfaceHintPattern)?.[0]?.trim() ?? "";
  if (interfaceHint === "") {
    return {
      globalWarning: raw,
      interfaceHint: "",
    };
  }

  return {
    globalWarning: raw.replace(interfaceHint, "").replace(/\s{2,}/g, " ").trim(),
    interfaceHint,
  };
}
