import { APP_DISPLAY_NAME } from "./constants";
import { StatusChip } from "./DashboardUi";
import type { DashboardDerivedState, DashboardState, NavKey } from "./types";

const NAV_ITEMS: Array<{ key: NavKey; label: string }> = [
  { key: "overview", label: "Quick MTU" },
  { key: "route", label: "Route & Clash" },
  { key: "scan", label: "Scan" },
  { key: "logs", label: "Logs" },
];

export function DashboardSidebar(props: {
  selected: NavKey;
  onSelect(key: NavKey): void;
  state: DashboardState;
  derived: DashboardDerivedState;
}) {
  const activeTaskKind =
    props.derived.taskBusy && props.state.taskState.kind
      ? props.state.taskState.kind
      : "idle";
  const showDetectingInterfaceState =
    props.state.detectingInterface && props.state.selectedInterface == null;
  const taskLabel = props.state.booting
    ? "Booting"
    : showDetectingInterfaceState
      ? "Detecting"
      : props.state.taskState.status;

  return (
    <aside className="side-nav">
      <div className="side-nav__brand">
        <div className="side-nav__eyebrow">ToolKit</div>
        <div className="side-nav__headline">{APP_DISPLAY_NAME}</div>
      </div>

      <div className="side-nav__status">
        <StatusChip
          label={taskLabel}
          tone={
            props.state.error
              ? "danger"
              : props.derived.taskBusy || showDetectingInterfaceState
                ? "warn"
                : "ok"
          }
        />
        <StatusChip
          label={props.state.selectionMode === "manual" ? "Manual Select" : "Auto Select"}
          tone={props.state.selectionMode === "manual" ? "neutral" : "ok"}
        />
        <StatusChip
          label={props.state.status?.is_admin ? "Admin/Root" : "User Mode"}
          tone={props.state.status?.is_admin ? "ok" : "neutral"}
        />
      </div>

      <nav className="side-nav__menu" aria-label="Workspace">
        {NAV_ITEMS.map((item) => (
          <button
            key={item.key}
            className={`nav-link${props.selected === item.key ? " nav-link--active" : ""}`}
            onClick={() => props.onSelect(item.key)}
            type="button"
          >
            <span>{item.label}</span>
          </button>
        ))}
      </nav>

      <div className="side-nav__footer">
        <div>Platform: {props.state.status?.platform_name ?? "loading"}</div>
        <div>Interface: {props.state.selectedInterface?.name ?? "—"}</div>
        <div>Task: {activeTaskKind}</div>
      </div>
    </aside>
  );
}
