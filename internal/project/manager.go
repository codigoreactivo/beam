package project

import (
	"fmt"

	"github.com/codigoreactivo/beam/internal/config"
	"github.com/codigoreactivo/beam/internal/credentials"
)

type Manager struct {
	cfg      *config.GlobalConfig
	projects *config.ProjectsFile
}

func NewManager(cfg *config.GlobalConfig) (*Manager, error) {
	projects, err := config.LoadProjects()
	if err != nil {
		return nil, fmt.Errorf("load projects: %w", err)
	}
	return &Manager{cfg: cfg, projects: projects}, nil
}

func (m *Manager) All() []config.Project {
	result := make([]config.Project, len(m.projects.Projects))
	for i, p := range m.projects.Projects {
		if p.Password == "" {
			if pass, err := credentials.Get(p.Name); err == nil {
				p.Password = pass
			}
		}
		result[i] = p
	}
	return result
}

func (m *Manager) Get(name string) (*config.Project, error) {
	p, err := m.projects.Find(name)
	if err != nil {
		return nil, err
	}
	// Backfill password from keychain when not stored in YAML.
	if p.Password == "" {
		if pass, err := credentials.Get(name); err == nil {
			p.Password = pass
		}
	}
	return p, nil
}

func (m *Manager) Add(p config.Project) error {
	if p.Port == 0 {
		p.Port = p.DefaultPort()
	}
	// Store password in the OS keychain; keep YAML credential-free.
	if p.Password != "" {
		if err := credentials.Set(p.Name, p.Password); err == nil {
			p.Password = "" // cleared from YAML on success
		}
		// If keychain fails (e.g. no keychain daemon), fall back to YAML.
	}
	if err := m.projects.Add(p); err != nil {
		return err
	}
	return m.projects.Save()
}

// Update replaces an existing project. The name field identifies the target;
// all other fields are overwritten with the new values.
func (m *Manager) Update(p config.Project) error {
	if p.Port == 0 {
		p.Port = p.DefaultPort()
	}
	if p.Password != "" {
		if err := credentials.Set(p.Name, p.Password); err == nil {
			p.Password = ""
		}
	}
	if err := m.projects.Remove(p.Name); err != nil {
		return err
	}
	if err := m.projects.Add(p); err != nil {
		return err
	}
	return m.projects.Save()
}

func (m *Manager) Remove(name string) error {
	if err := m.projects.Remove(name); err != nil {
		return err
	}
	_ = credentials.Delete(name) // best-effort; ignore keychain errors
	return m.projects.Save()
}
