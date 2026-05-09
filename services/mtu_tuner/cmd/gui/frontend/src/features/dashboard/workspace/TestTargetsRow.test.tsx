import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { TestTargetsRow } from "./TestTargetsRow";

const useSortableMock = vi.fn();

vi.mock("@dnd-kit/sortable", () => ({
  useSortable: (...args: unknown[]) => useSortableMock(...args),
}));

vi.mock("@dnd-kit/utilities", () => ({
  CSS: {
    Transform: {
      toString: () => undefined,
    },
  },
}));

describe("TestTargetsRow", () => {
  afterEach(() => {
    cleanup();
  });

  beforeEach(() => {
    useSortableMock.mockReset();
    useSortableMock.mockReturnValue({
      attributes: {},
      listeners: undefined,
      setNodeRef: () => undefined,
      transform: null,
      transition: undefined,
      isDragging: false,
      setActivatorNodeRef: () => undefined,
      setDroppableNodeRef: () => undefined,
      setDraggableNodeRef: () => undefined,
    });
  });

  it("uses a plain drag handle and leaves dragging rows in placeholder state", () => {
    useSortableMock.mockReturnValue({
      attributes: {},
      listeners: undefined,
      setNodeRef: () => undefined,
      transform: null,
      transition: undefined,
      isDragging: true,
      setActivatorNodeRef: () => undefined,
      setDroppableNodeRef: () => undefined,
      setDraggableNodeRef: () => undefined,
    });

    render(
      <TestTargetsRow
        changed
        dragId="10"
        expanded
        index={0}
        onNameChange={() => undefined}
        onRemove={() => undefined}
        onRequestEdit={() => undefined}
        onToggleEnabled={() => undefined}
        onToggleProfile={() => undefined}
        onURLChange={() => undefined}
        target={{
          name: "yt_page",
          url: "https://www.youtube.com/",
          enabled: true,
          profiles: ["browser", "stress"],
          order: 10,
        }}
      />,
    );

    const dragButton = screen.getByRole("button", { name: "Reorder yt_page" });
    const row = dragButton.closest(".drawer-target-row");

    expect(dragButton.className.includes("drawer-target-row__drag-button--plain")).toBe(true);
    expect(row?.className.includes("drawer-target-row--dirty")).toBe(true);
    expect(row?.className.includes("drawer-target-row--placeholder")).toBe(true);
  });

  it("renders a compact summary row until editing is requested", async () => {
    const user = userEvent.setup();
    const onRequestEdit = vi.fn();
    const onToggleEnabled = vi.fn();
    const onToggleProfile = vi.fn();
    const onRemove = vi.fn();

    render(
      <TestTargetsRow
        changed={false}
        dragId="10"
        expanded={false}
        index={0}
        onNameChange={() => undefined}
        onRemove={onRemove}
        onRequestEdit={onRequestEdit}
        onToggleEnabled={onToggleEnabled}
        onToggleProfile={onToggleProfile}
        onURLChange={() => undefined}
        target={{
          name: "yt_page",
          url: "https://www.youtube.com/",
          enabled: true,
          profiles: ["browser", "stress"],
          order: 10,
        }}
      />,
    );

    expect(screen.queryByLabelText("Target Name")).toBeNull();
    expect(screen.getByRole("button", { name: "Edit yt_page" })).toBeTruthy();
    expect(screen.getByText("browser +1")).toBeTruthy();
    expect(screen.getByRole("switch", { name: "Enable yt_page" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Remove yt_page" })).toBeTruthy();

    await user.click(screen.getByRole("button", { name: "Edit yt_page" }));
    expect(onRequestEdit).toHaveBeenCalledTimes(1);

    await user.click(screen.getByRole("switch", { name: "Enable yt_page" }));
    expect(onToggleEnabled).toHaveBeenCalledTimes(1);

    await user.click(screen.getByRole("button", { name: "Profiles for yt_page" }));
    await user.click(screen.getByRole("menuitemcheckbox", { name: "stress" }));
    expect(onToggleProfile).toHaveBeenCalledWith("stress");

    await user.click(screen.getByRole("button", { name: "Remove yt_page" }));
    expect(onRemove).toHaveBeenCalledTimes(1);
  });
});
