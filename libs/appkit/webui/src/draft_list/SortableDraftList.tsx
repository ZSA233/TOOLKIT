import {
  closestCenter,
  DndContext,
  DragOverlay,
  KeyboardSensor,
  PointerSensor,
  TouchSensor,
  type DragEndEvent,
  type DragStartEvent,
  useSensor,
  useSensors,
} from "@dnd-kit/core";
import {
  SortableContext,
  sortableKeyboardCoordinates,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { Fragment, useState, type ReactNode } from "react";
import { createPortal } from "react-dom";

import type {
  DraftListChangeReason,
  DraftListRenderOverlayInput,
  DraftListRenderRowInput,
  DraftListRow,
} from "./draft_list_types";

function dragOverlayStyle(width: number | null): { width: string } | undefined {
  if (typeof width !== "number" || !Number.isFinite(width) || width <= 0) {
    return undefined;
  }

  return { width: `${width}px` };
}

export function SortableDraftList<TItem extends { order: number }>(props: {
  rows: DraftListRow<TItem>[];
  rowIds: string[];
  rowChangesById: Map<string, Set<DraftListChangeReason>>;
  expandedRowId: string | null;
  onReorder(activeRowId: string, overRowId: string): void;
  renderRow(input: DraftListRenderRowInput<TItem>): ReactNode;
  renderOverlay(input: DraftListRenderOverlayInput<TItem>): ReactNode;
}) {
  const [draggingRowId, setDraggingRowId] = useState<string | null>(null);
  const [dragOverlayWidth, setDragOverlayWidth] = useState<number | null>(null);
  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(TouchSensor, {
      activationConstraint: {
        delay: 120,
        tolerance: 6,
      },
    }),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    }),
  );
  const activeIndex = draggingRowId == null
    ? -1
    : props.rows.findIndex((row) => row.rowId === draggingRowId);
  const activeRow = activeIndex >= 0 ? props.rows[activeIndex] ?? null : null;

  const handleDragStart = (event: DragStartEvent) => {
    setDraggingRowId(String(event.active.id));
    setDragOverlayWidth(event.active.rect.current.initial?.width ?? null);
  };

  const handleDragEnd = (event: DragEndEvent) => {
    setDraggingRowId(null);
    setDragOverlayWidth(null);
    if (!event.over) {
      return;
    }

    const activeRowId = String(event.active.id);
    const overRowId = String(event.over.id);
    if (activeRowId === overRowId) {
      return;
    }

    props.onReorder(activeRowId, overRowId);
  };

  const overlay = (
    <DragOverlay adjustScale={false}>
      {activeRow ? (
        <div className="drawer-target-row__overlay-shell" style={dragOverlayStyle(dragOverlayWidth)}>
          {props.renderOverlay({
            row: activeRow,
            index: activeIndex,
          })}
        </div>
      ) : null}
    </DragOverlay>
  );

  return (
    <DndContext
      collisionDetection={closestCenter}
      onDragCancel={() => {
        setDraggingRowId(null);
        setDragOverlayWidth(null);
      }}
      onDragEnd={handleDragEnd}
      onDragStart={handleDragStart}
      sensors={sensors}
    >
      <SortableContext items={props.rowIds} strategy={verticalListSortingStrategy}>
        <div className="drawer-target-list">
          {props.rows.map((row, index) =>
            (
              <Fragment key={row.rowId}>
                {props.renderRow({
                  row,
                  index,
                  changed: props.rowChangesById.has(row.rowId),
                  changeReasons: props.rowChangesById.get(row.rowId) ?? new Set(),
                  expanded: props.expandedRowId === row.rowId,
                })}
              </Fragment>
            ))}
        </div>
      </SortableContext>
      {typeof document === "undefined" ? overlay : createPortal(overlay, document.body)}
    </DndContext>
  );
}
