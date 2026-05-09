import { QUICK_MTU_VALUES } from "../constants";
import { interfaceKey } from "../controller/interface_state";
import { isInterfaceMutationPendingAction } from "../controller/pending_state";
import {
  ActionButton,
  FieldLabel,
  MetricCard,
  Panel,
  SectionCard,
  StatusChip,
} from "../DashboardUi";
import type {
  DashboardActions,
  DashboardDerivedState,
  DashboardState,
} from "../types";

export function QuickMtuWorkspace(props: {
  state: DashboardState;
  derived: DashboardDerivedState;
  actions: DashboardActions;
  active: boolean;
}) {
  const { state, derived, actions, active } = props;
  const selectedKey = derived.selectedInterfaceKey;
  const selectedInterface = state.selectedInterface;
  const showDetectingInterfaceState = state.detectingInterface && selectedInterface == null;
  const autoDetectBusy = state.interfaceAction === "auto-detect";
  const refreshBusy = state.interfaceAction === "refresh-interface";
  const applyBusy = state.pendingAction === "set-active-mtu";
  const restoreBusy = state.pendingAction === "restore-mtu";
  const persistBusy = state.pendingAction === "set-persistent-mtu";
  const testStartBusy = state.pendingAction === "run-test";
  const interfaceMutationBusy = isInterfaceMutationPendingAction(state.pendingAction);
  const taskStopping =
    derived.taskBusy &&
    derived.currentTaskKind === "connectivity_test" &&
    state.taskState.cancel_requested;

  return (
    <SectionCard
      title="Quick MTU"
      active={active}
      trailing={
        <div className="section-card__actions">
          <StatusChip
            label={state.selectionMode === "manual" ? "Manual" : "Auto"}
            tone={state.selectionMode === "manual" ? "neutral" : "ok"}
          />
          <StatusChip
            label={selectedInterface ? "Selected" : showDetectingInterfaceState ? "Detecting…" : "No Interface"}
            tone={selectedInterface ? "ok" : showDetectingInterfaceState ? "warn" : "neutral"}
          />
        </div>
      }
    >
      <div className="workspace-grid workspace-grid--single">
        <Panel>
          <div className="control-row control-row--interface">
            <label className="field field--select field--grow">
              <FieldLabel
                label="Interface"
                tip="Auto Detect prefers the underlying egress interface for MTU tuning. Manual choices stay in this session until you run Auto Detect again."
              />
              <select
                aria-label="Interface"
                disabled={!derived.canSelectInterface}
                value={selectedKey}
                onChange={(event) => void actions.selectInterface(event.target.value)}
              >
                <option value="">
                  {state.interfaceCandidates.length > 0 ? "Select interface" : "No interface"}
                </option>
                {state.interfaceCandidates.map((candidate) => {
                  const key = interfaceKey(candidate);
                  return (
                    <option key={key} value={key}>
                      {formatInterfaceOption(candidate)}
                    </option>
                  );
                })}
              </select>
              {state.interfaceHint ? <div className="field-hint field-hint--warn">{state.interfaceHint}</div> : null}
            </label>
            <ActionButton
              label="Auto Detect"
              busy={autoDetectBusy}
              onClick={() => void actions.autoDetectInterface(false)}
              disabled={state.booting || derived.taskBusy || autoDetectBusy || interfaceMutationBusy}
              tone="primary"
              data-testid="detect-route"
            />
            <ActionButton
              label="Refresh"
              busy={refreshBusy}
              onClick={() => void actions.refreshInterface()}
              disabled={state.booting || derived.taskBusy || refreshBusy || interfaceMutationBusy || !derived.canEditInterface}
            />
          </div>

          <div className="metric-strip metric-strip--compact">
            <MetricCard
              label="Current MTU"
              value={String(selectedInterface?.mtu ?? "--")}
              caption={selectedInterface?.gateway || selectedInterface?.local_address || "—"}
            />
            <MetricCard label="Original MTU" value={String(state.originalMtu ?? "--")} />
            <div className="metric metric--input">
              <label className="field">
                <FieldLabel label="Target MTU" tip="Quick value used by Apply MTU and Persist." />
                <input
                  aria-label="Target MTU"
                  type="number"
                  min={576}
                  max={9000}
                  value={state.settings.target_mtu}
                  onChange={(event) =>
                    actions.updateSetting("target_mtu", Number.parseInt(event.target.value, 10) || 0)
                  }
                />
              </label>
            </div>
          </div>

          <div className="preset-row">
            {QUICK_MTU_VALUES.map((value) => (
              <button
                key={value}
                className={`pill-button${state.settings.target_mtu === value ? " pill-button--active" : ""}`}
                onClick={() => actions.updateSetting("target_mtu", value)}
                type="button"
              >
                {value}
              </button>
            ))}
          </div>

          <div className="button-grid">
            <ActionButton
              label="Apply MTU"
              busy={applyBusy}
              onClick={() => void actions.setActiveMtu()}
              disabled={state.booting || derived.taskBusy || interfaceMutationBusy || !derived.canEditInterface}
              tone="primary"
            />
            <ActionButton
              label="Restore"
              busy={restoreBusy}
              onClick={() => void actions.restoreMtu()}
              disabled={state.booting || derived.taskBusy || interfaceMutationBusy || !derived.canEditInterface}
            />
            <ActionButton
              label={derived.testButtonLabel}
              busy={testStartBusy}
              onClick={() => void actions.runTest()}
              disabled={testStartBusy || !derived.canRunTest || taskStopping}
              tone="accent"
              data-testid="test-action"
            />
            <ActionButton
              label="Persist"
              busy={persistBusy}
              onClick={() => void actions.setPersistentMtu()}
              disabled={persistBusy || !derived.canPersist}
            />
          </div>

          <div className="progress-wrap progress-wrap--compact">
            <div className="progress-meta">
              <strong data-testid="task-progress-label">{state.taskProgress.label}</strong>
              <span>
                {state.taskProgress.done}/{state.taskProgress.total}
              </span>
            </div>
            <div className="progress-bar">
              <div
                className="progress-bar__fill"
                style={{ width: `${Math.max(0, Math.min(1, derived.progressRatio)) * 100}%` }}
              />
            </div>
            <div className="detail-strip">
              <span>{selectedInterface?.description || "No interface selected"}</span>
              <span>{selectedInterface?.local_address || "—"}</span>
              <span>{selectedInterface?.gateway || "—"}</span>
            </div>
          </div>
        </Panel>
      </div>
    </SectionCard>
  );
}

function formatInterfaceOption(info: DashboardState["selectedInterface"]): string {
  if (!info) {
    return "";
  }
  const parts = [info.name];
  if (info.local_address) {
    parts.push(info.local_address);
  }
  if (info.mtu) {
    parts.push(`MTU ${info.mtu}`);
  }
  return parts.join(" · ");
}
