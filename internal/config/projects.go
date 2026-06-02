package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Protocol string

const (
	ProtocolSFTP  Protocol = "sftp"
	ProtocolFTP   Protocol = "ftp"
	ProtocolFTPS  Protocol = "ftps"
)

type HookAction struct {
	Type     string `yaml:"type"              json:"type"`
	Cmd      string `yaml:"cmd,omitempty"     json:"cmd,omitempty"`
	URL      string `yaml:"url,omitempty"     json:"url,omitempty"`
	Method   string `yaml:"method,omitempty"  json:"method,omitempty"`
	Body     string `yaml:"body,omitempty"    json:"body,omitempty"`
	Title    string `yaml:"title,omitempty"   json:"title,omitempty"`
	Message  string `yaml:"message,omitempty" json:"message,omitempty"`
	Blocking bool   `yaml:"blocking,omitempty" json:"blocking,omitempty"`
}

type Hooks struct {
	PreUpload  []HookAction `yaml:"pre-upload,omitempty"  json:"pre_upload,omitempty"`
	PostUpload []HookAction `yaml:"post-upload,omitempty" json:"post_upload,omitempty"`
	PreDeploy  []HookAction `yaml:"pre-deploy,omitempty"  json:"pre_deploy,omitempty"`
	PostDeploy []HookAction `yaml:"post-deploy,omitempty" json:"post_deploy,omitempty"`
	OnConnect  []HookAction `yaml:"on-connect,omitempty"  json:"on_connect,omitempty"`
	OnError    []HookAction `yaml:"on-error,omitempty"    json:"on_error,omitempty"`
}

type Project struct {
	Name     string   `yaml:"name"              json:"name"`
	Protocol Protocol `yaml:"protocol"          json:"protocol"`
	Host     string   `yaml:"host"              json:"host"`
	Port     int      `yaml:"port"              json:"port"`
	User     string   `yaml:"user"              json:"user"`
	Password string   `yaml:"password,omitempty" json:"password,omitempty"`
	Key      string   `yaml:"key,omitempty"     json:"key,omitempty"`
	Local    string   `yaml:"local"             json:"local"`
	Remote   string   `yaml:"remote"            json:"remote"`
	Env      string   `yaml:"env,omitempty"     json:"env,omitempty"`
	Workers  int      `yaml:"workers,omitempty" json:"workers,omitempty"`
	Hooks    Hooks    `yaml:"hooks,omitempty"   json:"hooks,omitempty"`
}

func (p *Project) DefaultPort() int {
	switch p.Protocol {
	case ProtocolFTPS:
		return 990
	case ProtocolFTP:
		return 21
	default:
		return 22
	}
}

type ProjectsFile struct {
	Projects []Project `yaml:"projects"`
}

func projectsPath() string {
	return filepath.Join(BeamDir(), "projects.yaml")
}

func LoadProjects() (*ProjectsFile, error) {
	data, err := os.ReadFile(projectsPath())
	if os.IsNotExist(err) {
		return &ProjectsFile{}, nil
	}
	if err != nil {
		return nil, err
	}
	var pf ProjectsFile
	return &pf, yaml.Unmarshal(data, &pf)
}

func (pf *ProjectsFile) Save() error {
	if err := os.MkdirAll(BeamDir(), 0o700); err != nil {
		return err
	}
	data, err := yaml.Marshal(pf)
	if err != nil {
		return err
	}
	return os.WriteFile(projectsPath(), data, 0o600)
}

func (pf *ProjectsFile) Find(name string) (*Project, error) {
	for i := range pf.Projects {
		if pf.Projects[i].Name == name {
			return &pf.Projects[i], nil
		}
	}
	return nil, fmt.Errorf("project %q not found", name)
}

func (pf *ProjectsFile) Add(p Project) error {
	for _, existing := range pf.Projects {
		if existing.Name == p.Name {
			return fmt.Errorf("project %q already exists", p.Name)
		}
	}
	pf.Projects = append(pf.Projects, p)
	return nil
}

func (pf *ProjectsFile) Remove(name string) error {
	for i, p := range pf.Projects {
		if p.Name == name {
			pf.Projects = append(pf.Projects[:i], pf.Projects[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("project %q not found", name)
}
