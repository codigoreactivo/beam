package transfer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/codigoreactivo/beam/internal/config"
	"github.com/pkg/sftp"
	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

type sftpClient struct {
	ssh  *gossh.Client
	sftp *sftp.Client
}

func dialSFTP(p *config.Project) (*sftpClient, error) {
	auth, err := authMethods(p)
	if err != nil {
		return nil, fmt.Errorf("auth: %w", err)
	}

	hkcb, err := hostKeyCallback()
	if err != nil {
		return nil, fmt.Errorf("known_hosts: %w", err)
	}

	sshCfg := &gossh.ClientConfig{
		User:            p.User,
		Auth:            auth,
		HostKeyCallback: hkcb,
		Timeout:         15 * time.Second,
	}

	addr := net.JoinHostPort(p.Host, fmt.Sprintf("%d", p.Port))
	sshConn, err := gossh.Dial("tcp", addr, sshCfg)
	if err != nil {
		return nil, fmt.Errorf("ssh dial %s: %w", addr, err)
	}

	sftpConn, err := sftp.NewClient(sshConn,
		sftp.UseConcurrentWrites(true),
		sftp.UseConcurrentReads(true),
		sftp.MaxPacket(1<<15), // 32 KB packets (default ~34 KB but forces full pipeline)
	)
	if err != nil {
		sshConn.Close()
		return nil, fmt.Errorf("sftp session: %w", err)
	}

	return &sftpClient{ssh: sshConn, sftp: sftpConn}, nil
}

func authMethods(p *config.Project) ([]gossh.AuthMethod, error) {
	if p.Key != "" {
		keyPath := expandHome(p.Key)
		keyData, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("read key %s: %w", keyPath, err)
		}
		signer, err := gossh.ParsePrivateKey(keyData)
		if err != nil {
			return nil, fmt.Errorf("parse key %s: %w", keyPath, err)
		}
		return []gossh.AuthMethod{gossh.PublicKeys(signer)}, nil
	}
	if p.Password != "" {
		pass := p.Password // never expand env vars on passwords — $ is valid in passwords
		// Try both password and keyboard-interactive (required by many cPanel/WHM servers).
		return []gossh.AuthMethod{
			gossh.Password(pass),
			gossh.KeyboardInteractive(func(_, _ string, questions []string, _ []bool) ([]string, error) {
				answers := make([]string, len(questions))
				for i := range questions {
					answers[i] = pass
				}
				return answers, nil
			}),
		}, nil
	}
	return nil, fmt.Errorf("no credentials: set 'key' or 'password' in project config")
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[2:])
	}
	return p
}

func expandEnv(s string) string {
	return os.ExpandEnv(s)
}

func (c *sftpClient) Close() error {
	c.sftp.Close()
	return c.ssh.Close()
}

func (c *sftpClient) Walk(_ context.Context, remotePath string) ([]WalkEntry, error) {
	var entries []WalkEntry
	walker := c.sftp.Walk(remotePath)
	for walker.Step() {
		if walker.Err() != nil {
			continue // skip unreadable entries rather than aborting
		}
		fi := walker.Stat()
		full := walker.Path()
		rel := strings.TrimPrefix(full, remotePath)
		rel = strings.TrimPrefix(rel, "/")
		if rel == "" {
			continue
		}
		entries = append(entries, WalkEntry{
			RelPath: rel,
			Entry: Entry{
				Name:    fi.Name(),
				Size:    fi.Size(),
				IsDir:   fi.IsDir(),
				Mode:    fi.Mode(),
				ModTime: fi.ModTime(),
			},
		})
	}
	return entries, nil
}

func (c *sftpClient) List(_ context.Context, remotePath string) ([]Entry, error) {
	infos, err := c.sftp.ReadDir(remotePath)
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", remotePath, err)
	}
	entries := make([]Entry, len(infos))
	for i, fi := range infos {
		entries[i] = Entry{
			Name:    fi.Name(),
			Size:    fi.Size(),
			IsDir:   fi.IsDir(),
			Mode:    fi.Mode(),
			ModTime: fi.ModTime(),
		}
	}
	return entries, nil
}

func (c *sftpClient) Mkdir(_ context.Context, remotePath string) error {
	if err := c.sftp.MkdirAll(remotePath); err != nil {
		return fmt.Errorf("mkdir %s: %w", remotePath, err)
	}
	return nil
}

func (c *sftpClient) Remove(_ context.Context, remotePath string) error {
	// try file first, then directory
	if err := c.sftp.Remove(remotePath); err != nil {
		if err2 := c.sftp.RemoveDirectory(remotePath); err2 != nil {
			return fmt.Errorf("remove %s: %w", remotePath, err)
		}
	}
	return nil
}

func (c *sftpClient) Rename(_ context.Context, src, dst string) error {
	if err := c.sftp.Rename(src, dst); err != nil {
		return fmt.Errorf("rename %s → %s: %w", src, dst, err)
	}
	return nil
}

func (c *sftpClient) Upload(ctx context.Context, local, remote string) (int64, error) {
	info, err := os.Stat(local)
	if err != nil {
		return 0, fmt.Errorf("stat %s: %w", local, err)
	}
	if info.IsDir() {
		return c.uploadDir(ctx, local, remote)
	}
	return c.uploadFile(ctx, local, remote)
}

func (c *sftpClient) uploadFile(_ context.Context, local, remote string) (int64, error) {
	if err := c.sftp.MkdirAll(path.Dir(remote)); err != nil {
		return 0, fmt.Errorf("mkdir %s: %w", path.Dir(remote), err)
	}

	src, err := os.Open(local)
	if err != nil {
		return 0, fmt.Errorf("open %s: %w", local, err)
	}
	defer src.Close()

	dst, err := c.sftp.Create(remote)
	if err != nil {
		return 0, fmt.Errorf("create remote %s: %w", remote, err)
	}
	defer dst.Close()

	n, err := io.Copy(dst, src)
	if err != nil {
		return n, fmt.Errorf("copy %s → %s: %w", local, remote, err)
	}
	return n, nil
}

func (c *sftpClient) uploadDir(ctx context.Context, localDir, remoteDir string) (int64, error) {
	var total int64
	err := filepath.Walk(localDir, func(localPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		rel, _ := filepath.Rel(localDir, localPath)
		remotePath := path.Join(remoteDir, filepath.ToSlash(rel))
		if info.IsDir() {
			return c.sftp.MkdirAll(remotePath)
		}
		n, err := c.uploadFile(ctx, localPath, remotePath)
		total += n
		return err
	})
	return total, err
}

func (c *sftpClient) Download(_ context.Context, remote, local string) (int64, error) {
	src, err := c.sftp.Open(remote)
	if err != nil {
		return 0, fmt.Errorf("open remote %s: %w", remote, err)
	}
	defer src.Close()

	if err := os.MkdirAll(filepath.Dir(local), 0o755); err != nil {
		return 0, fmt.Errorf("mkdir %s: %w", filepath.Dir(local), err)
	}

	dst, err := os.Create(local)
	if err != nil {
		return 0, fmt.Errorf("create %s: %w", local, err)
	}
	defer dst.Close()

	n, err := io.Copy(dst, src)
	if err != nil {
		return n, fmt.Errorf("copy %s → %s: %w", remote, local, err)
	}
	return n, nil
}

func (c *sftpClient) Benchmark(_ context.Context, sizeBytes int64) (int64, int64, error) {
	remotePath := fmt.Sprintf("/tmp/.beam_bm_%d", time.Now().UnixNano())

	payload := syntheticPayload(sizeBytes)

	// Upload: write from memory directly to remote.
	uploadStart := time.Now()
	dst, err := c.sftp.Create(remotePath)
	if err != nil {
		return 0, 0, fmt.Errorf("benchmark create: %w", err)
	}
	if _, err := dst.Write(payload); err != nil {
		dst.Close()
		c.sftp.Remove(remotePath) //nolint
		return 0, 0, fmt.Errorf("benchmark write: %w", err)
	}
	dst.Close()
	uploadMs := time.Since(uploadStart).Milliseconds()

	// Download: read remote into /dev/null (io.Discard).
	downloadStart := time.Now()
	src, err := c.sftp.Open(remotePath)
	if err != nil {
		c.sftp.Remove(remotePath) //nolint
		return uploadMs, 0, fmt.Errorf("benchmark open: %w", err)
	}
	_, dlErr := io.Copy(io.Discard, src)
	src.Close()
	downloadMs := time.Since(downloadStart).Milliseconds()

	c.sftp.Remove(remotePath) //nolint — best-effort cleanup

	if dlErr != nil {
		return uploadMs, 0, fmt.Errorf("benchmark read: %w", dlErr)
	}
	return uploadMs, downloadMs, nil
}

// syntheticPayload returns a repeating byte pattern of the requested size.
func syntheticPayload(size int64) []byte {
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i & 0xff)
	}
	return data
}

// hostKeyCallback implements Trust-On-First-Use (TOFU) against
// ~/.beam/known_hosts. Unknown hosts are added automatically on first
// connection; a key mismatch (potential MITM) returns a hard error.
func hostKeyCallback() (gossh.HostKeyCallback, error) {
	khPath := config.KnownHostsPath()

	if _, err := os.Stat(khPath); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(khPath), 0o700); err != nil {
			return nil, err
		}
		if err := os.WriteFile(khPath, nil, 0o600); err != nil {
			return nil, err
		}
	}

	verify, err := knownhosts.New(khPath)
	if err != nil {
		return nil, err
	}

	return func(hostname string, remote net.Addr, key gossh.PublicKey) error {
		// If the server presents an SSH certificate, verify using its underlying
		// public key — this handles hosts that haven't pinned a CA in known_hosts.
		checkKey := key
		if cert, ok := key.(*gossh.Certificate); ok {
			checkKey = cert.Key
		}

		err := verify(hostname, remote, checkKey)
		if err == nil {
			return nil
		}
		var keyErr *knownhosts.KeyError
		if errors.As(err, &keyErr) && len(keyErr.Want) == 0 {
			// Host seen for the first time — record and trust (TOFU).
			f, openErr := os.OpenFile(khPath, os.O_APPEND|os.O_WRONLY, 0o600)
			if openErr != nil {
				return openErr
			}
			defer f.Close()
			line := knownhosts.Line([]string{knownhosts.Normalize(hostname)}, checkKey)
			_, writeErr := fmt.Fprintln(f, line)
			return writeErr
		}
		// Key mismatch — potential MITM, refuse connection.
		return fmt.Errorf("host key mismatch for %s: %w (edit %s to fix)", hostname, err, khPath)
	}, nil
}
