import { useState, useEffect } from "react";
import { UpdateProject, GetProjects } from "../wailsjs/go/gui/App";
import type { ProjectInfo, AddProjectParams, Result } from "../types";

interface Props {
  project: ProjectInfo;
  onClose: () => void;
  onSaved: () => void;
}

export default function EditProjectModal({ project, onClose, onSaved }: Props) {
  const [form, setForm] = useState<AddProjectParams>({
    name: project.name,
    protocol: project.protocol,
    host: project.host,
    port: project.port,
    user: project.user === "Default Username" ? "" : project.user,
    password: "",
    key: project.key ?? "",
    local: project.local ?? "",
    remote: project.remote ?? "/",
    env: project.env ?? "",
  });
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [showPass, setShowPass] = useState(false);

  // fetch current password from keychain via GetProjects (it backfills)
  useEffect(() => {
    GetProjects().then((list) => {
      const p = list?.find((x) => x.name === project.name);
      if (p && (p as any).password) {
        setForm((f) => ({ ...f, password: (p as any).password ?? "" }));
      }
    });
  }, [project.name]);

  const set = (key: keyof AddProjectParams, value: string | number) =>
    setForm((f) => ({ ...f, [key]: value }));

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!form.host || !form.user) {
      setError("Host and user are required.");
      return;
    }
    setError("");
    setLoading(true);
    const res: Result = await UpdateProject(form);
    setLoading(false);
    if (res.ok) {
      onSaved();
      onClose();
    } else {
      setError(res.error ?? "Unknown error");
    }
  };

  const missingCreds = !form.password && !form.key;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center"
      style={{ backgroundColor: "rgba(0,0,0,0.6)" }}
      onClick={(e) => e.target === e.currentTarget && onClose()}
    >
      <div
        className="w-[480px] max-h-[90vh] overflow-y-auto rounded-xl border shadow-2xl"
        style={{ backgroundColor: "var(--color-surface)", borderColor: "var(--color-border)" }}
      >
        {/* header */}
        <div
          className="flex items-center justify-between px-5 py-3 border-b"
          style={{ borderColor: "var(--color-border)" }}
        >
          <h2 className="text-sm font-bold text-[var(--color-text)]">
            Edit — {project.name}
          </h2>
          <button
            onClick={onClose}
            className="text-[var(--color-muted)] hover:text-[var(--color-text)] text-lg leading-none cursor-pointer"
          >
            ×
          </button>
        </div>

        <form onSubmit={handleSubmit} className="p-5 grid gap-4">
          {/* credentials banner */}
          {missingCreds && (
            <div className="rounded-lg border border-yellow-800/50 bg-yellow-900/20 px-3 py-2 text-xs text-yellow-300">
              ⚠ No credentials set — add a password or SSH key to connect.
            </div>
          )}

          {/* host + port */}
          <div className="grid grid-cols-[1fr_80px] gap-3">
            <Field label="Host *">
              <Input value={form.host} onChange={(v) => set("host", v)} placeholder="example.com" />
            </Field>
            <Field label="Port">
              <Input
                value={form.port === 0 ? "" : String(form.port)}
                onChange={(v) => set("port", parseInt(v) || 0)}
                placeholder="auto"
                type="number"
              />
            </Field>
          </div>

          {/* protocol + user */}
          <div className="grid grid-cols-2 gap-3">
            <Field label="Protocol">
              <div className="relative">
                <select
                  value={form.protocol}
                  onChange={(e) => set("protocol", e.target.value)}
                  className="w-full appearance-none bg-[var(--color-bg)] border border-[var(--color-border)] rounded px-2 py-1.5 text-xs text-[var(--color-text)] focus:outline-none focus:border-[var(--color-accent)] cursor-pointer pr-6"
                >
                  <option value="sftp">SFTP</option>
                  <option value="ftp">FTP</option>
                  <option value="ftps">FTPS</option>
                </select>
                <span className="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 text-[var(--color-muted)] text-[10px]">▾</span>
              </div>
            </Field>
            <Field label="User *">
              <Input value={form.user} onChange={(v) => set("user", v)} placeholder="username" />
            </Field>
          </div>

          {/* password */}
          <Field label="Password">
            <div className="relative">
              <input
                type={showPass ? "text" : "password"}
                value={form.password}
                onChange={(e) => set("password", e.target.value)}
                placeholder="(stored in OS keychain)"
                className="w-full bg-[var(--color-bg)] border border-[var(--color-border)] rounded px-2 py-1.5 text-xs text-[var(--color-text)] placeholder-[var(--color-muted)] focus:outline-none focus:border-[var(--color-accent)] pr-14"
              />
              <button
                type="button"
                onClick={() => setShowPass((v) => !v)}
                className="absolute right-2 top-1/2 -translate-y-1/2 text-[10px] text-[var(--color-muted)] hover:text-[var(--color-subtext)] cursor-pointer"
              >
                {showPass ? "hide" : "show"}
              </button>
            </div>
          </Field>

          {/* ssh key (sftp only) */}
          {form.protocol === "sftp" && (
            <Field label="SSH Key Path">
              <Input value={form.key} onChange={(v) => set("key", v)} placeholder="~/.ssh/id_rsa" />
            </Field>
          )}

          {/* paths */}
          <Field label="Local Path *">
            <Input value={form.local} onChange={(v) => set("local", v)} placeholder="/home/user/project" />
          </Field>
          <Field label="Remote Path *">
            <Input value={form.remote} onChange={(v) => set("remote", v)} placeholder="/public_html" />
          </Field>

          {/* env */}
          <Field label="Env Label">
            <Input value={form.env} onChange={(v) => set("env", v)} placeholder="production (optional)" />
          </Field>

          {error && <p className="text-xs text-[var(--color-error)]">{error}</p>}

          <div className="flex justify-end gap-2 pt-1">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-1.5 rounded text-xs font-medium border border-[var(--color-border)] text-[var(--color-subtext)] hover:text-[var(--color-text)] transition-colors cursor-pointer"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={loading}
              className="px-4 py-1.5 rounded text-xs font-medium bg-[var(--color-accent)] text-white hover:bg-[var(--color-accent-light)] disabled:opacity-40 transition-colors cursor-pointer"
            >
              {loading ? "Saving…" : "Save"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="flex flex-col gap-1">
      <span className="text-[11px] text-[var(--color-muted)]">{label}</span>
      {children}
    </label>
  );
}

function Input({
  value, onChange, placeholder, type = "text",
}: {
  value: string; onChange: (v: string) => void; placeholder?: string; type?: string;
}) {
  return (
    <input
      type={type}
      value={value}
      onChange={(e) => onChange(e.target.value)}
      placeholder={placeholder}
      className="w-full bg-[var(--color-bg)] border border-[var(--color-border)] rounded px-2 py-1.5 text-xs text-[var(--color-text)] placeholder-[var(--color-muted)] focus:outline-none focus:border-[var(--color-accent)]"
    />
  );
}
