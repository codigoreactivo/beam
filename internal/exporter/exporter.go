// Package exporter converts Beam projects to common FTP client formats.
//
// Supported formats:
//   - beam    Beam JSON (default, portable backup)
//   - filezilla  FileZilla sitemanager.xml
//   - env     .env file (FTP_HOST, FTP_USER, ...)
package exporter

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/codigoreactivo/beam/internal/config"
)

// Format constants.
const (
	FormatBeam      = "beam"
	FormatFileZilla = "filezilla"
	FormatEnv       = "env"
)

// Marshal serializes projects to the given format. Returns the file contents.
func Marshal(projects []config.Project, format string) ([]byte, error) {
	switch format {
	case FormatBeam, "json", "":
		return marshalBeam(projects)
	case FormatFileZilla, "fz":
		return marshalFileZilla(projects)
	case FormatEnv, "dotenv":
		return marshalEnv(projects)
	default:
		return nil, fmt.Errorf("unknown format %q (supported: beam, filezilla, env)", format)
	}
}

// Ext returns the recommended file extension for a given format.
func Ext(format string) string {
	switch format {
	case FormatFileZilla, "fz":
		return ".xml"
	case FormatEnv, "dotenv":
		return ".env"
	default:
		return ".json"
	}
}

// ── Beam JSON ────────────────────────────────────────────────────────────────

type beamExport struct {
	Version    int              `json:"version"`
	ExportedAt time.Time        `json:"exported_at"`
	Projects   []config.Project `json:"projects"`
}

func marshalBeam(projects []config.Project) ([]byte, error) {
	// Strip passwords before exporting — credentials live in the keychain.
	safe := make([]config.Project, len(projects))
	for i, p := range projects {
		safe[i] = p
		safe[i].Password = ""
	}
	exp := beamExport{
		Version:    1,
		ExportedAt: time.Now().UTC(),
		Projects:   safe,
	}
	data, err := json.MarshalIndent(exp, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// ── FileZilla sitemanager.xml ─────────────────────────────────────────────────

type fzRoot struct {
	XMLName xml.Name   `xml:"FileZilla3"`
	Version string     `xml:"version,attr"`
	Servers []fzServer `xml:"Servers>Server"`
}

type fzServer struct {
	Host      string `xml:"Host"`
	Port      int    `xml:"Port"`
	Protocol  int    `xml:"Protocol"`
	Logontype int    `xml:"Logontype"`
	User      string `xml:"User,omitempty"`
	Pass      string `xml:"Pass,omitempty"`
	Name      string `xml:"Name"`
	RemoteDir string `xml:"RemoteDir,omitempty"`
	LocalDir  string `xml:"LocalDir,omitempty"`
	Keyfile   string `xml:"Keyfile,omitempty"`
}

func marshalFileZilla(projects []config.Project) ([]byte, error) {
	root := fzRoot{Version: "3"}
	for _, p := range projects {
		s := fzServer{
			Host:      p.Host,
			Port:      p.Port,
			User:      p.User,
			Name:      p.Name,
			RemoteDir: p.Remote,
			LocalDir:  p.Local,
		}
		switch p.Protocol {
		case config.ProtocolSFTP:
			s.Protocol = 1
		case config.ProtocolFTPS:
			s.Protocol = 3
		default:
			s.Protocol = 0
		}
		if p.Key != "" {
			s.Logontype = 4
			s.Keyfile = p.Key
		} else if p.Password != "" {
			s.Logontype = 1
			s.Pass = p.Password
		}
		root.Servers = append(root.Servers, s)
	}

	out, err := xml.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), append(out, '\n')...), nil
}

// ── .env ──────────────────────────────────────────────────────────────────────

func marshalEnv(projects []config.Project) ([]byte, error) {
	if len(projects) > 1 {
		return nil, fmt.Errorf(".env format supports only one project at a time")
	}
	p := projects[0]
	prefix := strings.ToUpper(string(p.Protocol))
	var sb strings.Builder
	fmt.Fprintf(&sb, "# Beam export — %s\n", p.Name)
	fmt.Fprintf(&sb, "%s_HOST=%s\n", prefix, p.Host)
	fmt.Fprintf(&sb, "%s_PORT=%d\n", prefix, p.Port)
	fmt.Fprintf(&sb, "%s_USER=%s\n", prefix, p.User)
	if p.Password != "" {
		fmt.Fprintf(&sb, "%s_PASS=%s\n", prefix, p.Password)
	} else {
		fmt.Fprintf(&sb, "# %s_PASS=\n", prefix)
	}
	if p.Key != "" {
		fmt.Fprintf(&sb, "%s_KEY=%s\n", prefix, p.Key)
	}
	fmt.Fprintf(&sb, "%s_REMOTE=%s\n", prefix, p.Remote)
	if p.Local != "" {
		fmt.Fprintf(&sb, "%s_LOCAL=%s\n", prefix, p.Local)
	}
	return []byte(sb.String()), nil
}
