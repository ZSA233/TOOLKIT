export type DraftListChangeReason = "new" | "value" | "position";

export interface DraftListRow<TItem> {
  rowId: string;
  item: TItem;
}

export interface DraftListRenderRowInput<TItem> {
  row: DraftListRow<TItem>;
  index: number;
  changed: boolean;
  changeReasons: Set<DraftListChangeReason>;
  expanded: boolean;
}

export interface DraftListRenderOverlayInput<TItem> {
  row: DraftListRow<TItem>;
  index: number;
}

export interface DraftListSession<TItem extends { order: number }> {
  rows: DraftListRow<TItem>[];
  rowIds: string[];
  expandedRowId: string | null;
  rowChangesById: Map<string, Set<DraftListChangeReason>>;
  hasDirtyChanges: boolean;
  appendNewItem(): void;
  updateRow(rowId: string, updater: (item: TItem) => TItem): void;
  removeRow(rowId: string): void;
  reorderRows(activeRowId: string, overRowId: string): void;
  resetToBaseline(): void;
  promoteDraftToBaseline(): void;
  setExpandedRowId(rowId: string | null): void;
}
