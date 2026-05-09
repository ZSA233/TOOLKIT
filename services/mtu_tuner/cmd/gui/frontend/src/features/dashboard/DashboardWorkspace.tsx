import { APP_DISPLAY_NAME } from "./constants";
import { Banner, DashboardToolbar, GhostButton, StatusChip } from "./DashboardUi";
import { pendingActionStatusLabel } from "./controller/pending_state";
import type {
  DashboardActions,
  DashboardDerivedState,
  DashboardState,
  NavKey,
} from "./types";
import { LogsWorkspace } from "./workspace/LogsWorkspace";
import { QuickMtuWorkspace } from "./workspace/QuickMtuWorkspace";
import { RouteWorkspace } from "./workspace/RouteWorkspace";
import { ScanWorkspace } from "./workspace/ScanWorkspace";
import { TestTargetsDrawer } from "./workspace/TestTargetsDrawer";

export function DashboardWorkspace(props: {
  selected: NavKey;
  testTargetsOpen: boolean;
  sectionRefs: Record<NavKey, { current: HTMLDivElement | null }>;
  state: DashboardState;
  derived: DashboardDerivedState;
  actions: DashboardActions;
  onOpenTestTargets(): void;
  onCloseTestTargets(): void;
}) {
  const showDetectingInterfaceState =
    props.state.detectingInterface && props.state.selectedInterface == null;

  return (
    <>
      <DashboardToolbar
        title={APP_DISPLAY_NAME}
        subtitle={
          props.state.selectedInterface
            ? `${props.state.selectedInterface.name} · ${props.state.selectedInterface.local_address ?? "no IPv4"}`
            : showDetectingInterfaceState
              ? "Detecting preferred egress interface…"
              : "No interface selected"
        }
        meta={
          <>
            <StatusChip
              label={props.state.booting ? "Starting" : props.state.ready ? "Ready" : "Blocked"}
              tone={props.state.ready ? "ok" : props.state.error ? "danger" : "warn"}
              data-testid="boot-state"
            />
            <StatusChip
              label={
                showDetectingInterfaceState
                  ? "detecting-interface"
                  : pendingActionStatusLabel(props.state.pendingAction)
              }
              tone={showDetectingInterfaceState || props.state.pendingAction ? "warn" : "neutral"}
            />
          </>
        }
      />

      <div className="notice-stack">
        {props.derived.adminWarning ? (
          <Banner
            tone="warn"
            trailing={
              props.derived.canPromptAdminRelaunch ? (
                <GhostButton
                  label="Relaunch as Admin"
                  onClick={() => void props.actions.promptAdminRelaunch()}
                />
              ) : undefined
            }
          >
            {props.derived.adminWarning}
          </Banner>
        ) : null}
        {props.state.warning ? <Banner tone="warn">{props.state.warning}</Banner> : null}
        {props.state.error ? <Banner tone="danger">{props.state.error}</Banner> : null}
      </div>

      <div className="dashboard-stack">
        <div className="workspace-anchor" data-section="overview" ref={props.sectionRefs.overview}>
          <QuickMtuWorkspace
            active={props.selected === "overview"}
            state={props.state}
            derived={props.derived}
            actions={props.actions}
          />
        </div>
        <div className="workspace-anchor" data-section="route" ref={props.sectionRefs.route}>
          <RouteWorkspace
            active={props.selected === "route"}
            state={props.state}
            derived={props.derived}
            actions={props.actions}
          />
        </div>
        <div className="workspace-anchor" data-section="scan" ref={props.sectionRefs.scan}>
          <ScanWorkspace
            active={props.selected === "scan"}
            state={props.state}
            derived={props.derived}
            actions={props.actions}
            onOpenTestTargets={props.onOpenTestTargets}
          />
        </div>
        <div className="workspace-anchor" data-section="logs" ref={props.sectionRefs.logs}>
          <LogsWorkspace
            active={props.selected === "logs"}
            state={props.state}
            actions={props.actions}
          />
        </div>
      </div>
      <TestTargetsDrawer
        actions={props.actions}
        open={props.testTargetsOpen}
        settings={props.state.settings}
        onClose={props.onCloseTestTargets}
      />
    </>
  );
}
