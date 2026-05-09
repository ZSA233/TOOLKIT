import { useEffect, useRef, useState, type ReactNode } from "react";
import { createPortal } from "react-dom";

import { SortableDraftList } from "./SortableDraftList";
import type {
  DraftListRenderOverlayInput,
  DraftListRenderRowInput,
  DraftListSession,
} from "./draft_list_types";

export const DRAWER_TRANSITION_MS = 220;

export function DraftListEditorDrawer<TItem extends { order: number }>(props: {
  open: boolean;
  title: string;
  description?: string;
  closeLabel?: string;
  addLabel: string;
  doneLabel: string;
  discardTitle?: string;
  discardDescription?: string;
  keepEditingLabel?: string;
  discardLabel?: string;
  session: DraftListSession<TItem>;
  onRequestAdd(): void;
  onClose(): void;
  renderRow(input: DraftListRenderRowInput<TItem>): ReactNode;
  renderOverlay(input: DraftListRenderOverlayInput<TItem>): ReactNode;
}) {
  const previousOpenRef = useRef(props.open);
  const closeTimerRef = useRef<number | null>(null);
  const [phase, setPhase] = useState<"closed" | "open" | "closing">(
    props.open ? "open" : "closed",
  );
  const [discardConfirmOpen, setDiscardConfirmOpen] = useState(false);
  const rendered = props.open || phase !== "closed";
  const visualPhase = props.open ? "open" : phase;

  useEffect(() => {
    if (closeTimerRef.current != null) {
      window.clearTimeout(closeTimerRef.current);
      closeTimerRef.current = null;
    }

    if (props.open) {
      setPhase("open");
      return;
    }

    setPhase((current: "closed" | "open" | "closing") => {
      if (current === "closed") {
        return current;
      }

      closeTimerRef.current = window.setTimeout(() => {
        closeTimerRef.current = null;
        setPhase("closed");
      }, DRAWER_TRANSITION_MS);
      return "closing";
    });
  }, [props.open]);

  useEffect(() => {
    return () => {
      if (closeTimerRef.current != null) {
        window.clearTimeout(closeTimerRef.current);
      }
    };
  }, []);

  useEffect(() => {
    if (props.open && !previousOpenRef.current) {
      setDiscardConfirmOpen(false);
    }

    if (!props.open && previousOpenRef.current) {
      setDiscardConfirmOpen(false);
    }

    previousOpenRef.current = props.open;
  }, [props.open]);

  useEffect(() => {
    if (!rendered) {
      return;
    }

    const previousOverflow = document.body.style.overflow;
    const previousOverscrollBehavior = document.body.style.overscrollBehavior;
    document.body.style.overflow = "hidden";
    document.body.style.overscrollBehavior = "none";

    return () => {
      document.body.style.overflow = previousOverflow;
      document.body.style.overscrollBehavior = previousOverscrollBehavior;
    };
  }, [rendered]);

  if (!rendered) {
    return null;
  }

  const requestClose = () => {
    if (discardConfirmOpen) {
      return;
    }

    if (props.session.hasDirtyChanges) {
      setDiscardConfirmOpen(true);
      return;
    }

    props.onClose();
  };

  const handleDiscardChanges = () => {
    props.session.resetToBaseline();
    setDiscardConfirmOpen(false);
    props.onClose();
  };

  return createPortal(
    <div className="drawer-layer" data-state={visualPhase}>
      <button
        aria-label={props.closeLabel ?? `Close ${props.title}`}
        className="drawer-backdrop"
        onClick={requestClose}
        type="button"
      />
      <aside aria-label={props.title} aria-modal="true" className="drawer-sheet" role="dialog">
        <div className="drawer-sheet__header">
          <div className="drawer-sheet__copy">
            <h2>{props.title}</h2>
            {props.description ? <p>{props.description}</p> : null}
          </div>
          <div className="drawer-sheet__actions">
            <button className="draft-list-action-button draft-list-action-button--ghost" onClick={props.onRequestAdd} type="button">
              {props.addLabel}
            </button>
            <button className="draft-list-action-button draft-list-action-button--primary" onClick={requestClose} type="button">
              {props.doneLabel}
            </button>
          </div>
        </div>

        <div className="drawer-sheet__body">
          <SortableDraftList
            expandedRowId={props.session.expandedRowId}
            onReorder={props.session.reorderRows}
            renderOverlay={props.renderOverlay}
            renderRow={props.renderRow}
            rowChangesById={props.session.rowChangesById}
            rowIds={props.session.rowIds}
            rows={props.session.rows}
          />
        </div>
      </aside>
      {discardConfirmOpen ? (
        <div className="drawer-confirm-layer">
          <div
            aria-label={props.discardTitle ?? "Discard unsaved changes?"}
            aria-modal="true"
            className="drawer-confirm-card"
            role="alertdialog"
          >
            <h3>{props.discardTitle ?? "Discard unsaved changes?"}</h3>
            <p>
              {props.discardDescription
                ?? "Your current edits have not been saved. Discard them and close the drawer?"}
            </p>
            <div className="drawer-confirm-card__actions">
              <button
                className="draft-list-action-button draft-list-action-button--ghost"
                onClick={() => setDiscardConfirmOpen(false)}
                type="button"
              >
                {props.keepEditingLabel ?? "Keep Editing"}
              </button>
              <button
                className="draft-list-action-button draft-list-action-button--danger"
                onClick={handleDiscardChanges}
                type="button"
              >
                {props.discardLabel ?? "Discard Changes"}
              </button>
            </div>
          </div>
        </div>
      ) : null}
    </div>,
    document.body,
  );
}
