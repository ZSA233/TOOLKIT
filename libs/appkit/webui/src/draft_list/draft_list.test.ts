import { describe, expect, it } from "vitest";

import type { DraftListRow } from "./draft_list_types";
import { computeDraftRowChanges } from "./dirty_draft_rows";
import { reorderDraftRows } from "./reorder_draft_rows";

type ExampleItem = {
  name: string;
  order: number;
};

function createRows(items: ExampleItem[]): DraftListRow<ExampleItem>[] {
  return items.map((item) => ({
    rowId: item.name,
    item: {
      ...item,
    },
  }));
}

function equalItems(left: ExampleItem, right: ExampleItem): boolean {
  return left.name === right.name && left.order === right.order;
}

describe("reorderDraftRows", () => {
  it("marks both affected rows dirty when a row changes visible position", () => {
    const baselineRows = createRows([
      { name: "google_page", order: 40 },
      { name: "google_204", order: 50 },
    ]);

    const reorderedRows = reorderDraftRows(baselineRows, "google_page", "google_204", {
      cloneItem: (item) => ({ ...item }),
      assignMovedOrder: ({ previousOrder, nextOrder }) => {
        if (previousOrder == null) {
          return nextOrder == null ? 10 : Math.floor(nextOrder / 2);
        }
        if (nextOrder == null) {
          return previousOrder + 10;
        }
        if (nextOrder - previousOrder <= 1) {
          return null;
        }
        return previousOrder + Math.floor((nextOrder - previousOrder) / 2);
      },
      resequenceOrder: (index) => (index + 1) * 10,
    });

    const changes = computeDraftRowChanges(baselineRows, reorderedRows, equalItems);

    expect(changes.get("google_page")).toEqual(new Set(["value", "position"]));
    expect(changes.get("google_204")).toEqual(new Set(["position"]));
  });

  it("keeps unaffected sparse trailing rows clean when insertion gap exists", () => {
    const baselineRows = createRows([
      { name: "google_page", order: 40 },
      { name: "google_204", order: 50 },
      { name: "gstatic_204", order: 60 },
      { name: "google_logo", order: 65 },
      { name: "connectivity_204", order: 70 },
    ]);

    const reorderedRows = reorderDraftRows(baselineRows, "google_page", "google_204", {
      cloneItem: (item) => ({ ...item }),
      assignMovedOrder: ({ previousOrder, nextOrder }) => {
        if (previousOrder == null) {
          return nextOrder == null ? 10 : Math.floor(nextOrder / 2);
        }
        if (nextOrder == null) {
          return previousOrder + 10;
        }
        if (nextOrder - previousOrder <= 1) {
          return null;
        }
        return previousOrder + Math.floor((nextOrder - previousOrder) / 2);
      },
      resequenceOrder: (index) => (index + 1) * 10,
    });

    const changes = computeDraftRowChanges(baselineRows, reorderedRows, equalItems);

    expect(reorderedRows.map((row) => row.item.order)).toEqual([50, 55, 60, 65, 70]);
    expect(changes.get("google_204")).toEqual(new Set(["position"]));
    expect(changes.get("google_page")).toEqual(new Set(["value", "position"]));
    expect(changes.get("google_logo"))?.toBeUndefined();
    expect(changes.get("connectivity_204"))?.toBeUndefined();
  });
});
