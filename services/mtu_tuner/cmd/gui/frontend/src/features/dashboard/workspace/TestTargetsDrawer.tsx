import {
  DraftListEditorDrawer,
  DRAWER_TRANSITION_MS,
  useDraftListSession,
} from "@toolkit/appkit-webui";

import type { DashboardActions, SessionSettings } from "../types";
import {
  assignMovedTargetOrder,
  cloneTarget,
  cloneTargets,
  createTargetDraft,
  targetSettingsEqual,
  toggleProfile,
} from "./TestTargetsDraftList";
import { TestTargetsRow, TestTargetsRowOverlay } from "./TestTargetsRow";

export { DRAWER_TRANSITION_MS };

export function TestTargetsDrawer(props: {
  open: boolean;
  settings: SessionSettings;
  actions: DashboardActions;
  onClose(): void;
}) {
  const session = useDraftListSession({
    open: props.open,
    items: props.settings.test_targets,
    cloneItem: cloneTarget,
    itemsEqual: targetSettingsEqual,
    createNewItem: (targets) => createTargetDraft(cloneTargets(targets)),
    assignMovedOrder: ({ previousOrder, nextOrder }) =>
      assignMovedTargetOrder({ previousOrder, nextOrder }),
    resequenceOrder: (index) => (index + 1) * 10,
    onDraftChange: (targets) => props.actions.updateSetting("test_targets", targets),
  });

  return (
    <DraftListEditorDrawer
      addLabel="Add Target"
      closeLabel="Close test target configuration"
      description="Adjust the named URLs used by scan profiles. Changes stay local until you use Save."
      discardDescription="Your edits in Test Targets have not been saved. Discard them and close the drawer?"
      discardTitle="Discard unsaved test target changes?"
      doneLabel="Done"
      keepEditingLabel="Keep Editing"
      onClose={props.onClose}
      onRequestAdd={() => session.appendNewItem()}
      open={props.open}
      renderOverlay={({ row, index }) => <TestTargetsRowOverlay index={index} target={row.item} />}
      renderRow={({ row, index, changed, expanded }) => (
        <TestTargetsRow
          changed={changed}
          dragId={row.rowId}
          expanded={expanded}
          index={index}
          key={row.rowId}
          onNameChange={(value) =>
            session.updateRow(row.rowId, (target) => ({
              ...target,
              name: value,
            }))}
          onRemove={() => session.removeRow(row.rowId)}
          onRequestEdit={() => session.setExpandedRowId(row.rowId)}
          onToggleEnabled={() =>
            session.updateRow(row.rowId, (target) => ({
              ...target,
              enabled: !target.enabled,
            }))}
          onToggleProfile={(profile) =>
            session.updateRow(row.rowId, (target) => ({
              ...target,
              profiles: toggleProfile(target, profile),
            }))}
          onURLChange={(value) =>
            session.updateRow(row.rowId, (target) => ({
              ...target,
              url: value,
            }))}
          target={row.item}
        />
      )}
      session={session}
      title="Advanced Test Targets"
    />
  );
}
