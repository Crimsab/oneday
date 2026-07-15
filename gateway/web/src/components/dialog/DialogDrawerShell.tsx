import { useEffect, useId, useRef, type ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { X } from "lucide-react";

interface DialogDrawerShellProps {
  title: string;
  className?: string;
  children: ReactNode;
  onClose: () => void;
}

export function DialogDrawerShell({ title, className = "", children, onClose }: DialogDrawerShellProps) {
  const { t } = useTranslation("drawer");
  const dialogRef = useRef<HTMLElement>(null);
  const titleId = useId();
  const onCloseRef = useRef(onClose);
  onCloseRef.current = onClose;

  useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog) return;
    const previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    const focusableElements = () => Array.from(dialog.querySelectorAll<HTMLElement>(
      'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"]), [contenteditable="true"]',
    )).filter((element) => element.getClientRects().length > 0 && element.getAttribute("aria-hidden") !== "true");

    focusableElements()[0]?.focus();
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        event.preventDefault();
        event.stopPropagation();
        onCloseRef.current();
        return;
      }
      if (event.key !== "Tab") return;
      const focusable = focusableElements();
      if (focusable.length === 0) {
        event.preventDefault();
        return;
      }
      const first = focusable[0];
      const last = focusable.at(-1)!;
      if (event.shiftKey && (document.activeElement === first || !dialog.contains(document.activeElement))) {
        event.preventDefault();
        last.focus();
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first.focus();
      }
    };
    document.addEventListener("keydown", handleKeyDown);
    return () => {
      document.removeEventListener("keydown", handleKeyDown);
      previousFocus?.focus();
    };
  }, []);

  return (
    <div className="overlay-backdrop" role="presentation" onMouseDown={onClose}>
      <section
        ref={dialogRef}
        className={`overlay-panel ${className}`.trim()}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        onMouseDown={(event) => event.stopPropagation()}
      >
        <div className="overlay-head">
          <h2 id={titleId}>{title}</h2>
          <button type="button" className="square-button" onClick={onClose} aria-label={t("close")} title={t("close")}>
            <X size={16} />
          </button>
        </div>
        {children}
      </section>
    </div>
  );
}
