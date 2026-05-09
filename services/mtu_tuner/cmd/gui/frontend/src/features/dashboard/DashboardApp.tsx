import { useEffect, useRef, useState } from "react";

import type { DashboardDeps } from "./deps";
import { DashboardSidebar } from "./DashboardSidebar";
import { DashboardWorkspace } from "./DashboardWorkspace";
import type { NavKey } from "./types";
import { useDashboardController } from "./useDashboardController";

const SECTION_KEYS: NavKey[] = ["overview", "route", "scan", "logs"];
const VIEWPORT_FOCUS_OFFSET = 164;

export function DashboardApp(props: { deps: DashboardDeps }) {
  const [selected, setSelected] = useState<NavKey>("overview");
  const [testTargetsOpen, setTestTargetsOpen] = useState(false);
  const { state, derived, actions } = useDashboardController(props.deps);
  const overviewRef = useRef<HTMLDivElement | null>(null);
  const routeRef = useRef<HTMLDivElement | null>(null);
  const scanRef = useRef<HTMLDivElement | null>(null);
  const logsRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const syncSelectedFromScroll = () => {
      const next = resolveFocusedSection({
        overview: overviewRef.current,
        route: routeRef.current,
        scan: scanRef.current,
        logs: logsRef.current,
      });
      setSelected((current) => (current === next ? current : next));
    };

    syncSelectedFromScroll();
    window.addEventListener("scroll", syncSelectedFromScroll, { passive: true });
    window.addEventListener("resize", syncSelectedFromScroll);
    return () => {
      window.removeEventListener("scroll", syncSelectedFromScroll);
      window.removeEventListener("resize", syncSelectedFromScroll);
    };
  }, []);

  const focusSection = (key: NavKey) => {
    setSelected(key);
    const target = resolveSectionElement(key, {
      overview: overviewRef.current,
      route: routeRef.current,
      scan: scanRef.current,
      logs: logsRef.current,
    });
    target?.scrollIntoView?.({ behavior: "smooth", block: "start" });
  };

  return (
    <>
      {state.notice ? (
        <div aria-atomic="true" aria-live="polite" className="toast-layer" role="status">
          <div className={`toast toast--${state.noticeTone}`}>{state.notice}</div>
        </div>
      ) : null}
      <div className="dashboard-shell">
        <DashboardSidebar
          selected={selected}
          onSelect={focusSection}
          state={state}
          derived={derived}
        />
        <main className="dashboard-main">
          <DashboardWorkspace
            selected={selected}
            testTargetsOpen={testTargetsOpen}
            sectionRefs={{
              overview: overviewRef,
              route: routeRef,
              scan: scanRef,
              logs: logsRef,
            }}
            state={state}
            derived={derived}
            actions={actions}
            onOpenTestTargets={() => setTestTargetsOpen(true)}
            onCloseTestTargets={() => setTestTargetsOpen(false)}
          />
        </main>
      </div>
    </>
  );
}

function resolveSectionElement(
  key: NavKey,
  refs: Record<NavKey, HTMLDivElement | null>,
): HTMLDivElement | null {
  return refs[key];
}

function resolveFocusedSection(refs: Record<NavKey, HTMLDivElement | null>): NavKey {
  const focusLine = Math.min(window.innerHeight * 0.28, VIEWPORT_FOCUS_OFFSET);
  let fallback: NavKey | null = null;

  for (const key of SECTION_KEYS) {
    const node = refs[key];
    if (!node) {
      continue;
    }
    const rect = node.getBoundingClientRect();
    if (rect.height === 0 && rect.top === 0 && rect.bottom === 0) {
      continue;
    }
    fallback = key;
    if (rect.top <= focusLine && rect.bottom > focusLine) {
      return key;
    }
    if (rect.top > focusLine) {
      return key;
    }
  }

  return fallback ?? "overview";
}
