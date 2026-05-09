import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import type { DraftListRow, DraftListSession } from "./draft_list_types";
import { DraftListEditorDrawer } from "./DraftListEditorDrawer";

type ExampleItem = {
  name: string;
  order: number;
};

function createSession(overrides: Partial<DraftListSession<ExampleItem>> = {}): DraftListSession<ExampleItem> {
  const rows: DraftListRow<ExampleItem>[] = [
    {
      rowId: "alpha",
      item: {
        name: "alpha",
        order: 10,
      },
    },
  ];

  return {
    rows,
    rowIds: rows.map((row) => row.rowId),
    expandedRowId: null,
    rowChangesById: new Map(),
    hasDirtyChanges: false,
    appendNewItem: vi.fn(),
    updateRow: vi.fn(),
    removeRow: vi.fn(),
    reorderRows: vi.fn(),
    resetToBaseline: vi.fn(),
    promoteDraftToBaseline: vi.fn(),
    setExpandedRowId: vi.fn(),
    ...overrides,
  };
}

describe("DraftListEditorDrawer", () => {
  it("opens discard confirmation instead of closing immediately when dirty changes exist", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    const session = createSession({
      hasDirtyChanges: true,
      rowChangesById: new Map([["alpha", new Set(["value"])]]),
    });

    render(
      <DraftListEditorDrawer
        addLabel="Add Target"
        description="Shared drawer description"
        doneLabel="Done"
        onClose={onClose}
        onRequestAdd={() => session.appendNewItem()}
        open
        renderOverlay={({ row }) => <div>{row.item.name}</div>}
        renderRow={({ row }) => <div>{row.item.name}</div>}
        session={session}
        title="Targets"
      />,
    );

    await user.click(screen.getByRole("button", { name: "Done" }));

    expect(onClose).not.toHaveBeenCalled();
    expect(screen.getByRole("alertdialog", { name: "Discard unsaved changes?" })).toBeTruthy();
  });

  it("renders an explicit save action when the drawer provides one", async () => {
    const user = userEvent.setup();
    const onSave = vi.fn(async () => undefined);
    const session = createSession({
      hasDirtyChanges: true,
      rowChangesById: new Map([["alpha", new Set(["value"])]]),
    });

    render(
      <DraftListEditorDrawer
        addLabel="Add Target"
        doneLabel="Done"
        onClose={vi.fn()}
        onRequestAdd={() => session.appendNewItem()}
        onSave={onSave}
        open
        renderOverlay={({ row }) => <div>{row.item.name}</div>}
        renderRow={({ row }) => <div>{row.item.name}</div>}
        saveLabel="Save"
        session={session}
        title="Targets"
      />,
    );

    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(onSave).toHaveBeenCalledTimes(1);
  });
});
