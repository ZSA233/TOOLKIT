import type { DraftListRow } from "./draft_list_types";

export interface ReorderDraftRowsOptions<TItem extends { order: number }> {
  cloneItem(item: TItem): TItem;
  assignMovedOrder(input: {
    previousOrder: number | null;
    nextOrder: number | null;
    movedItem: TItem;
  }): number | null;
  resequenceOrder(index: number, item: TItem): number;
}

function cloneRows<TItem extends { order: number }>(
  rows: DraftListRow<TItem>[],
  cloneItem: (item: TItem) => TItem,
): DraftListRow<TItem>[] {
  return rows.map((row) => ({
    ...row,
    item: cloneItem(row.item),
  }));
}

export function reorderDraftRows<TItem extends { order: number }>(
  rows: DraftListRow<TItem>[],
  activeRowId: string,
  overRowId: string,
  options: ReorderDraftRowsOptions<TItem>,
): DraftListRow<TItem>[] {
  const nextRows = cloneRows(rows, options.cloneItem);
  if (activeRowId === overRowId) {
    return nextRows;
  }

  const activeIndex = nextRows.findIndex((row) => row.rowId === activeRowId);
  const overIndex = nextRows.findIndex((row) => row.rowId === overRowId);
  if (activeIndex < 0 || overIndex < 0) {
    return nextRows;
  }

  const [activeRow] = nextRows.splice(activeIndex, 1);
  if (!activeRow) {
    return nextRows;
  }
  nextRows.splice(overIndex, 0, activeRow);

  const movedRow = nextRows[overIndex];
  if (!movedRow) {
    return nextRows;
  }

  const nextOrder = options.assignMovedOrder({
    previousOrder: nextRows[overIndex - 1]?.item.order ?? null,
    nextOrder: nextRows[overIndex + 1]?.item.order ?? null,
    movedItem: movedRow.item,
  });

  if (nextOrder == null) {
    return nextRows.map((row, index) => ({
      ...row,
      item: {
        ...options.cloneItem(row.item),
        order: options.resequenceOrder(index, row.item),
      },
    }));
  }

  movedRow.item.order = nextOrder;
  return nextRows;
}
