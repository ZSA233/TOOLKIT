import { TEST_PROFILES } from "../constants";
import { isTaskPendingAction } from "../controller/pending_state";
import {
  ActionButton,
  FieldLabel,
  FormField,
  Panel,
  SectionCard,
} from "../DashboardUi";
import type {
  DashboardActions,
  DashboardDerivedState,
  DashboardState,
  SessionSettings,
} from "../types";
import { DetailRow } from "./DetailRow";

export function ScanWorkspace(props: {
  state: DashboardState;
  derived: DashboardDerivedState;
  actions: DashboardActions;
  active: boolean;
  onOpenTestTargets(): void;
}) {
  const { state, derived, actions, active, onOpenTestTargets } = props;
  const taskKindLabel = derived.taskBusy && state.taskState.kind ? state.taskState.kind : "idle";
  const sweepStartBusy = state.pendingAction === "run-sweep";
  const cancelBusy = state.pendingAction === "cancel-task";
  const scanConfigBusy = state.booting || derived.taskBusy || isTaskPendingAction(state.pendingAction);
  const sweepStopping =
    derived.taskBusy &&
    derived.currentTaskKind === "mtu_sweep" &&
    state.taskState.cancel_requested;

  return (
    <SectionCard title="Scan" active={active}>
      <div className="workspace-grid">
        <Panel>
          <div className="form-grid">
            <label className="field">
              <FieldLabel label="Test Profile" />
              <select
                aria-label="Test Profile"
                disabled={scanConfigBusy}
                value={state.settings.test_profile}
                onChange={(event) =>
                  actions.updateSetting("test_profile", event.target.value as SessionSettings["test_profile"])
                }
              >
                {TEST_PROFILES.map((profile) => (
                  <option key={profile} value={profile}>
                    {profile}
                  </option>
                ))}
              </select>
            </label>
            <FormField
              label="Rounds"
              value={state.settings.rounds}
              onChange={(value) => actions.updateSetting("rounds", value)}
              disabled={scanConfigBusy}
              type="number"
            />
            <FormField
              label="Concurrency"
              value={state.settings.concurrency}
              onChange={(value) => actions.updateSetting("concurrency", value)}
              disabled={scanConfigBusy}
              type="number"
            />
            <FormField
              label="Sweep MTUs"
              value={state.settings.sweep_mtus}
              onChange={(value) => actions.updateSetting("sweep_mtus", value)}
              disabled={scanConfigBusy}
              tip="Comma or space separated. Sweep always restores the original MTU."
            />
          </div>
        </Panel>

        <Panel>
          <div className="button-grid">
            <ActionButton
              label="Test Targets"
              onClick={onOpenTestTargets}
              disabled={scanConfigBusy}
            />
            <ActionButton
              label={derived.sweepButtonLabel}
              busy={sweepStartBusy}
              onClick={() => void actions.runSweep()}
              disabled={sweepStartBusy || !derived.canRunSweep || sweepStopping}
              tone="primary"
            />
            <ActionButton
              label="Cancel Task"
              busy={cancelBusy}
              onClick={() => void actions.cancelTask()}
              disabled={!derived.canCancel || cancelBusy}
            />
          </div>

          <dl className="detail-grid">
            <DetailRow label="Task" value={taskKindLabel} />
            <DetailRow label="Status" value={state.taskState.status} />
            <DetailRow label="Cancel Requested" value={state.taskState.cancel_requested ? "yes" : "no"} />
            <DetailRow
              label="Persistent MTU"
              value={state.status?.supports_persistent_mtu ? "supported" : "session-only"}
            />
          </dl>
        </Panel>
      </div>
    </SectionCard>
  );
}
