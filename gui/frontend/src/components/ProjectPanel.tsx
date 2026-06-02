import { useState, useEffect, useCallback, lazy, Suspense } from "react";
const EditProjectModal = lazy(() => import("./EditProjectModal"));
import {
  TestProject,
  SyncProject,
  DeployProject,
  GetLogs,
  ExportProjects,
  SpeedTest,
} from "../wailsjs/go/gui/App";
import type { ProjectInfo, ActionState, TestResult, SyncResult, DeployResult, SpeedTestResult } from "../types";
import LogViewer from "./LogViewer";

interface Props {
  project: ProjectInfo;
  onRemove: (name: string) => Promise<{ ok: boolean; error?: string }>;
  onReload: () => Promise<void>;
}

type Tab = "overview" | "logs";

const protocolBadge: Record<string, string> = {
  sftp: "bg-violet-900/60 text-violet-300",
  ftp:  "bg-blue-900/60  text-blue-300",
  ftps: "bg-cyan-900/60  text-cyan-300",
};

export default function ProjectPanel({ project, onRemove, onReload }: Props) {
  const [tab, setTab]         = useState<Tab>("overview");
  const [test, setTest]       = useState<ActionState>({ status: "idle" });
  const [sync, setSync]       = useState<ActionState>({ status: "idle" });
  const [deploy, setDeploy]   = useState<ActionState>({ status: "idle" });
  const [logs, setLogs]       = useState<string[]>([]);
  const [testDetail, setTestDetail]       = useState<TestResult | null>(null);
  const [syncDetail, setSyncDetail]       = useState<SyncResult | null>(null);
  const [deployDetail, setDeployDetail]   = useState<DeployResult | null>(null);
  const [speed, setSpeed]                 = useState<ActionState>({ status: "idle" });
  const [speedDetail, setSpeedDetail]     = useState<SpeedTestResult | null>(null);
  const [confirmRemove, setConfirmRemove] = useState(false);
  const [showEdit, setShowEdit]           = useState(false);

  const fetchLogs = useCallback(async () => {
    const lines = await GetLogs(project.name, 100);
    setLogs(lines ?? []);
  }, [project.name]);

  useEffect(() => {
    setTest({ status: "idle" });
    setSync({ status: "idle" });
    setDeploy({ status: "idle" });
    setTestDetail(null);
    setSyncDetail(null);
    setDeployDetail(null);
    setSpeed({ status: "idle" });
    setSpeedDetail(null);
    setConfirmRemove(false);
    setTab("overview");
    fetchLogs();
  }, [project.name, fetchLogs]);

  const handleTest = async () => {
    setTest({ status: "loading" });
    setTestDetail(null);
    const res = await TestProject(project.name);
    setTestDetail(res);
    setTest(res.ok
      ? { status: "success", message: `${res.latency_ms}ms` }
      : { status: "error",   message: res.error });
    await onReload();
  };

  const handleSync = async (dryRun = false) => {
    setSync({ status: "loading" });
    setSyncDetail(null);
    const res = await SyncProject(project.name, dryRun);
    setSyncDetail(res);
    setSync(res.ok
      ? { status: "success", message: dryRun
            ? `dry: +${res.added} ~${res.updated} =${res.skipped}`
            : `+${res.added} ~${res.updated} =${res.skipped} · ${fmtBytes(res.bytes)}` }
      : { status: "error", message: res.error });
    if (!dryRun) fetchLogs();
  };

  const handleDeploy = async () => {
    setDeploy({ status: "loading" });
    setDeployDetail(null);
    const res = await DeployProject(project.name);
    setDeployDetail(res);
    setDeploy(res.ok
      ? { status: "success", message: `+${res.added} ~${res.updated} · ${res.duration_ms}ms` }
      : { status: "error",   message: res.error });
    fetchLogs();
  };

  const handleSpeedTest = async () => {
    setSpeed({ status: "loading" });
    setSpeedDetail(null);
    const res = await SpeedTest(project.name);
    setSpeedDetail(res);
    setSpeed(res.ok
      ? { status: "success", message: `↑ ${res.upload_mbps} Mbps  ↓ ${res.download_mbps} Mbps` }
      : { status: "error", message: res.error });
  };

  const handleRemove = async () => {
    if (!confirmRemove) { setConfirmRemove(true); return; }
    await onRemove(project.name);
  };

  const connStatus =
    test.status === "success" ? "ok"  :
    test.status === "error"   ? "err" : "idle";

  return (
    <div className="flex flex-col h-full" style={{ backgroundColor: "var(--color-bg)" }}>

      {/* ── header ── */}
      <header
        className="shrink-0 flex items-center gap-3 px-5 py-3 border-b"
        style={{ borderColor: "var(--color-border)", backgroundColor: "var(--color-surface)" }}
      >
        {/* connection status dot */}
        <span className={[
          "w-2 h-2 rounded-full shrink-0 transition-colors",
          connStatus === "ok"  ? "bg-emerald-400 shadow-[0_0_8px_rgba(52,211,153,0.6)]" :
          connStatus === "err" ? "bg-red-400" :
          "bg-gray-600",
        ].join(" ")} />

        {/* name + badges */}
        <div className="flex items-center gap-2 min-w-0">
          <h1 className="text-sm font-bold text-[var(--color-text)] truncate">
            {project.name}
          </h1>
          <span className={`text-[10px] px-1.5 py-px rounded font-mono shrink-0 ${
            protocolBadge[project.protocol] ?? "bg-gray-700 text-gray-300"
          }`}>
            {project.protocol.toUpperCase()}
          </span>
          {project.env && (
            <span className="text-[10px] px-1.5 py-px rounded bg-gray-800 text-gray-400 shrink-0">
              {project.env}
            </span>
          )}
        </div>

        {/* host info */}
        <span className="text-xs text-[var(--color-muted)] truncate flex-1 hidden sm:block">
          {project.user}@{project.host}:{project.port}
        </span>

        {/* tabs + edit */}
        <div className="flex items-center gap-0.5 shrink-0">
          <TabBtn active={tab === "overview"} onClick={() => setTab("overview")}>
            Overview
          </TabBtn>
          <TabBtn active={tab === "logs"} onClick={() => { setTab("logs"); fetchLogs(); }}>
            Logs
          </TabBtn>
          <div className="w-px h-4 mx-2 shrink-0" style={{ backgroundColor: "var(--color-border)" }} />
          <button
            onClick={() => setShowEdit(true)}
            title="Edit project"
            className="w-7 h-7 flex items-center justify-center rounded text-[var(--color-muted)] hover:text-[var(--color-text)] hover:bg-white/5 transition-colors cursor-pointer text-base"
          >
            ✎
          </button>
        </div>
      </header>

      {showEdit && (
        <Suspense fallback={null}>
          <EditProjectModal
            project={project}
            onClose={() => setShowEdit(false)}
            onSaved={onReload}
          />
        </Suspense>
      )}

      {/* ── body ── */}
      <div className="flex-1 overflow-y-auto">
        {tab === "overview" ? (
          <Overview
            project={project}
            test={test}         testDetail={testDetail}
            sync={sync}         syncDetail={syncDetail}
            deploy={deploy}     deployDetail={deployDetail}
            speed={speed}       speedDetail={speedDetail}
            confirmRemove={confirmRemove}
            onTest={handleTest}
            onSync={handleSync}
            onDeploy={handleDeploy}
            onSpeedTest={handleSpeedTest}
            onRemove={handleRemove}
            onCancelRemove={() => setConfirmRemove(false)}
          />
        ) : (
          <div className="p-4 flex flex-col gap-2 h-full">
            <div className="flex items-center justify-between">
              <span className="text-xs text-[var(--color-muted)]">Last 100 lines</span>
              <button
                onClick={fetchLogs}
                className="text-xs text-[var(--color-accent)] hover:underline cursor-pointer"
              >
                Refresh
              </button>
            </div>
            <LogViewer lines={logs} maxHeight="calc(100vh - 8rem)" />
          </div>
        )}
      </div>
    </div>
  );
}

// ── sub-components ─────────────────────────────────────────────────────────────

function TabBtn({
  active, onClick, children,
}: { active: boolean; onClick: () => void; children: React.ReactNode }) {
  return (
    <button
      onClick={onClick}
      className={[
        "px-3 py-1 rounded text-xs font-medium transition-colors cursor-pointer",
        active
          ? "bg-[var(--color-accent)] text-white"
          : "text-[var(--color-subtext)] hover:text-[var(--color-text)] hover:bg-white/5",
      ].join(" ")}
    >
      {children}
    </button>
  );
}

// ── overview ───────────────────────────────────────────────────────────────────

interface OverviewProps {
  project: ProjectInfo;
  test: ActionState;      testDetail: TestResult | null;
  sync: ActionState;      syncDetail: SyncResult | null;
  deploy: ActionState;    deployDetail: DeployResult | null;
  speed: ActionState;     speedDetail: SpeedTestResult | null;
  confirmRemove: boolean;
  onTest: () => void;
  onSync: (dryRun?: boolean) => void;
  onDeploy: () => void;
  onSpeedTest: () => void;
  onRemove: () => void;
  onCancelRemove: () => void;
}

function Overview({
  project, test, testDetail, sync, syncDetail, deploy, deployDetail,
  speed, speedDetail, confirmRemove, onTest, onSync, onDeploy, onSpeedTest, onRemove, onCancelRemove,
}: OverviewProps) {
  return (
    <div className="p-5 flex flex-col gap-5 max-w-2xl">

      {/* ── Connection strip ── */}
      <div>
        <SectionLabel>Connection</SectionLabel>
        <div
          className="mt-2 rounded-lg p-3 flex flex-wrap gap-x-5 gap-y-2"
          style={{ backgroundColor: "var(--color-surface)", border: "1px solid var(--color-border)" }}
        >
          <InfoPill label="Host"   value={project.host} />
          <InfoPill label="Port"   value={String(project.port)} />
          <InfoPill label="User"   value={project.user} />
          {project.local  && <InfoPill label="Local"  value={project.local}  mono />}
          <InfoPill label="Remote" value={project.remote} mono />
          {project.key    && <InfoPill label="Key"    value={project.key}    mono />}
        </div>
      </div>

      {/* ── Actions ── */}
      <div>
        <SectionLabel>Actions</SectionLabel>
        <div
          className="mt-2 rounded-lg overflow-hidden"
          style={{ border: "1px solid var(--color-border)" }}
        >
          {/* Test */}
          <ActionRow
            label="Test Connection"
            hint="Verify connectivity and measure latency"
            state={test}
            onPrimary={onTest}
            primaryLabel="Test"
          >
            {testDetail?.ok && <Chip color="success">{testDetail.latency_ms}ms</Chip>}
          </ActionRow>

          <RowDivider />

          {/* Sync */}
          <ActionRow
            label="Sync"
            hint="Upload changed files to remote"
            state={sync}
            onPrimary={() => onSync(false)}
            primaryLabel="↑ Sync"
            onSecondary={() => onSync(true)}
            secondaryLabel="Dry Run"
          >
            {syncDetail?.ok && (
              <>
                <Chip color="success">+{syncDetail.added}</Chip>
                <Chip color="warn">~{syncDetail.updated}</Chip>
                <Chip color="muted">={syncDetail.skipped}</Chip>
                <Chip color="muted">{fmtBytes(syncDetail.bytes)}</Chip>
              </>
            )}
          </ActionRow>

          <RowDivider />

          {/* Speed Test */}
          <ActionRow
            label="Speed Test"
            hint="Upload + download 1 MB to measure transfer speed"
            state={speed}
            onPrimary={onSpeedTest}
            primaryLabel="Run"
          >
            {speedDetail?.ok && (
              <>
                <Chip color="success">↑ {speedDetail.upload_mbps} Mbps</Chip>
                <Chip color="info">↓ {speedDetail.download_mbps} Mbps</Chip>
                <Chip color="muted">{fmtBytes(speedDetail.size_bytes)} · {speedDetail.upload_ms + speedDetail.download_ms}ms</Chip>
              </>
            )}
          </ActionRow>

          <RowDivider />

          {/* Deploy — visually prominent */}
          <div className="p-3" style={{ backgroundColor: "rgba(124,58,237,0.05)" }}>
            <div className="flex items-center gap-2">
              <div className="flex-1 min-w-0">
                <div className="text-xs font-semibold text-[var(--color-text)]">Deploy</div>
                <div className="text-[11px] text-[var(--color-muted)]">
                  pre-hooks → sync → post-hooks
                </div>
              </div>
              <Btn variant="primary" onClick={onDeploy} disabled={deploy.status === "loading"}>
                {deploy.status === "loading" ? "Deploying…" : "Deploy"}
              </Btn>
            </div>
            {deploy.status !== "idle" && (
              <div className="mt-2 flex flex-wrap items-center gap-1.5">
                {deploy.status === "loading" && (
                  <span className="text-[11px] text-[var(--color-muted)] animate-pulse">Running…</span>
                )}
                {deploy.status === "success" && (
                  <span className="text-[11px] text-[var(--color-success)]">✓ {deploy.message}</span>
                )}
                {deploy.status === "error" && (
                  <span className="text-[11px] text-[var(--color-error)]">✗ {deploy.message}</span>
                )}
                {(deployDetail?.hooks_run?.length ?? 0) > 0 &&
                  deployDetail!.hooks_run.map((h, i) => (
                    <span key={i} className="text-[10px] px-1.5 py-0.5 rounded bg-violet-900/40 text-violet-300">
                      {h}
                    </span>
                  ))
                }
              </div>
            )}
          </div>
        </div>
      </div>

      {/* ── Footer: Export + Remove ── */}
      <div className="flex items-center justify-between pt-1 pb-2">
        <div className="flex items-center gap-2">
          <span className="text-[11px] text-[var(--color-muted)]">Export</span>
          <ExportBtn project={project} format="beam"      label="JSON" />
          <ExportBtn project={project} format="filezilla" label="FileZilla" />
          <ExportBtn project={project} format="env"       label=".env" />
        </div>

        <div className="flex items-center gap-2">
          {confirmRemove ? (
            <>
              <span className="text-xs text-[var(--color-error)]">
                Remove "{project.name}"?
              </span>
              <Btn variant="danger" onClick={onRemove}>Confirm</Btn>
              <Btn variant="ghost"  onClick={onCancelRemove}>Cancel</Btn>
            </>
          ) : (
            <button
              onClick={onRemove}
              className="text-xs text-[var(--color-muted)] hover:text-[var(--color-error)] transition-colors cursor-pointer"
            >
              Remove project
            </button>
          )}
        </div>
      </div>
    </div>
  );
}

// ── shared primitives ──────────────────────────────────────────────────────────

function SectionLabel({ children }: { children: React.ReactNode }) {
  return (
    <span className="text-[10px] font-semibold tracking-widest uppercase text-[var(--color-muted)]">
      {children}
    </span>
  );
}

function InfoPill({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="flex items-center gap-1.5">
      <span className="text-[10px] text-[var(--color-muted)]">{label}</span>
      <span className={`text-xs text-[var(--color-text)] ${mono ? "font-mono" : ""}`}>
        {value || "—"}
      </span>
    </div>
  );
}

function RowDivider() {
  return <div className="h-px" style={{ backgroundColor: "var(--color-border)" }} />;
}

interface ActionRowProps {
  label: string;
  hint: string;
  state: ActionState;
  onPrimary: () => void;
  primaryLabel: string;
  onSecondary?: () => void;
  secondaryLabel?: string;
  children?: React.ReactNode;
}

function ActionRow({
  label, hint, state, onPrimary, primaryLabel, onSecondary, secondaryLabel, children,
}: ActionRowProps) {
  const loading = state.status === "loading";
  return (
    <div className="p-3" style={{ backgroundColor: "var(--color-surface)" }}>
      <div className="flex items-center gap-2">
        <div className="flex-1 min-w-0">
          <div className="text-xs font-medium text-[var(--color-text)]">{label}</div>
          <div className="text-[11px] text-[var(--color-muted)]">{hint}</div>
        </div>
        <div className="flex items-center gap-1.5 shrink-0">
          {onSecondary && secondaryLabel && (
            <Btn variant="ghost" onClick={onSecondary} disabled={loading}>
              {secondaryLabel}
            </Btn>
          )}
          <Btn variant="primary" onClick={onPrimary} disabled={loading}>
            {loading ? "…" : primaryLabel}
          </Btn>
        </div>
      </div>
      {state.status !== "idle" && (
        <div className="mt-1.5 flex items-center flex-wrap gap-1.5">
          {state.status === "loading" && (
            <span className="text-[11px] text-[var(--color-muted)] animate-pulse">Running…</span>
          )}
          {state.status === "success" && (
            <span className="text-[11px] text-[var(--color-success)]">✓ {state.message}</span>
          )}
          {state.status === "error" && (
            <span className="text-[11px] text-[var(--color-error)]">✗ {state.message}</span>
          )}
          {children}
        </div>
      )}
    </div>
  );
}

function Chip({ color, children }: { color: "success" | "warn" | "muted" | "error" | "info"; children: React.ReactNode }) {
  const cls = {
    success: "bg-emerald-900/50 text-emerald-300",
    warn:    "bg-yellow-900/50  text-yellow-300",
    muted:   "bg-gray-800       text-gray-400",
    error:   "bg-red-900/50     text-red-300",
    info:    "bg-sky-900/50     text-sky-300",
  }[color];
  return (
    <span className={`text-[10px] px-1.5 py-0.5 rounded font-mono ${cls}`}>
      {children}
    </span>
  );
}

function Btn({
  variant, onClick, disabled, children,
}: { variant: "primary" | "ghost" | "danger"; onClick: () => void; disabled?: boolean; children: React.ReactNode }) {
  const cls = {
    primary: "bg-[var(--color-accent)] text-white hover:bg-[var(--color-accent-light)]",
    ghost:   "bg-transparent border border-[var(--color-border)] text-[var(--color-subtext)] hover:text-[var(--color-text)] hover:border-[var(--color-muted)]",
    danger:  "bg-red-900/40 text-red-300 border border-red-800/60 hover:bg-red-900/70",
  }[variant];
  return (
    <button
      onClick={onClick}
      disabled={disabled}
      className={`px-3 py-1 rounded text-xs font-medium transition-colors cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed ${cls}`}
    >
      {children}
    </button>
  );
}

function ExportBtn({ project, format, label }: { project: ProjectInfo; format: string; label: string }) {
  const [state, setState] = useState<"idle" | "loading" | "error">("idle");

  const handleClick = async () => {
    setState("loading");
    const res = await ExportProjects(project.name, format);
    if (!res.ok) {
      setState("error");
      setTimeout(() => setState("idle"), 2000);
      return;
    }
    const mime =
      format === "env"      ? "text/plain" :
      format === "filezilla"? "application/xml" : "application/json";
    const blob = new Blob([res.content], { type: mime });
    const url  = URL.createObjectURL(blob);
    const a    = document.createElement("a");
    a.href     = url;
    a.download = `${project.name}${res.ext}`;
    a.click();
    URL.revokeObjectURL(url);
    setState("idle");
  };

  return (
    <button
      onClick={handleClick}
      disabled={state === "loading"}
      className="px-2.5 py-1 rounded text-xs border border-[var(--color-border)] text-[var(--color-subtext)] hover:text-[var(--color-text)] hover:border-[var(--color-muted)] disabled:opacity-40 transition-colors cursor-pointer"
    >
      {state === "loading" ? "…" : state === "error" ? "✗" : `↓ ${label}`}
    </button>
  );
}

function fmtBytes(bytes: number): string {
  if (bytes < 1024)           return `${bytes}B`;
  if (bytes < 1024 * 1024)    return `${(bytes / 1024).toFixed(1)}KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)}MB`;
}
