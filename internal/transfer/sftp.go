package transfer

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/jesusjhoel/beam/internal/config"
	"github.com/pkg/sftp"
	gossh "golang.org/x/crypto/ssh"
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

	sshCfg := &gossh.ClientConfig{
		User:            p.User,
		Auth:            auth,
		HostKeyCallback: gossh.InsecureIgnoreHostKey(), // TODO: use known_hosts
		Timeout:         15 * time.Second,
	}

	addr := net.JoinHostPort(p.Host, fmt.Sprintf("%d", p.Port))
	sshConn, err := gossh.Dial("tcp", addr, sshCfg)
	if err != nil {
		return nil, fmt.Errorf("ssh dial %s: %w", addr, err)
	}

	sftpConn, err := sftp.NewClient(sshConn)
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
		return []gossh.AuthMethod{gossh.Password(expandEnv(p.Password))}, nil
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
