import { useState } from "react";
import type { AddProjectParams, Result } from "../types";

interface Props {
  onClose: () => void;
  onAdd: (params: AddProjectParams) => Promise<Result>;
}

const defaultParams: AddProjectParams = {
  name: "",
  protocol: "sftp",
  host: "",
  port: 0,
  user: "",
  password: "",
  key: "",
  local: "",
  remote: "/",
  env: "",
};

export default function AddProjectModal({ onClose, onAdd }: Props) {
  const [form, setForm] = useState<AddProjectParams>(defaultParams);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const set = (key: keyof AddProjectParams, value: string | number) => {
    setForm((f) => ({ ...f, [key]: value }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!form.name || !form.host || !form.user) {
      setError("Name, host and user are required.");
      return;
    }
    setError("");
    setLoading(true);
    const res = await onAdd(form);
    setLoading(false);
    if (!res.ok) {
      setError(res.error ?? "Unknown error");
    }
  };

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center"
      style={{ backgroundColor: "rgba(0,0,0,0.6)" }}
      onClick={(e) => e.target === e.currentTarget && onClose()}
    >
      <div
        className="w-[480px] max-h-[90vh] overflow-y-auto rounded-xl border shadow-2xl"
        style={{
          backgroundColor: "var(--color-surface)",
          borderColor: "var(--color-border)",
        }}
      >
        {/* header */}
        <div
          className="flex items-center justify-between px-5 py-3 border-b"
          style={{ borderColor: "var(--color-border)" }}
        >
          <h2 className="text-sm font-bold text-[var(--color-text)]">
            Add Project
          </h2>
          <button
            onClick={onClose}
            className="text-[var(--color-muted)] hover:text-[var(--color-text)] text-lg leading-none cursor-pointer"
          >
            ×
          </button>
        </div>

        <form onSubmit={handleSubmit} className="p-5 grid gap-4">
          {/* row: name + protocol */}
          <div className="grid grid-cols-2 gap-3">
            <Field label="Name *">
              <Input
                value={form.name}
                onChange={(v) => set("name", v)}
                placeholder="my-server"
                autoFocus
              />
            </Field>
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
          </div>

          {/* row: host + port */}
          <div className="grid grid-cols-[1fr_80px] gap-3">
            <Field label="Host *">
              <Input
                value={form.host}
                onChange={(v) => set("host", v)}
                placeholder="example.com"
              />
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

          {/* row: user + password */}
          <div className="grid grid-cols-2 gap-3">
            <Field label="User *">
              <Input
                value={form.user}
                onChange={(v) => set("user", v)}
                placeholder="deploy"
              />
            </Field>
            <Field label="Password">
              <Input
                value={form.password}
                onChange={(v) => set("password", v)}
                placeholder="(optional)"
                type="password"
              />
            </Field>
          </div>

          {/* SSH key (SFTP only) */}
          {form.protocol === "sftp" && (
            <Field label="SSH Key Path">
              <Input
                value={form.key}
                onChange={(v) => set("key", v)}
                placeholder="~/.ssh/id_ed25519"
              />
            </Field>
          )}

          {/* paths */}
          <Field label="Local Path *">
            <Input
              value={form.local}
              onChange={(v) => set("local", v)}
              placeholder="/home/user/project"
            />
          </Field>
          <Field label="Remote Path *">
            <Input
              value={form.remote}
              onChange={(v) => set("remote", v)}
              placeholder="/var/www/html"
            />
          </Field>

          {/* env */}
          <Field label="Env File">
            <Input
              value={form.env}
              onChange={(v) => set("env", v)}
              placeholder=".env (optional)"
            />
          </Field>

          {error && (
            <p className="text-xs text-[var(--color-error)]">{error}</p>
          )}

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
              {loading ? "Adding…" : "Add Project"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

function Field({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <label className="flex flex-col gap-1">
      <span className="text-[11px] text-[var(--color-muted)]">{label}</span>
      {children}
    </label>
  );
}

function Input({
  value,
  onChange,
  placeholder,
  type = "text",
  autoFocus,
}: {
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
  type?: string;
  autoFocus?: boolean;
}) {
  return (
    <input
      type={type}
      value={value}
      onChange={(e) => onChange(e.target.value)}
      placeholder={placeholder}
      autoFocus={autoFocus}
      className="w-full bg-[var(--color-bg)] border border-[var(--color-border)] rounded px-2 py-1.5 text-xs text-[var(--color-text)] placeholder-[var(--color-muted)] focus:outline-none focus:border-[var(--color-accent)]"
    />
  );
}
