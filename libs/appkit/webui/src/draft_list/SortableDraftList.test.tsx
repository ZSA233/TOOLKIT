import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import type {
  DraftListChangeReason,
  DraftListRenderOverlayInput,
  DraftListRenderRowInput,
  DraftListRow,
} from "./draft_list_types";
import { SortableDraftList } from "./SortableDraftList";

vi.mock("@dnd-kit/core", () => ({
  closestCenter: vi.fn(),
  DndContext: (props: {
    children: React.ReactNode;
    onDragStart?(event: unknown): void;
  }) => (
    <div>
      <button
        onClick={() =>
          props.onDragStart?.({
            active: {
              id: "row-alpha",
              rect: {
                current: {
                  initial: {
                    width: 320,
                  },
                },
              },
            },
          })}
        type="button"
      >
        Start Drag
      </button>
      {props.children}
    </div>
  ),
  DragOverlay: (props: { children: React.ReactNode }) => (
    <div data-testid="drag-overlay">{props.children}</div>
  ),
  KeyboardSensor: class KeyboardSensor {},
  PointerSensor: class PointerSensor {},
  TouchSensor: class TouchSensor {},
  useSensor: vi.fn(() => ({})),
  useSensors: vi.fn(() => []),
}));

vi.mock("@dnd-kit/sortable", () => ({
  SortableContext: (props: { children: React.ReactNode }) => <>{props.children}</>,
  sortableKeyboardCoordinates: vi.fn(),
  verticalListSortingStrategy: {},
}));

afterEach(() => {
  document.body.innerHTML = "";
});

describe("SortableDraftList", () => {
  it("renders the drag overlay through a body-level portal instead of inside the drawer sheet", async () => {
    const user = userEvent.setup();
    const rows: DraftListRow<{ name: string; order: number }>[] = [
      {
        rowId: "row-alpha",
        item: {
          name: "alpha",
          order: 10,
        },
      },
    ];

    render(
      <div data-testid="drawer-sheet">
        <SortableDraftList
          expandedRowId={null}
          onReorder={() => undefined}
          renderOverlay={({ row }: DraftListRenderOverlayInput<{ name: string; order: number }>) => (
            <div>{row.item.name}</div>
          )}
          renderRow={({ row }: DraftListRenderRowInput<{ name: string; order: number }>) => (
            <div>{row.item.name}</div>
          )}
          rowChangesById={new Map<string, Set<DraftListChangeReason>>()}
          rowIds={rows.map((row) => row.rowId)}
          rows={rows}
        />
      </div>,
    );

    await user.click(screen.getByRole("button", { name: "Start Drag" }));

    const drawerSheet = screen.getByTestId("drawer-sheet");
    expect(within(drawerSheet).queryByTestId("drag-overlay")).toBeNull();
    expect(within(document.body).getByTestId("drag-overlay")).toBeTruthy();
  });
});
