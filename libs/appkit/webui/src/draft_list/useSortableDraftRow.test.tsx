import { render, screen } from "@testing-library/react";
import type { DraggableSyntheticListeners } from "@dnd-kit/core";
import { describe, expect, it, vi } from "vitest";

import { useSortableDraftRow } from "./useSortableDraftRow";

const useSortableMock = vi.fn();

vi.mock("@dnd-kit/sortable", () => ({
  useSortable: (...args: unknown[]) => useSortableMock(...args),
}));

vi.mock("@dnd-kit/utilities", () => ({
  CSS: {
    Transform: {
      toString: () => "translate3d(12px, 24px, 0)",
    },
  },
}));

function Harness() {
  const sortableRow = useSortableDraftRow("row-alpha");

  return (
    <div data-testid="sortable-row" ref={sortableRow.setRowRef} style={sortableRow.style}>
      <button
        data-testid="sortable-handle"
        ref={sortableRow.setHandleRef}
        type="button"
        {...sortableRow.dragAttributes}
        {...sortableRow.dragListeners}
      >
        drag
      </button>
      <span data-testid="sortable-dragging">{String(sortableRow.isDragging)}</span>
    </div>
  );
}

describe("useSortableDraftRow", () => {
  it("adapts dnd-kit sortable output into reusable row bindings", () => {
    useSortableMock.mockReturnValue({
      attributes: {
        "data-sortable": "yes",
      },
      listeners: {
        onPointerDown: vi.fn(),
      } satisfies DraggableSyntheticListeners,
      setActivatorNodeRef: () => undefined,
      setNodeRef: () => undefined,
      transform: { x: 12, y: 24, scaleX: 1, scaleY: 1 },
      transition: "transform 200ms ease",
      isDragging: true,
    });

    render(<Harness />);

    expect(useSortableMock).toHaveBeenCalledWith({ id: "row-alpha" });
    expect(screen.getByTestId("sortable-row").getAttribute("style")).toContain(
      "translate3d(12px, 24px, 0)",
    );
    expect(screen.getByTestId("sortable-dragging").textContent).toBe("true");
    expect(screen.getByTestId("sortable-handle").getAttribute("data-sortable")).toBe("yes");
  });
});
