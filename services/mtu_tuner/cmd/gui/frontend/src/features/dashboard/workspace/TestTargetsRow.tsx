import type { DraggableAttributes, DraggableSyntheticListeners } from "@dnd-kit/core";
import { useSortableDraftRow } from "@toolkit/appkit-webui";

import type { TestTargetProfile, TestTargetSettings } from "../types";
import { summarizeProfiles, targetDisplayLabel } from "./TestTargetsDraftList";
import { TestTargetsProfileMenu } from "./TestTargetsProfileMenu";

function TestTargetsRowBody(props: {
  target: TestTargetSettings;
  index: number;
  changed?: boolean;
  expanded?: boolean;
  dragging?: boolean;
  overlay?: boolean;
  placeholder?: boolean;
  dragAttributes?: DraggableAttributes;
  dragListeners?: DraggableSyntheticListeners;
  dragActivatorRef?: (node: HTMLButtonElement | null) => void;
  onRequestEdit?(): void;
  onNameChange?(value: string): void;
  onURLChange?(value: string): void;
  onToggleEnabled?(): void;
  onToggleProfile?(profile: TestTargetProfile): void;
  onRemove?(): void;
}) {
  const targetLabel = targetDisplayLabel(props.target, props.index);
  const interactive = !props.overlay;
  const expanded = interactive && Boolean(props.expanded);
  const compact = !props.overlay && !expanded;

  return (
    <div
      className={`drawer-target-row${props.changed ? " drawer-target-row--dirty" : ""}${
        expanded ? " drawer-target-row--expanded" : ""
      }${
        compact ? " drawer-target-row--compact" : ""
      }${props.dragging ? " drawer-target-row--dragging" : ""}${
        props.placeholder ? " drawer-target-row--placeholder" : ""
      }${
        props.overlay ? " drawer-target-row--overlay" : ""
      }`}
    >
      <div className="drawer-target-row__handle">
        {interactive ? (
          <button
            aria-label={`Reorder ${targetLabel}`}
            className="drawer-target-row__drag-button drawer-target-row__drag-button--plain"
            ref={props.dragActivatorRef}
            type="button"
            {...props.dragAttributes}
            {...props.dragListeners}
          >
            <span aria-hidden="true" className="drawer-target-row__drag-glyph" />
          </button>
        ) : (
          <span aria-hidden="true" className="drawer-target-row__drag-glyph" />
        )}
      </div>

      <div className="drawer-target-row__main">
        {expanded ? (
          <>
            <input
              aria-label="Target Name"
              className="drawer-target-row__name"
              placeholder="Target name"
              value={props.target.name}
              onChange={(event) => props.onNameChange?.(event.target.value)}
            />
            <input
              aria-label="Target URL"
              className="drawer-target-row__url"
              placeholder="https://"
              value={props.target.url}
              onChange={(event) => props.onURLChange?.(event.target.value)}
            />
          </>
        ) : compact ? (
          <button
            aria-label={`Edit ${targetLabel}`}
            className="drawer-target-row__summary-button"
            onClick={() => props.onRequestEdit?.()}
            type="button"
          >
            <div className="drawer-target-row__overlay-name">{targetLabel}</div>
            <div className="drawer-target-row__overlay-url">{props.target.url}</div>
          </button>
        ) : (
          <>
            <div className="drawer-target-row__overlay-name">{targetLabel}</div>
            <div className="drawer-target-row__overlay-url">{props.target.url}</div>
          </>
        )}
      </div>

      <div className="drawer-target-row__profiles">
        {interactive ? (
          <TestTargetsProfileMenu
            targetLabel={targetLabel}
            profiles={props.target.profiles}
            onToggle={(profile) => props.onToggleProfile?.(profile)}
          />
        ) : (
          <div className="drawer-profile-menu__trigger drawer-profile-menu__trigger--static">
            {summarizeProfiles(props.target.profiles)}
          </div>
        )}
      </div>

      <div className="drawer-target-row__switch-slot">
        {interactive ? (
          <button
            aria-checked={props.target.enabled ? "true" : "false"}
            aria-label={`Enable ${targetLabel}`}
            className={`drawer-target-row__switch${
              props.target.enabled ? " drawer-target-row__switch--on" : ""
            }`}
            onClick={() => props.onToggleEnabled?.()}
            role="switch"
            type="button"
          >
            <span aria-hidden="true" className="drawer-target-row__switch-track">
              <span className="drawer-target-row__switch-thumb" />
            </span>
          </button>
        ) : (
          <span
            aria-hidden="true"
            className={`drawer-target-row__switch drawer-target-row__switch--static${
              props.target.enabled ? " drawer-target-row__switch--on" : ""
            }`}
          >
            <span className="drawer-target-row__switch-track">
              <span className="drawer-target-row__switch-thumb" />
            </span>
          </span>
        )}
      </div>

      <div className="drawer-target-row__remove">
        {interactive ? (
          <button
            aria-label={`Remove ${targetLabel}`}
            className="drawer-target-row__icon-button"
            onClick={() => props.onRemove?.()}
            type="button"
          >
            <span aria-hidden="true" className="drawer-target-row__remove-glyph">
              +
            </span>
          </button>
        ) : (
          <span
            aria-hidden="true"
            className="drawer-target-row__icon-button drawer-target-row__icon-button--static"
          >
            <span className="drawer-target-row__remove-glyph">+</span>
          </span>
        )}
      </div>
    </div>
  );
}

export function TestTargetsRow(props: {
  changed?: boolean;
  dragId: string;
  target: TestTargetSettings;
  index: number;
  expanded: boolean;
  onRequestEdit(): void;
  onNameChange(value: string): void;
  onURLChange(value: string): void;
  onToggleEnabled(): void;
  onToggleProfile(profile: TestTargetProfile): void;
  onRemove(): void;
}) {
  const sortableRow = useSortableDraftRow(props.dragId);

  return (
    <div className="drawer-target-row__item" ref={sortableRow.setRowRef} style={sortableRow.style}>
      <TestTargetsRowBody
        dragAttributes={sortableRow.dragAttributes}
        dragActivatorRef={sortableRow.setHandleRef}
        dragListeners={sortableRow.dragListeners}
        changed={props.changed}
        dragging={sortableRow.isDragging}
        expanded={props.expanded}
        index={props.index}
        onNameChange={props.onNameChange}
        onRemove={props.onRemove}
        onRequestEdit={props.onRequestEdit}
        onToggleEnabled={props.onToggleEnabled}
        onToggleProfile={props.onToggleProfile}
        onURLChange={props.onURLChange}
        placeholder={sortableRow.isDragging}
        target={props.target}
      />
    </div>
  );
}

export function TestTargetsRowOverlay(props: {
  target: TestTargetSettings;
  index: number;
}) {
  return <TestTargetsRowBody dragging overlay index={props.index} target={props.target} />;
}
