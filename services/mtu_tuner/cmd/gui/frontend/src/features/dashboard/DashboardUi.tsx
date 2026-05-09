import type { ReactNode } from "react";

export function DashboardToolbar(props: {
  title: string;
  subtitle: string;
  meta: ReactNode;
}) {
  return (
    <header className="toolbar">
      <div className="toolbar__copy">
        <div className="toolbar__title">{props.title}</div>
        <div className="toolbar__subtitle">{props.subtitle}</div>
      </div>
      <div className="toolbar__meta">{props.meta}</div>
    </header>
  );
}

export function SectionCard(props: {
  title: string;
  children: ReactNode;
  trailing?: ReactNode;
  active?: boolean;
}) {
  return (
    <section className={`section-card${props.active ? " section-card--active" : ""}`}>
      <div className="section-card__header">
        <h2>{props.title}</h2>
        {props.trailing ? <div className="section-card__trailing">{props.trailing}</div> : null}
      </div>
      {props.children}
    </section>
  );
}

export function Panel(props: { children: ReactNode; className?: string }) {
  return <div className={`panel${props.className ? ` ${props.className}` : ""}`}>{props.children}</div>;
}

export function MetricCard(props: {
  label: string;
  value: string;
  caption?: string;
  trailing?: ReactNode;
}) {
  return (
    <div className="metric">
      <div className="metric__head">
        <small>{props.label}</small>
        {props.trailing}
      </div>
      <strong>{props.value}</strong>
      {props.caption ? <span>{props.caption}</span> : null}
    </div>
  );
}

export function StatusChip(props: {
  label: string;
  tone: "ok" | "warn" | "danger" | "neutral";
  "data-testid"?: string;
}) {
  return (
    <span className={`status-chip status-chip--${props.tone}`} data-testid={props["data-testid"]}>
      {props.label}
    </span>
  );
}

export function Banner(props: {
  tone: "info" | "warn" | "danger";
  children: ReactNode;
  trailing?: ReactNode;
}) {
  return (
    <div className={`banner banner--${props.tone}`}>
      <div className="banner__body">{props.children}</div>
      {props.trailing ? <div className="banner__actions">{props.trailing}</div> : null}
    </div>
  );
}

export function FieldLabel(props: { label: string; tip?: string }) {
  return (
    <span className="field-label">
      {props.label}
      {props.tip ? <HelpTip text={props.tip} /> : null}
    </span>
  );
}

export function FormField(props: {
  label: string;
  value: string | number;
  onChange(value: string): void;
  type?: string;
  disabled?: boolean;
  tip?: string;
}) {
  return (
    <label className="field">
      <FieldLabel label={props.label} tip={props.tip} />
      <input
        aria-label={props.label}
        disabled={props.disabled}
        type={props.type ?? "text"}
        value={props.value}
        onChange={(event) => props.onChange(event.target.value)}
      />
    </label>
  );
}

export function InlinePathField(props: {
  label: string;
  value: string;
  onChange(value: string): void;
  onPick(): void;
  disabled?: boolean;
  tip?: string;
}) {
  return (
    <label className="field">
      <FieldLabel label={props.label} tip={props.tip} />
      <div className="field-inline">
        <input
          aria-label={props.label}
          disabled={props.disabled}
          value={props.value}
          onChange={(event) => props.onChange(event.target.value)}
        />
        <button className="icon-button" disabled={props.disabled} onClick={props.onPick} type="button">
          Pick
        </button>
      </div>
    </label>
  );
}

export function ActionButton(props: {
  label: string;
  onClick(): void;
  disabled?: boolean;
  tone?: "primary" | "accent" | "secondary";
  busy?: boolean;
  "data-testid"?: string;
}) {
  const tone = props.tone ?? "secondary";
  return (
    <button
      aria-busy={props.busy ? "true" : undefined}
      className={`action-button action-button--${tone}${props.busy ? " action-button--busy" : ""}`}
      data-testid={props["data-testid"]}
      disabled={props.disabled}
      onClick={props.onClick}
      type="button"
    >
      <span className="action-button__content">
        <span className="action-button__label">{props.label}</span>
        <span className="action-button__busy-slot" aria-hidden="true">
          {props.busy ? <span className="action-button__busy-dot" /> : null}
        </span>
      </span>
    </button>
  );
}

export function GhostButton(props: {
  label: string;
  onClick(): void;
  disabled?: boolean;
}) {
  return (
    <button className="ghost-button" disabled={props.disabled} onClick={props.onClick} type="button">
      {props.label}
    </button>
  );
}

export function HelpTip(props: { text: string }) {
  return (
    <span className="help-tip">
      <button className="help-tip__trigger" type="button" aria-label={props.text}>
        i
      </button>
      <span className="help-tip__bubble" role="tooltip">
        {props.text}
      </span>
    </span>
  );
}
