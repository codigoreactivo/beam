import type { ProjectInfo } from "../types";

interface Props {
  projects: ProjectInfo[];
  selected: string | null;
  loading: boolean;
  onSelect: (name: string) => void;
  onAddClick: () => void;
  onImportClick: () => void;
  onExportAll: () => void;
}

const protocolBadge: Record<string, string> = {
  sftp: "bg-violet-900/60 text-violet-300",
  ftp:  "bg-blue-900/60  text-blue-300",
  ftps: "bg-cyan-900/60  text-cyan-300",
};

export default function Sidebar({
  projects, selected, loading, onSelect, onAddClick, onImportClick, onExportAll,
}: Props) {
  return (
    <aside
      className="w-52 shrink-0 flex flex-col border-r"
      style={{ backgroundColor: "var(--color-surface)", borderColor: "var(--color-border)" }}
    >
      {/* logo */}
      <div
        className="flex items-center gap-2 px-3 py-2.5 border-b"
        style={{ borderColor: "var(--color-border)", backgroundColor: "var(--color-accent)" }}
      >
        <span className="text-white font-bold text-sm tracking-wide select-none">⚡ BEAM</span>
      </div>

      {/* project list */}
      <div className="flex-1 overflow-y-auto py-1">
        {loading ? (
          <div className="px-3 py-2 text-[var(--color-muted)] text-xs">Loading…</div>
        ) : projects.length === 0 ? (
          <div className="px-3 py-4 text-center">
            <p className="text-[var(--color-muted)] text-xs">No projects yet</p>
            <p className="text-[var(--color-muted)] text-[11px] mt-0.5">Click + Add Project to start</p>
          </div>
        ) : (
          projects.map((p) => {
            const isActive = p.name === selected;
            return (
              <button
                key={p.name}
                title={p.name}
                onClick={() => onSelect(p.name)}
                className={[
                  "w-full text-left px-3 py-2 flex flex-col gap-0.5 transition-colors cursor-pointer border-l-2",
                  isActive
                    ? "bg-[var(--color-accent)]/15 border-[var(--color-accent)]"
                    : "border-transparent hover:bg-white/5 hover:border-white/10",
                ].join(" ")}
              >
                <span className={`text-xs truncate ${
                  isActive ? "font-semibold text-[var(--color-text)]" : "text-[var(--color-subtext)]"
                }`}>
                  {p.name}
                </span>
                <div className="flex items-center gap-1.5">
                  <span className={`text-[9px] px-1 py-px rounded font-mono ${
                    protocolBadge[p.protocol] ?? "bg-gray-700 text-gray-300"
                  }`}>
                    {p.protocol.toUpperCase()}
                  </span>
                  <span className="text-[10px] text-[var(--color-muted)] truncate">
                    {p.host}
                  </span>
                </div>
              </button>
            );
          })
        )}
      </div>

      {/* footer */}
      <div
        className="p-2.5 border-t flex flex-col gap-1.5"
        style={{ borderColor: "var(--color-border)" }}
      >
        <button
          onClick={onAddClick}
          className="w-full py-1.5 rounded text-xs font-semibold bg-[var(--color-accent)] text-white hover:bg-[var(--color-accent-light)] transition-colors cursor-pointer"
        >
          + Add Project
        </button>
        <div className="flex gap-1.5">
          <button
            onClick={onImportClick}
            className="flex-1 py-1.5 rounded text-[11px] font-medium border border-[var(--color-border)] text-[var(--color-muted)] hover:text-[var(--color-subtext)] hover:border-[var(--color-muted)] transition-colors cursor-pointer"
          >
            ↑ Import
          </button>
          <button
            onClick={onExportAll}
            className="flex-1 py-1.5 rounded text-[11px] font-medium border border-[var(--color-border)] text-[var(--color-muted)] hover:text-[var(--color-subtext)] hover:border-[var(--color-muted)] transition-colors cursor-pointer"
          >
            ↓ Export
          </button>
        </div>
      </div>
    </aside>
  );
}
