import { useEffect, useMemo, useRef, useState } from "react";

import { computeDraftRowChanges } from "./dirty_draft_rows";
import { reorderDraftRows } from "./reorder_draft_rows";
import type { DraftListRow, DraftListSession } from "./draft_list_types";

export interface UseDraftListSessionOptions<TItem extends { order: number }> {
  open: boolean;
  items: TItem[];
  cloneItem(item: TItem): TItem;
  itemsEqual(left: TItem, right: TItem): boolean;
  createNewItem(items: TItem[]): TItem;
  assignMovedOrder(input: {
    previousOrder: number | null;
    nextOrder: number | null;
    movedItem: TItem;
  }): number | null;
  resequenceOrder(index: number, item: TItem): number;
  onDraftChange?(items: TItem[]): void;
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

function toSortedRows<TItem extends { order: number }>(
  items: TItem[],
  cloneItem: (item: TItem) => TItem,
  createRowId: () => string,
): DraftListRow<TItem>[] {
  return items
    .map((item) => cloneItem(item))
    .sort((left, right) => left.order - right.order)
    .map((item) => ({
      rowId: createRowId(),
      item,
    }));
}

function cloneItems<TItem extends { order: number }>(
  rows: DraftListRow<TItem>[],
  cloneItem: (item: TItem) => TItem,
): TItem[] {
  return rows.map((row) => cloneItem(row.item));
}

function hasRemovedRows<TItem>(
  baselineRows: DraftListRow<TItem>[],
  draftRows: DraftListRow<TItem>[],
): boolean {
  const draftIds = new Set(draftRows.map((row) => row.rowId));
  return baselineRows.some((row) => !draftIds.has(row.rowId));
}

export function useDraftListSession<TItem extends { order: number }>(
  options: UseDraftListSessionOptions<TItem>,
): DraftListSession<TItem> {
  const rowIdSeedRef = useRef(0);
  const previousOpenRef = useRef(options.open);
  const baselineRowsRef = useRef<DraftListRow<TItem>[]>([]);
  const createRowId = () => {
    rowIdSeedRef.current += 1;
    return `draft-row-${rowIdSeedRef.current}`;
  };
  const [rows, setRows] = useState<DraftListRow<TItem>[]>(() => {
    const initialRows = toSortedRows(options.items, options.cloneItem, createRowId);
    baselineRowsRef.current = cloneRows(initialRows, options.cloneItem);
    return initialRows;
  });
  const [expandedRowId, setExpandedRowId] = useState<string | null>(null);
  const [baselineRevision, setBaselineRevision] = useState(0);

  const commitRows = (nextRows: DraftListRow<TItem>[], nextExpandedRowId = expandedRowId) => {
    setRows(nextRows);
    setExpandedRowId(nextExpandedRowId);
    options.onDraftChange?.(cloneItems(nextRows, options.cloneItem));
  };

  useEffect(() => {
    if (options.open && !previousOpenRef.current) {
      const nextRows = toSortedRows(options.items, options.cloneItem, createRowId);
      baselineRowsRef.current = cloneRows(nextRows, options.cloneItem);
      setRows(nextRows);
      setExpandedRowId(null);
    }

    if (!options.open && previousOpenRef.current) {
      setExpandedRowId(null);
    }

    previousOpenRef.current = options.open;
  }, [options.open, options.items, options.cloneItem]);

  const rowChangesById = useMemo(
    () => computeDraftRowChanges(baselineRowsRef.current, rows, options.itemsEqual),
    [baselineRevision, rows, options.itemsEqual],
  );
  const hasDirtyChanges = rowChangesById.size > 0 || hasRemovedRows(baselineRowsRef.current, rows);

  return {
    rows,
    rowIds: rows.map((row: DraftListRow<TItem>) => row.rowId),
    expandedRowId,
    rowChangesById,
    hasDirtyChanges,
    appendNewItem() {
      const nextItem = options.createNewItem(cloneItems(rows, options.cloneItem));
      const nextRow = {
        rowId: createRowId(),
        item: options.cloneItem(nextItem),
      };
      commitRows([...rows, nextRow], nextRow.rowId);
    },
    updateRow(rowId, updater) {
      commitRows(
        rows.map((row: DraftListRow<TItem>) =>
          row.rowId === rowId
            ? {
                ...row,
                item: options.cloneItem(updater(options.cloneItem(row.item))),
              }
            : {
                ...row,
                item: options.cloneItem(row.item),
              },
        ),
      );
    },
    removeRow(rowId) {
      commitRows(
        rows
          .filter((row: DraftListRow<TItem>) => row.rowId !== rowId)
          .map((row: DraftListRow<TItem>) => ({
            ...row,
            item: options.cloneItem(row.item),
          })),
        expandedRowId === rowId ? null : expandedRowId,
      );
    },
    reorderRows(activeRowId, overRowId) {
      commitRows(
        reorderDraftRows(rows, activeRowId, overRowId, {
          cloneItem: options.cloneItem,
          assignMovedOrder: options.assignMovedOrder,
          resequenceOrder: options.resequenceOrder,
        }),
      );
    },
    resetToBaseline() {
      const restoredRows = cloneRows(baselineRowsRef.current, options.cloneItem);
      commitRows(restoredRows, null);
    },
    promoteDraftToBaseline() {
      baselineRowsRef.current = cloneRows(rows, options.cloneItem);
      setBaselineRevision((current) => current + 1);
    },
    setExpandedRowId,
  };
}
