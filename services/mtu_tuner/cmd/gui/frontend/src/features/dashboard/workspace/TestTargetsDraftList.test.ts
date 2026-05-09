import { computeDraftRowChanges, reorderDraftRows, type DraftListRow } from "@toolkit/appkit-webui";
import { describe, expect, it } from "vitest";

import type { TestTargetSettings } from "../types";
import {
  assignMovedTargetOrder,
  cloneTarget,
  createTargetDraft,
  targetSettingsEqual,
} from "./TestTargetsDraftList";

function createRows(targets: TestTargetSettings[]): DraftListRow<TestTargetSettings>[] {
  return targets.map((target) => ({
    rowId: target.name,
    item: cloneTarget(target),
  }));
}

describe("TestTargetsDraftList", () => {
  it("creates new targets after the current maximum order", () => {
    const draft = createTargetDraft([
      {
        name: "alpha",
        url: "https://alpha.example",
        enabled: true,
        profiles: ["browser"],
        order: 10,
      },
      {
        name: "beta",
        url: "https://beta.example",
        enabled: true,
        profiles: ["browser"],
        order: 30,
      },
    ]);

    expect(draft.order).toBe(40);
    expect(draft.url).toBe("https://");
  });

  it("keeps sparse trailing orders stable when a target moves down one slot", () => {
    const baselineRows = createRows([
      {
        name: "google_page",
        url: "https://www.google.com/",
        enabled: true,
        profiles: ["browser"],
        order: 40,
      },
      {
        name: "google_204",
        url: "https://www.google.com/generate_204",
        enabled: true,
        profiles: ["browser"],
        order: 50,
      },
      {
        name: "gstatic_204",
        url: "https://www.gstatic.com/generate_204",
        enabled: true,
        profiles: ["browser"],
        order: 60,
      },
      {
        name: "google_logo",
        url: "https://www.google.com/images/branding/googlelogo.png",
        enabled: true,
        profiles: ["chrome"],
        order: 65,
      },
      {
        name: "connectivity_204",
        url: "https://connectivitycheck.gstatic.com/generate_204",
        enabled: true,
        profiles: ["browser"],
        order: 70,
      },
    ]);

    const reorderedRows = reorderDraftRows(baselineRows, "google_page", "google_204", {
      cloneItem: cloneTarget,
      assignMovedOrder: ({ previousOrder, nextOrder }) =>
        assignMovedTargetOrder({ previousOrder, nextOrder }),
      resequenceOrder: (index) => (index + 1) * 10,
    });

    expect(reorderedRows.map((row) => row.item.name)).toEqual([
      "google_204",
      "google_page",
      "gstatic_204",
      "google_logo",
      "connectivity_204",
    ]);
    expect(reorderedRows.map((row) => row.item.order)).toEqual([50, 55, 60, 65, 70]);
  });

  it("marks both moved rows dirty when visible order changes", () => {
    const baselineRows = createRows([
      {
        name: "google_page",
        url: "https://www.google.com/",
        enabled: true,
        profiles: ["browser"],
        order: 40,
      },
      {
        name: "google_204",
        url: "https://www.google.com/generate_204",
        enabled: true,
        profiles: ["browser"],
        order: 50,
      },
      {
        name: "google_logo",
        url: "https://www.google.com/images/branding/googlelogo.png",
        enabled: true,
        profiles: ["chrome"],
        order: 65,
      },
    ]);

    const reorderedRows = reorderDraftRows(baselineRows, "google_page", "google_204", {
      cloneItem: cloneTarget,
      assignMovedOrder: ({ previousOrder, nextOrder }) =>
        assignMovedTargetOrder({ previousOrder, nextOrder }),
      resequenceOrder: (index) => (index + 1) * 10,
    });

    const changes = computeDraftRowChanges(baselineRows, reorderedRows, targetSettingsEqual);

    expect(changes.get("google_page")).toEqual(new Set(["value", "position"]));
    expect(changes.get("google_204")).toEqual(new Set(["position"]));
    expect(changes.get("google_logo")).toBeUndefined();
  });
});
