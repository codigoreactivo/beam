// Package importer parses FTP/SFTP configuration files from common clients
// and converts them to Beam projects.
//
// Supported formats:
//   - Beam JSON  (.json with "version" field)
//   - FileZilla  (sitemanager.xml or any XML with <FileZilla3>)
//   - Cyberduck  (.cyberduckprofile / .duck / Apple plist with <plist>)
//   - CoreFTP    (XML with <SiteManager> or <FTPSite>)
//   - .env       (KEY=VALUE lines: FTP_HOST, FTP_USER, etc.)
package importer

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/codigoreactivo/beam/internal/config"
)

// BeamExport is the canonical Beam JSON format for import/export.
type BeamExport struct {
	Version    int              `json:"version"`
	ExportedAt time.Time        `json:"exported_at"`
	Projects   []config.Project `json:"projects"`
}

// Parse reads the file at path, auto-detects its format, and returns projects.
func Parse(path string) ([]config.Project, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return ParseContent(filepath.Base(path), data)
}

// ParseContent parses project configs from raw bytes.
// filename is used only for format detection (no file I/O).
func ParseContent(filename string, data []byte) ([]config.Project, error) {
	format := detect(filename, data)
	switch format {
	case "beam":
		return parseBeam(data)
	case "filezilla":
		return parseFileZilla(data)
	case "cyberduck":
		return parseCyberduck(data)
	case "coreftp":
		return parseCoreRTP(data)
	case "env":
		return parseEnv(data)
	default:
		return nil, fmt.Errorf("unrecognized format for %q (supported: Beam JSON, FileZilla, Cyberduck, CoreFTP, .env)", filename)
	}
}

// ── format detection ─────────────────────────────────────────────────────────

func detect(path string, data []byte) string {
	ext := strings.ToLower(filepath.Ext(path))
	s := string(data)

	if ext == ".cyberduckprofile" || ext == ".duck" {
		return "cyberduck"
	}
	if strings.Contains(s, "<FileZilla3>") || strings.Contains(s, "<FileZilla3 ") {
		return "filezilla"
	}
	if strings.Contains(s, "<!DOCTYPE plist") || strings.Contains(s, "<plist ") {
		return "cyberduck"
	}
	if strings.Contains(s, "<SiteManager>") || strings.Contains(s, "<FTPSite>") || strings.Contains(s, "<CoreFTP>") {
		return "coreftp"
	}
	if ext == ".json" || (strings.Contains(s, `"version"`) && strings.Contains(s, `"projects"`)) {
		return "beam"
	}
	if ext == ".env" || looksLikeEnv(s) {
		return "env"
	}
	return ""
}

func looksLikeEnv(s string) bool {
	for _, key := range []string{"FTP_HOST", "SFTP_HOST", "FTP_USER", "SFTP_USER"} {
		if strings.Contains(s, key) {
			return true
		}
	}
	return false
}

// ── Beam JSON ────────────────────────────────────────────────────────────────

func parseBeam(data []byte) ([]config.Project, error) {
	var exp BeamExport
	if err := json.Unmarshal(data, &exp); err != nil {
		return nil, fmt.Errorf("beam JSON: %w", err)
	}
	if len(exp.Projects) == 0 {
		return nil, fmt.Errorf("beam JSON: no projects found")
	}
	return exp.Projects, nil
}

// ── FileZilla sitemanager.xml ─────────────────────────────────────────────────

type fzRoot struct {
	XMLName xml.Name   `xml:"FileZilla3"`
	Servers []fzServer `xml:"Servers>Server"`
}

type fzServer struct {
	Host     string `xml:"Host"`
	Port     int    `xml:"Port"`
	Protocol int    `xml:"Protocol"` // 0=FTP 1=SFTP 3=FTPS-explicit 4=FTPS-implicit
	User     string `xml:"User"`
	Pass     struct {
		Encoding string `xml:"encoding,attr"`
		Value    string `xml:",chardata"`
	} `xml:"Pass"`
	Name      string `xml:"Name"`
	RemoteDir string `xml:"RemoteDir"`
	LocalDir  string `xml:"LocalDir"`
	Logontype int    `xml:"Logontype"` // 0=anon 1=normal 4=key
	Keyfile   string `xml:"Keyfile"`
}

func parseFileZilla(data []byte) ([]config.Project, error) {
	var root fzRoot
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("filezilla XML: %w", err)
	}
	var projects []config.Project
	for _, s := range root.Servers {
		p := config.Project{
			Name:   sanitizeName(s.Name),
			Host:   s.Host,
			Port:   s.Port,
			User:   s.User,
			Local:  s.LocalDir,
			Remote: s.RemoteDir,
		}
		switch s.Protocol {
		case 1:
			p.Protocol = config.ProtocolSFTP
		case 3, 4:
			p.Protocol = config.ProtocolFTPS
		default:
			p.Protocol = config.ProtocolFTP
		}
		if p.Port == 0 {
			p.Port = p.DefaultPort()
		}
		if s.Pass.Value != "" {
			pass := s.Pass.Value
			if strings.ToLower(s.Pass.Encoding) == "base64" {
				if dec, err := base64.StdEncoding.DecodeString(pass); err == nil {
					pass = string(dec)
				}
			}
			p.Password = pass
		}
		if s.Keyfile != "" {
			p.Key = s.Keyfile
		}
		projects = append(projects, p)
	}
	if len(projects) == 0 {
		return nil, fmt.Errorf("filezilla: no servers found in file")
	}
	return projects, nil
}

// ── Cyberduck plist (.cyberduckprofile / .duck) ───────────────────────────────

func parseCyberduck(data []byte) ([]config.Project, error) {
	kv, err := parsePlistDict(data)
	if err != nil {
		return nil, fmt.Errorf("cyberduck plist: %w", err)
	}

	p := config.Project{
		Host:   first(kv, "Hostname", "Default Hostname"),
		User:   first(kv, "Username", "Default Username"),
		Remote: first(kv, "Path", "Default Path", "/"),
	}

	name := first(kv, "Nickname", "Default Nickname", "Vendor", p.Host)
	p.Name = sanitizeName(name)

	proto := strings.ToLower(first(kv, "Protocol"))
	switch proto {
	case "sftp":
		p.Protocol = config.ProtocolSFTP
	case "ftps":
		p.Protocol = config.ProtocolFTPS
	default:
		p.Protocol = config.ProtocolFTP
	}

	portStr := first(kv, "Port", "Default Port")
	if n, err := strconv.Atoi(portStr); err == nil && n > 0 {
		p.Port = n
	} else {
		p.Port = p.DefaultPort()
	}

	if p.Host == "" {
		return nil, fmt.Errorf("cyberduck: no hostname found in profile")
	}
	return []config.Project{p}, nil
}

// parsePlistDict extracts a flat map from an Apple plist <dict>.
func parsePlistDict(data []byte) (map[string]string, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	result := map[string]string{}
	var inKey, inValue bool
	var currentKey string

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "key":
				inKey, inValue = true, false
			case "string", "integer", "real":
				inKey, inValue = false, true
			case "true":
				result[currentKey] = "true"
				inKey, inValue = false, false
			case "false":
				result[currentKey] = "false"
				inKey, inValue = false, false
			}
		case xml.EndElement:
			inKey, inValue = false, false
		case xml.CharData:
			text := strings.TrimSpace(string(t))
			if text == "" {
				continue
			}
			if inKey {
				currentKey = text
			} else if inValue && currentKey != "" {
				result[currentKey] = text
				inValue = false
			}
		}
	}
	return result, nil
}

// ── CoreFTP XML ───────────────────────────────────────────────────────────────

type coreFTPRoot struct {
	XMLName xml.Name      `xml:"SiteManager"`
	Sites   []coreFTPSite `xml:"Site"`
}

type coreFTPRootAlt struct {
	XMLName xml.Name      `xml:"CoreFTP"`
	Sites   []coreFTPSite `xml:"Site"`
}

type coreFTPSite struct {
	Name     string `xml:"Name"`
	Host     string `xml:"Host"`
	Server   string `xml:"Server"` // alternate field name
	Port     int    `xml:"Port"`
	Protocol string `xml:"Protocol"`
	User     string `xml:"User"`
	Password string `xml:"Password"`
	Remote   string `xml:"RemoteDir"`
	Local    string `xml:"LocalDir"`
}

func parseCoreRTP(data []byte) ([]config.Project, error) {
	var sites []coreFTPSite

	var root coreFTPRoot
	if err := xml.Unmarshal(data, &root); err == nil && len(root.Sites) > 0 {
		sites = root.Sites
	} else {
		var alt coreFTPRootAlt
		if err := xml.Unmarshal(data, &alt); err == nil {
			sites = alt.Sites
		}
	}

	if len(sites) == 0 {
		// Try single <FTPSite> root element
		var single struct {
			XMLName  xml.Name `xml:"FTPSite"`
			coreFTPSite
		}
		if err := xml.Unmarshal(data, &single); err == nil && single.Host != "" {
			sites = []coreFTPSite{single.coreFTPSite}
		}
	}

	if len(sites) == 0 {
		return nil, fmt.Errorf("coreftp: no sites found")
	}

	var projects []config.Project
	for _, s := range sites {
		host := s.Host
		if host == "" {
			host = s.Server
		}
		p := config.Project{
			Name:     sanitizeName(s.Name),
			Host:     host,
			Port:     s.Port,
			User:     s.User,
			Password: s.Password,
			Remote:   s.Remote,
			Local:    s.Local,
		}
		switch strings.ToUpper(s.Protocol) {
		case "SFTP", "1":
			p.Protocol = config.ProtocolSFTP
		case "FTPS", "3", "4":
			p.Protocol = config.ProtocolFTPS
		default:
			p.Protocol = config.ProtocolFTP
		}
		if p.Port == 0 {
			p.Port = p.DefaultPort()
		}
		if p.Name == "" {
			p.Name = sanitizeName(host)
		}
		projects = append(projects, p)
	}
	return projects, nil
}

// ── .env ──────────────────────────────────────────────────────────────────────

func parseEnv(data []byte) ([]config.Project, error) {
	kv := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
			kv[key] = val
		}
	}

	// Support both FTP_* and SFTP_* prefixes
	host := first(kv, "FTP_HOST", "SFTP_HOST", "FTP_SERVER", "SFTP_SERVER")
	user := first(kv, "FTP_USER", "SFTP_USER", "FTP_USERNAME", "SFTP_USERNAME")
	pass := first(kv, "FTP_PASS", "SFTP_PASS", "FTP_PASSWORD", "SFTP_PASSWORD")
	remote := first(kv, "FTP_REMOTE", "SFTP_REMOTE", "FTP_PATH", "SFTP_PATH", "DEPLOY_PATH", "/")
	local := first(kv, "FTP_LOCAL", "SFTP_LOCAL", "LOCAL_PATH", "")
	portStr := first(kv, "FTP_PORT", "SFTP_PORT", "")
	proto := strings.ToLower(first(kv, "FTP_PROTOCOL", "SFTP_PROTOCOL", "PROTOCOL", ""))

	if host == "" {
		return nil, fmt.Errorf(".env: no FTP_HOST or SFTP_HOST found")
	}

	p := config.Project{
		Name:     sanitizeName(host),
		Host:     host,
		User:     user,
		Password: pass,
		Remote:   remote,
		Local:    local,
	}

	if proto == "sftp" || strings.Contains(strings.Join(keys(kv), " "), "SFTP") {
		p.Protocol = config.ProtocolSFTP
	} else if proto == "ftps" {
		p.Protocol = config.ProtocolFTPS
	} else {
		p.Protocol = config.ProtocolFTP
	}

	if n, err := strconv.Atoi(portStr); err == nil && n > 0 {
		p.Port = n
	} else {
		p.Port = p.DefaultPort()
	}

	return []config.Project{p}, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func first(m map[string]string, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != "" {
			return v
		}
	}
	// last arg may be a default
	if len(keys) > 0 {
		last := keys[len(keys)-1]
		if _, ok := m[last]; !ok {
			return last // treat as literal default
		}
	}
	return ""
}

func keys(m map[string]string) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}

func sanitizeName(s string) string {
	s = strings.TrimSpace(s)
	s = strings.NewReplacer(" ", "-", "/", "-", "\\", "-").Replace(s)
	s = strings.ToLower(s)
	return s
}
