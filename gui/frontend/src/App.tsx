import { useState, useEffect, useCallback } from "react";
import {
  GetProjects,
  AddProject,
  RemoveProject,
  ExportProjects,
} from "./wailsjs/go/gui/App";
import type { ProjectInfo, AddProjectParams } from "./types";
import Sidebar from "./components/Sidebar";
import ProjectPanel from "./components/ProjectPanel";
import AddProjectModal from "./components/AddProjectModal";
import ImportModal from "./components/ImportModal";

export default function App() {
  const [projects, setProjects] = useState<ProjectInfo[]>([]);
  const [selected, setSelected] = useState<string | null>(null);
  const [showAdd, setShowAdd] = useState(false);
  const [showImport, setShowImport] = useState(false);
  const [loading, setLoading] = useState(true);

  const reload = useCallback(async () => {
    try {
      const list = await GetProjects();
      setProjects(list ?? []);
      if (list?.length && !selected) {
        setSelected(list[0].name);
      }
    } finally {
      setLoading(false);
    }
  }, [selected]);

  useEffect(() => {
    reload();
  }, []);

  const handleAdd = async (params: AddProjectParams) => {
    const res = await AddProject(params);
    if (res.ok) {
      setShowAdd(false);
      await reload();
      setSelected(params.name);
    }
    return res;
  };

  const handleExportAll = async () => {
    const res = await ExportProjects("", "beam");
    if (!res.ok) return;
    const blob = new Blob([res.content], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "beam-projects" + res.ext;
    a.click();
    URL.revokeObjectURL(url);
  };

  const handleRemove = async (name: string) => {
    const res = await RemoveProject(name);
    if (res.ok) {
      const next = projects.find((p) => p.name !== name);
      setSelected(next?.name ?? null);
      await reload();
    }
    return res;
  };

  const activeProject = projects.find((p) => p.name === selected) ?? null;

  return (
    <div className="flex h-screen w-screen overflow-hidden">
      {/* sidebar */}
      <Sidebar
        projects={projects}
        selected={selected}
        loading={loading}
        onSelect={setSelected}
        onAddClick={() => setShowAdd(true)}
        onImportClick={() => setShowImport(true)}
        onExportAll={handleExportAll}
      />

      {/* main panel */}
      <div className="flex-1 flex flex-col min-w-0">
        {activeProject ? (
          <ProjectPanel
            project={activeProject}
            onRemove={handleRemove}
            onReload={reload}
          />
        ) : (
          <Empty onAdd={() => setShowAdd(true)} />
        )}
      </div>

      {showAdd && (
        <AddProjectModal onClose={() => setShowAdd(false)} onAdd={handleAdd} />
      )}
      {showImport && (
        <ImportModal
          onClose={() => setShowImport(false)}
          onImported={reload}
        />
      )}
    </div>
  );
}

function Empty({ onAdd }: { onAdd: () => void }) {
  return (
    <div className="flex-1 flex flex-col items-center justify-center gap-4 text-[var(--color-muted)]">
      <div className="text-5xl">⚡</div>
      <p className="text-lg font-semibold text-[var(--color-subtext)]">
        No project selected
      </p>
      <p className="text-sm">Add a project to get started</p>
      <button
        onClick={onAdd}
        className="mt-2 px-4 py-2 rounded bg-[var(--color-accent)] text-white text-sm hover:bg-[var(--color-accent-light)] transition-colors cursor-pointer"
      >
        + Add Project
      </button>
    </div>
  );
}
