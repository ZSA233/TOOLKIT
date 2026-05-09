import { useEffect, useRef, useState } from "react";

import { TEST_PROFILES } from "../constants";
import type { TestTargetProfile } from "../types";
import { summarizeProfiles } from "./TestTargetsDraftList";

export function TestTargetsProfileMenu(props: {
  targetLabel: string;
  profiles: TestTargetProfile[];
  onToggle(profile: TestTargetProfile): void;
}) {
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (!open) {
      return;
    }

    const handlePointerDown = (event: PointerEvent) => {
      if (!rootRef.current?.contains(event.target as Node)) {
        setOpen(false);
      }
    };
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setOpen(false);
      }
    };

    document.addEventListener("pointerdown", handlePointerDown);
    document.addEventListener("keydown", handleKeyDown);
    return () => {
      document.removeEventListener("pointerdown", handlePointerDown);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [open]);

  return (
    <div className="drawer-profile-menu" ref={rootRef}>
      <button
        aria-expanded={open ? "true" : "false"}
        aria-haspopup="menu"
        aria-label={`Profiles for ${props.targetLabel}`}
        className="drawer-profile-menu__trigger"
        onClick={() => setOpen((current) => !current)}
        type="button"
      >
        <span>{summarizeProfiles(props.profiles)}</span>
        <span aria-hidden="true" className="drawer-profile-menu__chevron">
          +
        </span>
      </button>

      {open ? (
        <div
          aria-label={`Profiles for ${props.targetLabel}`}
          className="drawer-profile-menu__panel"
          role="menu"
        >
          {TEST_PROFILES.map((profile) => {
            const selected = props.profiles.includes(profile);
            return (
              <button
                aria-checked={selected ? "true" : "false"}
                className={`drawer-profile-menu__item${
                  selected ? " drawer-profile-menu__item--selected" : ""
                }`}
                key={profile}
                onClick={() => props.onToggle(profile)}
                role="menuitemcheckbox"
                type="button"
              >
                <span>{profile}</span>
                <span aria-hidden="true" className="drawer-profile-menu__check">
                  {selected ? "+" : ""}
                </span>
              </button>
            );
          })}
        </div>
      ) : null}
    </div>
  );
}
