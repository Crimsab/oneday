import { Check, ChevronDown } from "lucide-react";
import { createPortal } from "react-dom";
import { useEffect, useId, useLayoutEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";

export interface SelectOption {
  value: string;
  label: string;
  disabled?: boolean;
}

interface CustomSelectProps {
  value: string;
  options: SelectOption[];
  onChange: (value: string) => void;
  disabled?: boolean;
  ariaLabel?: string;
  className?: string;
}

export function CustomSelect({ value, options, onChange, disabled = false, ariaLabel, className = "" }: CustomSelectProps) {
  const { t } = useTranslation("controls");
  const id = useId();
  const triggerRef = useRef<HTMLButtonElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);
  const [open, setOpen] = useState(false);
  const [activeIndex, setActiveIndex] = useState(() => Math.max(0, options.findIndex((option) => option.value === value)));
  const [position, setPosition] = useState({ top: 0, left: 0, width: 180 });
  const selected = options.find((option) => option.value === value) ?? options[0];
  const accessibleLabel = ariaLabel || t("select.label");

  const placeMenu = () => {
    const trigger = triggerRef.current;
    if (!trigger) return;
    const rect = trigger.getBoundingClientRect();
    const menuHeight = menuRef.current?.getBoundingClientRect().height ?? Math.min(300, options.length * 36 + 10);
    const below = window.innerHeight - rect.bottom;
    const top = below >= Math.min(menuHeight, 260) || rect.top < menuHeight
      ? rect.bottom + 5
      : Math.max(8, rect.top - menuHeight - 5);
    setPosition({
      top,
      left: Math.min(Math.max(8, rect.left), Math.max(8, window.innerWidth - rect.width - 8)),
      width: rect.width,
    });
  };

  useLayoutEffect(() => {
    if (!open) return;
    placeMenu();
    const frame = window.requestAnimationFrame(placeMenu);
    return () => window.cancelAnimationFrame(frame);
  }, [open, options.length]);

  useEffect(() => {
    if (!open) return;
    const closeOutside = (event: PointerEvent) => {
      const target = event.target as Node;
      if (!triggerRef.current?.contains(target) && !menuRef.current?.contains(target)) setOpen(false);
    };
    const reposition = () => placeMenu();
    window.addEventListener("pointerdown", closeOutside, true);
    window.addEventListener("resize", reposition);
    window.addEventListener("scroll", reposition, true);
    return () => {
      window.removeEventListener("pointerdown", closeOutside, true);
      window.removeEventListener("resize", reposition);
      window.removeEventListener("scroll", reposition, true);
    };
  }, [open]);

  useEffect(() => {
    const selectedIndex = options.findIndex((option) => option.value === value);
    if (selectedIndex >= 0) setActiveIndex(selectedIndex);
  }, [options, value]);

  const move = (direction: -1 | 1) => {
    if (!options.length) return;
    let next = activeIndex;
    for (let index = 0; index < options.length; index += 1) {
      next = (next + direction + options.length) % options.length;
      if (!options[next]?.disabled) break;
    }
    setActiveIndex(next);
  };

  const choose = (option: SelectOption) => {
    if (option.disabled) return;
    onChange(option.value);
    setOpen(false);
    window.requestAnimationFrame(() => triggerRef.current?.focus());
  };

  return (
    <div className={`custom-select ${className}`.trim()}>
      <button
        ref={triggerRef}
        type="button"
        className="custom-select-trigger"
        role="combobox"
        aria-label={accessibleLabel}
        aria-controls={`${id}-options`}
        aria-expanded={open}
        aria-haspopup="listbox"
        disabled={disabled}
        onClick={() => setOpen((value) => !value)}
        onKeyDown={(event) => {
          if (event.key === "Escape") { setOpen(false); return; }
          if (event.key === "ArrowDown" || event.key === "ArrowUp") {
            event.preventDefault();
            if (!open) setOpen(true);
            move(event.key === "ArrowDown" ? 1 : -1);
            return;
          }
          if ((event.key === "Enter" || event.key === " ") && open && options[activeIndex]) {
            event.preventDefault();
            choose(options[activeIndex]);
          }
        }}
      >
        <span>{selected?.label ?? value}</span>
        <ChevronDown size={14} aria-hidden="true" />
      </button>
      {open && createPortal(
        <div
          ref={menuRef}
          id={`${id}-options`}
          className="custom-select-menu"
          role="listbox"
          aria-label={accessibleLabel}
          style={{ top: position.top, left: position.left, width: position.width }}
        >
          {options.map((option, index) => (
            <button
              type="button"
              role="option"
              aria-selected={option.value === value}
              className={index === activeIndex ? "active" : ""}
              disabled={option.disabled}
              key={option.value}
              onPointerMove={() => setActiveIndex(index)}
              onClick={() => choose(option)}
            >
              <span>{option.label}</span>
              {option.value === value && <Check size={14} aria-hidden="true" />}
            </button>
          ))}
        </div>,
        document.body,
      )}
    </div>
  );
}
