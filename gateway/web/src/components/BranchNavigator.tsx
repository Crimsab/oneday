import { Check, ChevronDown, GitBranch, GitFork, Pencil } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import type { TimelineResponse } from "../types";

interface BranchNavigatorProps {
  timeline: TimelineResponse | null;
  busy: boolean;
  onFork: (name: string) => Promise<void>;
  onRename: (branchId: string, name: string) => Promise<void>;
  onCheckout: (branchId: string) => Promise<void>;
}

export function BranchNavigator({ timeline, busy, onFork, onRename, onCheckout }: BranchNavigatorProps) {
  const [selected, setSelected] = useState("");
  const [name, setName] = useState("");
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLElement>(null);
  useEffect(() => setSelected(timeline?.active_branch_id ?? ""), [timeline?.active_branch_id]);
  useEffect(() => {
    if (!open) return;
    const closeOutside = (event: PointerEvent) => {
      if (!rootRef.current?.contains(event.target as Node)) setOpen(false);
    };
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") setOpen(false);
    };
    document.addEventListener("pointerdown", closeOutside);
    document.addEventListener("keydown", closeOnEscape);
    return () => {
      document.removeEventListener("pointerdown", closeOutside);
      document.removeEventListener("keydown", closeOnEscape);
    };
  }, [open]);
  if (!timeline) return null;
  const active = timeline.branches.find((branch) => branch.id === timeline.active_branch_id);
  const target = timeline.branches.find((branch) => branch.id === selected);

  return (
    <section className="rail-block branch-navigator" ref={rootRef}>
      <button type="button" className="rail-title split branch-toggle" aria-expanded={open} aria-controls="branch-menu" onClick={() => setOpen((value) => !value)}>
        <span id="branch-title"><GitBranch size={14} />Story branches</span>
        <small>{timeline.branches.length}</small>
        <ChevronDown size={14} />
      </button>
	  {open && <div className="branch-popover" id="branch-menu" aria-labelledby="branch-title">
      <div className="branch-list" role="list" aria-label="Available story branches">
        {timeline.branches.map((branch) => (
          <button type="button" role="listitem" key={branch.id} className={branch.id === selected ? "selected" : ""} onClick={() => setSelected(branch.id)} disabled={busy}>
            <span>{branch.name}</span>
            <small>Turn {branch.head_turn}</small>
            {branch.id === timeline.active_branch_id && <Check size={13} aria-label="Active branch" />}
          </button>
        ))}
      </div>
      <label className="branch-name"><span>Branch name</span><input value={name} onChange={(event) => setName(event.target.value)} maxLength={60} placeholder={active ? `${active.name} alternate` : "Alternate path"} disabled={busy} /></label>
      <div className="branch-actions">
        <button type="button" disabled={busy || !name.trim() || !timeline.head} onClick={() => void onFork(name.trim()).then(() => setName(""))}><GitFork size={14} />Fork</button>
        <button type="button" title="Rename active branch" disabled={busy || !name.trim() || !target || target.id !== timeline.active_branch_id} onClick={() => target && void onRename(target.id, name.trim()).then(() => setName(""))}><Pencil size={14} />Rename</button>
        <button type="button" className="branch-checkout" title="Switch to selected branch" disabled={busy || !target || target.id === timeline.active_branch_id} onClick={() => target && window.confirm(`Switch to “${target.name}”? Your current branch remains available.`) && void onCheckout(target.id).then(() => setOpen(false))}>Switch</button>
      </div>
	  </div>}
    </section>
  );
}
