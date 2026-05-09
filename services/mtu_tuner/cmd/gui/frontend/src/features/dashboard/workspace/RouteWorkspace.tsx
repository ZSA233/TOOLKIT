import { isInterfaceMutationPendingAction } from "../controller/pending_state";
import {
  ActionButton,
  FormField,
  InlinePathField,
  Panel,
  SectionCard,
} from "../DashboardUi";
import type {
  DashboardActions,
  DashboardDerivedState,
  DashboardState,
} from "../types";
import { DetailRow } from "./DetailRow";

export function RouteWorkspace(props: {
  state: DashboardState;
  derived: DashboardDerivedState;
  actions: DashboardActions;
  active: boolean;
}) {
  const { state, derived, actions, active } = props;
  const clashDetectBusy = state.interfaceAction === "detect-clash-current";
  const resolveBusy = state.pendingAction === "resolve-clash-target";
  const saveBusy = state.pendingAction === "save-settings";
  const reloadBusy = state.pendingAction === "reload-settings";
  const interfaceMutationBusy = isInterfaceMutationPendingAction(state.pendingAction);

  return (
    <SectionCard title="Route & Clash" active={active}>
      <div className="workspace-grid">
        <Panel>
          <div className="form-grid">
            <FormField
              label="Route Probe"
              value={state.settings.route_probe}
              onChange={(value) => actions.updateSetting("route_probe", value)}
              disabled={!derived.canEditConfig}
              tip="Use auto for Clash-aware route detection, or fill a direct probe IP/domain."
            />
            <FormField
              label="Fallback Probe"
              value={state.settings.fallback_probe}
              onChange={(value) => actions.updateSetting("fallback_probe", value)}
              disabled={!derived.canEditConfig}
            />
            <FormField
              label="HTTP Proxy"
              value={state.settings.http_proxy}
              onChange={(value) => actions.updateSetting("http_proxy", value)}
              disabled={!derived.canEditConfig}
            />
            <FormField
              label="Clash API"
              value={state.settings.clash_api}
              onChange={(value) => actions.updateSetting("clash_api", value)}
              disabled={!derived.canEditConfig}
            />
            <FormField
              label="Proxy Group"
              value={state.settings.proxy_group}
              onChange={(value) => actions.updateSetting("proxy_group", value)}
              disabled={!derived.canEditConfig}
            />
            <FormField
              label="Clash Secret"
              value={state.settings.clash_secret}
              onChange={(value) => actions.updateSetting("clash_secret", value)}
              disabled={!derived.canEditConfig}
              type="password"
              tip="Session-only. This field is never written to config.json."
            />
            <InlinePathField
              label="Clash Config"
              value={state.settings.config_path}
              onChange={(value) => actions.updateSetting("config_path", value)}
              onPick={() => void actions.pickClashConfigPath()}
              disabled={!derived.canEditConfig}
            />
            <InlinePathField
              label="Browser Path"
              value={state.settings.browser_path}
              onChange={(value) => actions.updateSetting("browser_path", value)}
              onPick={() => void actions.pickBrowserPath()}
              disabled={!derived.canEditConfig}
            />
          </div>
        </Panel>

        <Panel>
          <div className="button-grid">
            <ActionButton
              label="Use Current Clash Node"
              busy={clashDetectBusy}
              onClick={() => void actions.autoDetectInterface(true)}
              disabled={state.booting || derived.taskBusy || clashDetectBusy || interfaceMutationBusy}
              tone="primary"
            />
            <ActionButton
              label="Resolve Clash"
              busy={resolveBusy}
              onClick={() => void actions.resolveClashTarget()}
              disabled={state.booting || derived.taskBusy || resolveBusy}
            />
            <ActionButton
              label="Save"
              busy={saveBusy}
              onClick={() => void actions.saveSettings()}
              disabled={state.booting || derived.taskBusy || saveBusy || reloadBusy}
              data-testid="save-config"
            />
            <ActionButton
              label="Reload"
              busy={reloadBusy}
              onClick={() => void actions.reloadSettings()}
              disabled={state.booting || derived.taskBusy || saveBusy || reloadBusy}
            />
          </div>

          {state.clashTarget ? (
            <dl className="detail-grid">
              <DetailRow label="Group" value={state.clashTarget.group} />
              <DetailRow label="Leaf" value={state.clashTarget.leaf} />
              <DetailRow label="Server" value={`${state.clashTarget.server}:${state.clashTarget.port ?? "-"}`} />
              <DetailRow
                label="Resolved"
                value={`${state.clashTarget.resolved_ip} (${state.clashTarget.source ?? "unknown"})`}
              />
              <DetailRow label="Config" value={state.clashTarget.config_path || "—"} />
            </dl>
          ) : (
            <div className="detail-placeholder" data-testid="clash-target-placeholder">
              <div className="empty-copy">No Clash target resolved yet.</div>
              <dl className="detail-grid detail-grid--placeholder">
                <DetailRow label="Group" value="—" />
                <DetailRow label="Leaf" value="—" />
                <DetailRow label="Server" value="—" />
                <DetailRow label="Resolved" value="—" />
                <DetailRow label="Config" value="—" />
              </dl>
            </div>
          )}
        </Panel>
      </div>
    </SectionCard>
  );
}
