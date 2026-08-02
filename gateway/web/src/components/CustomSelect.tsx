import { Check, ChevronDown } from "lucide-react";
import { createPortal } from "react-dom";
import { useEffect, useId, useLayoutEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";

export interface SelectOption {
  value: string;
  label: string;
  iconSrc?: string;
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
  const placementFrameRef = useRef<number | null>(null);
  const typeaheadRef = useRef("");
  const typeaheadTimerRef = useRef<number | null>(null);
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
    const next = {
      top,
      left: Math.min(Math.max(8, rect.left), Math.max(8, window.innerWidth - rect.width - 8)),
      width: rect.width,
    };
    setPosition((current) => current.top === next.top && current.left === next.left && current.width === next.width ? current : next);
  };

  const schedulePlacement = () => {
    if (placementFrameRef.current !== null) return;
    placementFrameRef.current = window.requestAnimationFrame(() => {
      placementFrameRef.current = null;
      placeMenu();
    });
  };

  useLayoutEffect(() => {
    if (!open) return;
    placeMenu();
    return () => {
      if (placementFrameRef.current !== null) window.cancelAnimationFrame(placementFrameRef.current);
      placementFrameRef.current = null;
    };
  }, [open, options.length]);

  useEffect(() => {
    if (!open) return;
    document.getElementById(`${id}-option-${activeIndex}`)?.scrollIntoView({ block: "nearest" });
  }, [activeIndex, id, open]);

  useEffect(() => {
    if (!open) return;
    const closeOutside = (event: PointerEvent) => {
      const target = event.target as Node;
      if (!triggerRef.current?.contains(target) && !menuRef.current?.contains(target)) setOpen(false);
    };
    window.addEventListener("pointerdown", closeOutside, true);
    window.addEventListener("resize", schedulePlacement);
    window.addEventListener("scroll", schedulePlacement, { capture: true, passive: true });
    return () => {
      window.removeEventListener("pointerdown", closeOutside, true);
      window.removeEventListener("resize", schedulePlacement);
      window.removeEventListener("scroll", schedulePlacement, true);
      if (placementFrameRef.current !== null) window.cancelAnimationFrame(placementFrameRef.current);
      placementFrameRef.current = null;
    };
  }, [open]);

  useEffect(() => () => {
    if (typeaheadTimerRef.current !== null) window.clearTimeout(typeaheadTimerRef.current);
  }, []);

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

  const moveToEdge = (edge: "start" | "end") => {
    const indexes = options.map((_, index) => index);
    if (edge === "end") indexes.reverse();
    const next = indexes.find((index) => !options[index]?.disabled);
    if (next !== undefined) setActiveIndex(next);
  };

  const typeahead = (key: string) => {
    typeaheadRef.current += key.toLocaleLowerCase();
    if (typeaheadTimerRef.current !== null) window.clearTimeout(typeaheadTimerRef.current);
    typeaheadTimerRef.current = window.setTimeout(() => { typeaheadRef.current = ""; }, 600);
    const query = typeaheadRef.current;
    const next = options.findIndex((option) => !option.disabled && option.label.toLocaleLowerCase().startsWith(query));
    if (next >= 0) setActiveIndex(next);
  };

  const choose = (option: SelectOption) => {
    if (option.disabled) return;
    onChange(option.value);
    setOpen(false);
    window.requestAnimationFrame(() => triggerRef.current?.focus());
  };

  const optionLabel = (option: SelectOption | undefined) => (
    <span className="custom-select-option-label">
      {option?.iconSrc && <img src={option.iconSrc} alt="" aria-hidden="true" />}
      <span>{option?.label ?? value}</span>
    </span>
  );

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
        aria-activedescendant={open && options[activeIndex] ? `${id}-option-${activeIndex}` : undefined}
        disabled={disabled || options.length === 0}
        onClick={() => {
          setOpen((current) => !current);
        }}
        onKeyDown={(event) => {
          if (event.key === "Escape" && open) {
            event.preventDefault();
            event.stopPropagation();
            setOpen(false);
            return;
          }
          if (event.key === "ArrowDown" || event.key === "ArrowUp") {
            event.preventDefault();
            if (!open) setOpen(true);
            move(event.key === "ArrowDown" ? 1 : -1);
            return;
          }
          if (event.key === "Home" || event.key === "End") {
            event.preventDefault();
            if (!open) setOpen(true);
            moveToEdge(event.key === "Home" ? "start" : "end");
            return;
          }
          if ((event.key === "Enter" || event.key === " ") && open && options[activeIndex]) {
            event.preventDefault();
            choose(options[activeIndex]);
            return;
          }
          if (event.key === "Tab" && open) setOpen(false);
          if (event.key.length === 1 && !event.ctrlKey && !event.metaKey && !event.altKey && event.key !== " ") {
            if (!open) setOpen(true);
            typeahead(event.key);
          }
        }}
      >
        {optionLabel(selected)}
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
              id={`${id}-option-${index}`}
              role="option"
              aria-selected={option.value === value}
              className={index === activeIndex ? "active" : ""}
              disabled={option.disabled}
              key={option.value}
              onPointerMove={() => {
                if (activeIndex !== index) setActiveIndex(index);
              }}
              onClick={() => choose(option)}
            >
              {optionLabel(option)}
              {option.value === value && <Check size={14} aria-hidden="true" />}
            </button>
          ))}
        </div>,
        document.body,
      )}
    </div>
  );
}
