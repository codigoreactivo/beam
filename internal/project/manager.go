package project

import (
	"fmt"

	"github.com/jesusjhoel/beam/internal/config"
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
	return m.projects.Projects
}

func (m *Manager) Get(name string) (*config.Project, error) {
	return m.projects.Find(name)
}

func (m *Manager) Add(p config.Project) error {
	if p.Port == 0 {
		p.Port = p.DefaultPort()
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
	return m.projects.Save()
}
