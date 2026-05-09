import { useEffect, useRef } from "react";

import { GhostButton, Panel, SectionCard } from "../DashboardUi";
import type {
  DashboardActions,
  DashboardState,
} from "../types";

export function LogsWorkspace(props: {
  state: DashboardState;
  actions: DashboardActions;
  active: boolean;
}) {
  const { state, actions, active } = props;
  const panelRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (!state.autoScrollLogs || !panelRef.current) {
      return;
    }
    panelRef.current.scrollTop = panelRef.current.scrollHeight;
  }, [state.logs, state.autoScrollLogs]);

  return (
    <SectionCard
      title="Logs"
      active={active}
      trailing={
        <div className="section-card__actions">
          <GhostButton
            label={state.autoScrollLogs ? "Auto Scroll On" : "Auto Scroll Off"}
            onClick={() => actions.toggleAutoScrollLogs()}
          />
          <GhostButton label="Clear Logs" onClick={() => actions.clearLogs()} />
        </div>
      }
    >
      <Panel className="panel--log">
        <div aria-label="Task Logs" className="log-list" data-testid="task-log-panel" ref={panelRef} role="log">
          {state.logs.length === 0 ? (
            <div className="log-empty">No task logs yet.</div>
          ) : (
            state.logs.map((entry, index) => (
              <article className="log-row" key={`${entry.ts}-${index}`}>
                <time className="log-row__time">{entry.ts}</time>
                <div className="log-row__line">{entry.line}</div>
              </article>
            ))
          )}
        </div>
      </Panel>
    </SectionCard>
  );
}
