import type { DraftListChangeReason, DraftListRow } from "./draft_list_types";

export function computeDraftRowChanges<TItem>(
  baselineRows: DraftListRow<TItem>[],
  draftRows: DraftListRow<TItem>[],
  itemsEqual: (left: TItem, right: TItem) => boolean,
): Map<string, Set<DraftListChangeReason>> {
  const changesById = new Map<string, Set<DraftListChangeReason>>();
  const baselineRowsById = new Map(baselineRows.map((row) => [row.rowId, row]));
  const baselineIndexById = new Map(baselineRows.map((row, index) => [row.rowId, index]));

  draftRows.forEach((row, index) => {
    const rowChanges = new Set<DraftListChangeReason>();
    const baselineRow = baselineRowsById.get(row.rowId);
    if (!baselineRow) {
      rowChanges.add("new");
    } else {
      if (!itemsEqual(baselineRow.item, row.item)) {
        rowChanges.add("value");
      }
      const baselineIndex = baselineIndexById.get(row.rowId);
      if (typeof baselineIndex === "number" && baselineIndex !== index) {
        rowChanges.add("position");
      }
    }

    if (rowChanges.size > 0) {
      changesById.set(row.rowId, rowChanges);
    }
  });

  return changesById;
}
