import { ChevronDown } from "lucide-react";
import type { ReactNode, SyntheticEvent } from "react";

interface SettingsDisclosureProps {
  title: ReactNode;
  description?: ReactNode;
  meta?: ReactNode;
  children: ReactNode;
  className?: string;
  defaultOpen?: boolean;
  onToggle?: (event: SyntheticEvent<HTMLDetailsElement>) => void;
}

/**
 * Shared disclosure for settings surfaces.
 *
 * Native <details> keeps keyboard and screen-reader behaviour predictable;
 * the shared header prevents each settings section from inventing its own
 * chevron, Show/Hide label, spacing, and hit target.
 */
export function SettingsDisclosure({
  title,
  description,
  meta,
  children,
  className = "",
  defaultOpen = false,
  onToggle,
}: SettingsDisclosureProps) {
  return (
    <details
      className={`settings-disclosure ${className}`.trim()}
      open={defaultOpen || undefined}
      onToggle={onToggle}
    >
      <summary>
        <span className="settings-disclosure-copy">
          <strong>{title}</strong>
          {description ? <small>{description}</small> : null}
        </span>
        <span className="settings-disclosure-end">
          {meta ? <span className="settings-disclosure-meta">{meta}</span> : null}
          <ChevronDown size={17} aria-hidden="true" />
        </span>
      </summary>
      <div className="settings-disclosure-content">{children}</div>
    </details>
  );
}
