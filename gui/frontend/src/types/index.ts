export interface Project {
  name: string;
  protocol: string;
  host: string;
  port: number;
  user: string;
  password?: string;
  key?: string;
  local: string;
  remote: string;
  env?: string;
  workers?: number;
}

export interface ProjectInfo extends Project {
  connected: boolean;
  watching: boolean;
}

export interface AddProjectParams {
  name: string;
  protocol: string;
  host: string;
  port: number;
  user: string;
  password: string;
  key: string;
  local: string;
  remote: string;
  env: string;
}

export interface Result {
  ok: boolean;
  error?: string;
}

export interface TestResult {
  ok: boolean;
  latency_ms: number;
  error?: string;
}

export interface SyncResult {
  ok: boolean;
  added: number;
  updated: number;
  skipped: number;
  bytes: number;
  error?: string;
}

export interface DeployResult {
  ok: boolean;
  added: number;
  updated: number;
  hooks_run: string[];
  duration_ms: number;
  error?: string;
}

export interface RemoteEntry {
  name: string;
  size: number;
  is_dir: boolean;
  mod_time: string;
}

export interface GlobalConfig {
  log_level?: string;
  log_dir?: string;
}

export interface ImportResult {
  ok: boolean;
  imported: number;
  skipped: number;
  projects: string[];
  error?: string;
}

export interface ExportResult {
  ok: boolean;
  content: string;
  ext: string;
  error?: string;
}

export interface SpeedTestResult {
  ok: boolean;
  upload_mbps: number;
  download_mbps: number;
  upload_ms: number;
  download_ms: number;
  size_bytes: number;
  error?: string;
}

export type Status = "idle" | "loading" | "success" | "error";

export interface ActionState {
  status: Status;
  message?: string;
}
