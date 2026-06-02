import { useRef, useState } from "react";
import { ImportProjects } from "../wailsjs/go/gui/App";
import type { ImportResult } from "../types";

interface Props {
  onClose: () => void;
  onImported: () => void;
}

const ACCEPTED = ".json,.xml,.cyberduckprofile,.duck,.env";

export default function ImportModal({ onClose, onImported }: Props) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [dragging, setDragging] = useState(false);
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<ImportResult | null>(null);
  const [overwrite, setOverwrite] = useState(false);

  const process = async (file: File) => {
    setLoading(true);
    setResult(null);
    const content = await file.text();
    const res = await ImportProjects(file.name, content, overwrite);
    setResult(res);
    setLoading(false);
    if (res.ok && res.imported > 0) {
      onImported();
    }
  };

  const onFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) process(file);
    e.target.value = "";
  };

  const onDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setDragging(false);
    const file = e.dataTransfer.files?.[0];
    if (file) process(file);
  };

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center"
      style={{ backgroundColor: "rgba(0,0,0,0.6)" }}
      onClick={(e) => e.target === e.currentTarget && onClose()}
    >
      <div
        className="w-[420px] rounded-xl border shadow-2xl"
        style={{ backgroundColor: "var(--color-surface)", borderColor: "var(--color-border)" }}
      >
        {/* header */}
        <div
          className="flex items-center justify-between px-5 py-3 border-b"
          style={{ borderColor: "var(--color-border)" }}
        >
          <h2 className="text-sm font-bold text-[var(--color-text)]">
            Import Projects
          </h2>
          <button
            onClick={onClose}
            className="text-[var(--color-muted)] hover:text-[var(--color-text)] text-lg leading-none cursor-pointer"
          >
            ×
          </button>
        </div>

        <div className="p-5 flex flex-col gap-4">
          {/* drop zone */}
          <div
            onDragOver={(e) => { e.preventDefault(); setDragging(true); }}
            onDragLeave={() => setDragging(false)}
            onDrop={onDrop}
            onClick={() => inputRef.current?.click()}
            className={[
              "flex flex-col items-center justify-center gap-2 rounded-lg border-2 border-dashed py-8 cursor-pointer transition-colors",
              dragging
                ? "border-[var(--color-accent)] bg-[var(--color-accent)]/10"
                : "border-[var(--color-border)] hover:border-[var(--color-muted)]",
            ].join(" ")}
          >
            <span className="text-2xl">📂</span>
            <p className="text-xs text-[var(--color-subtext)]">
              Drop file here or <span className="text-[var(--color-accent)]">click to browse</span>
            </p>
            <p className="text-[10px] text-[var(--color-muted)]">
              Cyberduck · FileZilla · CoreFTP · Beam JSON · .env
            </p>
            <input
              ref={inputRef}
              type="file"
              accept={ACCEPTED}
              className="hidden"
              onChange={onFileChange}
            />
          </div>

          {/* overwrite toggle */}
          <label className="flex items-center gap-2 cursor-pointer select-none">
            <input
              type="checkbox"
              checked={overwrite}
              onChange={(e) => setOverwrite(e.target.checked)}
              className="accent-[var(--color-accent)]"
            />
            <span className="text-xs text-[var(--color-subtext)]">
              Replace existing projects with same name
            </span>
          </label>

          {/* result */}
          {loading && (
            <p className="text-xs text-[var(--color-muted)]">Importing…</p>
          )}
          {result && (
            <div
              className={`rounded-lg p-3 text-xs border ${
                result.ok
                  ? "border-emerald-800/50 bg-emerald-900/20"
                  : "border-red-800/50 bg-red-900/20"
              }`}
            >
              {result.ok ? (
                <>
                  <p className="font-semibold text-[var(--color-success)] mb-1">
                    ✓ {result.imported} imported
                    {result.skipped > 0 && `, ${result.skipped} skipped`}
                  </p>
                  {result.projects?.map((n) => (
                    <p key={n} className="text-[var(--color-subtext)] pl-2">
                      · {n}
                    </p>
                  ))}
                  {result.projects?.some((_, i) => !result.projects[i]) && (
                    <p className="text-[var(--color-warn)] mt-1">
                      ⚠ Set local path: project → edit
                    </p>
                  )}
                </>
              ) : (
                <p className="text-[var(--color-error)]">✗ {result.error}</p>
              )}
            </div>
          )}

          <div className="flex justify-end">
            <button
              onClick={onClose}
              className="px-4 py-1.5 rounded text-xs font-medium border border-[var(--color-border)] text-[var(--color-subtext)] hover:text-[var(--color-text)] transition-colors cursor-pointer"
            >
              {result?.ok ? "Close" : "Cancel"}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
