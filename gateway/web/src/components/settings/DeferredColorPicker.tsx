import { RotateCcw, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";

interface DeferredColorPickerProps {
  value: string;
  defaultValue: string;
  history?: string[];
  label: string;
  description?: string;
  onApply: (color: string) => void;
}

export function DeferredColorPicker({ value, defaultValue, history = [], label, description, onApply }: DeferredColorPickerProps) {
  const { t } = useTranslation("settings_ui");
  const dialogRef = useRef<HTMLDialogElement>(null);
  const [draft, setDraft] = useState(value);
  const normalizedDraft = normalizeHex(draft);

  useEffect(() => setDraft(value), [value]);

  const open = () => {
    setDraft(value);
    dialogRef.current?.showModal();
  };

  const apply = () => {
    if (!normalizedDraft) return;
    onApply(normalizedDraft);
    dialogRef.current?.close();
  };

  return (
    <div className="deferred-color-picker">
      <button type="button" className="color-picker-trigger" aria-haspopup="dialog" onClick={open}>
        <span className="color-swatch" style={{ backgroundColor: value }} aria-hidden="true" />
        <span className="color-picker-copy"><strong>{label}</strong>{description && <small>{description}</small>}</span>
        <code>{value.toUpperCase()}</code>
      </button>
      <dialog ref={dialogRef} className="deferred-color-dialog" aria-label={label} onCancel={() => setDraft(value)}>
        <div className="color-dialog-head">
          <div><strong>{label}</strong><p>{t("color.deferredHint")}</p></div>
          <button type="button" className="square-button" onClick={() => dialogRef.current?.close()} aria-label={t("common.cancel")}><X size={15} aria-hidden="true" /></button>
        </div>
        <div className="color-dialog-editor">
          <label className="native-color-field">
            <span>{t("color.visualPicker")}</span>
            <input
              type="color"
              value={normalizedDraft || defaultValue}
              onInput={(event) => setDraft((event.target as HTMLInputElement).value)}
              aria-label={t("color.visualPicker")}
            />
          </label>
          <label className="hex-color-field">
            <span>{t("color.hex")}</span>
            <input value={draft} onChange={(event) => setDraft(event.target.value)} maxLength={7} spellCheck={false} />
            {!normalizedDraft && <small role="alert">{t("color.invalid")}</small>}
          </label>
          <div className="color-draft-preview" style={{ backgroundColor: normalizedDraft || defaultValue }} aria-hidden="true" />
        </div>
        <div className="color-dialog-presets">
          <div className="color-dialog-section-head"><strong>{t("color.history")}</strong><span>{history.length}</span></div>
          <div className="color-history" aria-label={t("color.history")}>
            {history.length ? history.map((color) => (
              <button key={color} type="button" className={normalizeHex(draft) === color ? "active" : ""} onClick={() => setDraft(color)} title={color} aria-label={color}>
                <span style={{ backgroundColor: color }} aria-hidden="true" />
              </button>
            )) : <p>{t("color.emptyHistory")}</p>}
          </div>
          <button type="button" className="color-default-button" onClick={() => setDraft(defaultValue)}>
            <RotateCcw size={15} aria-hidden="true" /> {t("color.default")}
          </button>
        </div>
        <div className="color-dialog-actions">
          <button type="button" onClick={() => dialogRef.current?.close()}>{t("common.cancel")}</button>
          <button type="button" className="primary-action" disabled={!normalizedDraft} onClick={apply}>{t("common.apply")}</button>
        </div>
      </dialog>
    </div>
  );
}

function normalizeHex(value: string): string | null {
  const normalized = value.trim().toLowerCase();
  if (/^#[0-9a-f]{6}$/.test(normalized)) return normalized;
  if (/^#[0-9a-f]{3}$/.test(normalized)) return `#${normalized.slice(1).split("").map((digit) => digit + digit).join("")}`;
  return null;
}
