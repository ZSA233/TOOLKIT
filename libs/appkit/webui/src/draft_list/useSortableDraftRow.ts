import type { DraggableAttributes, DraggableSyntheticListeners } from "@dnd-kit/core";
import { useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import type { CSSProperties } from "react";

export interface SortableDraftRowBindings {
  dragAttributes: DraggableAttributes;
  dragListeners?: DraggableSyntheticListeners;
  isDragging: boolean;
  setHandleRef(node: HTMLButtonElement | null): void;
  setRowRef(node: HTMLDivElement | null): void;
  style: CSSProperties;
}

export function useSortableDraftRow(dragId: string): SortableDraftRowBindings {
  const {
    attributes,
    listeners,
    setActivatorNodeRef,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({
    id: dragId,
  });

  return {
    dragAttributes: attributes,
    dragListeners: listeners,
    isDragging,
    setHandleRef: setActivatorNodeRef as (node: HTMLButtonElement | null) => void,
    setRowRef: setNodeRef as (node: HTMLDivElement | null) => void,
    style: {
      transform: CSS.Transform.toString(transform),
      transition,
    },
  };
}
