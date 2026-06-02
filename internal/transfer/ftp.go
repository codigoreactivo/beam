package transfer

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/codigoreactivo/beam/internal/config"
	goftp "github.com/jlaffaye/ftp"
)

type ftpClient struct {
	conn    *goftp.ServerConn
	project *config.Project
}

func dialFTP(p *config.Project) (*ftpClient, error) {
	addr := net.JoinHostPort(p.Host, fmt.Sprintf("%d", p.Port))

	opts := []goftp.DialOption{
		goftp.DialWithTimeout(15 * time.Second),
	}

	if p.Protocol == config.ProtocolFTPS {
		opts = append(opts, goftp.DialWithExplicitTLS(&tls.Config{
			ServerName: p.Host,
			MinVersion: tls.VersionTLS12,
		}))
	}

	conn, err := goftp.Dial(addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("ftp dial %s: %w", addr, err)
	}

	password := p.Password // never expand env vars — $ is valid in passwords
	if err := conn.Login(p.User, password); err != nil {
		conn.Quit()
		return nil, fmt.Errorf("ftp login %s@%s: %w", p.User, addr, err)
	}

	return &ftpClient{conn: conn, project: p}, nil
}

func (c *ftpClient) Close() error {
	return c.conn.Quit()
}

func (c *ftpClient) List(_ context.Context, remotePath string) ([]Entry, error) {
	entries, err := c.conn.List(remotePath)
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", remotePath, err)
	}
	result := make([]Entry, 0, len(entries))
	for _, e := range entries {
		if e.Name == "." || e.Name == ".." {
			continue
		}
		result = append(result, Entry{
			Name:    e.Name,
			Size:    int64(e.Size),
			IsDir:   e.Type == goftp.EntryTypeFolder,
			ModTime: e.Time,
		})
	}
	return result, nil
}

func (c *ftpClient) Walk(ctx context.Context, remotePath string) ([]WalkEntry, error) {
	var entries []WalkEntry
	if err := c.walkDir(ctx, remotePath, remotePath, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func (c *ftpClient) walkDir(ctx context.Context, root, current string, out *[]WalkEntry) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	items, err := c.conn.List(current)
	if err != nil {
		return fmt.Errorf("list %s: %w", current, err)
	}
	for _, item := range items {
		if item.Name == "." || item.Name == ".." {
			continue
		}
		full := path.Join(current, item.Name)
		rel := strings.TrimPrefix(strings.TrimPrefix(full, root), "/")
		entry := WalkEntry{
			RelPath: rel,
			Entry: Entry{
				Name:    item.Name,
				Size:    int64(item.Size),
				IsDir:   item.Type == goftp.EntryTypeFolder,
				ModTime: item.Time,
			},
		}
		*out = append(*out, entry)
		if item.Type == goftp.EntryTypeFolder {
			if err := c.walkDir(ctx, root, full, out); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *ftpClient) Mkdir(_ context.Context, remotePath string) error {
	// MkdirAll equivalent: create each segment
	parts := strings.Split(strings.Trim(remotePath, "/"), "/")
	current := ""
	for _, p := range parts {
		current = path.Join("/", current, p)
		// ignore "already exists" errors
		c.conn.MakeDir(current) //nolint:errcheck
	}
	return nil
}

func (c *ftpClient) Remove(_ context.Context, remotePath string) error {
	if err := c.conn.Delete(remotePath); err != nil {
		// try directory
		if err2 := c.conn.RemoveDirRecur(remotePath); err2 != nil {
			return fmt.Errorf("remove %s: %w", remotePath, err)
		}
	}
	return nil
}

func (c *ftpClient) Rename(_ context.Context, src, dst string) error {
	if err := c.conn.Rename(src, dst); err != nil {
		return fmt.Errorf("rename %s → %s: %w", src, dst, err)
	}
	return nil
}

func (c *ftpClient) Upload(ctx context.Context, local, remote string) (int64, error) {
	info, err := os.Stat(local)
	if err != nil {
		return 0, fmt.Errorf("stat %s: %w", local, err)
	}
	if info.IsDir() {
		return c.uploadDir(ctx, local, remote)
	}
	return c.uploadFile(ctx, local, remote)
}

func (c *ftpClient) uploadFile(_ context.Context, local, remote string) (int64, error) {
	// ensure parent directory exists
	c.Mkdir(context.Background(), path.Dir(remote)) //nolint:errcheck

	f, err := os.Open(local)
	if err != nil {
		return 0, fmt.Errorf("open %s: %w", local, err)
	}
	defer f.Close()

	// wrap to count bytes
	cr := &countReader{r: f}
	if err := c.conn.Stor(remote, cr); err != nil {
		return cr.n, fmt.Errorf("stor %s: %w", remote, err)
	}
	return cr.n, nil
}

func (c *ftpClient) uploadDir(ctx context.Context, localDir, remoteDir string) (int64, error) {
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
			c.Mkdir(ctx, remotePath) //nolint:errcheck
			return nil
		}
		n, err := c.uploadFile(ctx, localPath, remotePath)
		total += n
		return err
	})
	return total, err
}

func (c *ftpClient) Download(_ context.Context, remote, local string) (int64, error) {
	resp, err := c.conn.Retr(remote)
	if err != nil {
		return 0, fmt.Errorf("retr %s: %w", remote, err)
	}
	defer resp.Close()

	if err := os.MkdirAll(filepath.Dir(local), 0o755); err != nil {
		return 0, fmt.Errorf("mkdir %s: %w", filepath.Dir(local), err)
	}

	dst, err := os.Create(local)
	if err != nil {
		return 0, fmt.Errorf("create %s: %w", local, err)
	}
	defer dst.Close()

	n, err := io.Copy(dst, resp)
	if err != nil {
		return n, fmt.Errorf("copy %s → %s: %w", remote, local, err)
	}
	return n, nil
}

func (c *ftpClient) Benchmark(_ context.Context, sizeBytes int64) (int64, int64, error) {
	remotePath := fmt.Sprintf("/tmp/.beam_bm_%d", time.Now().UnixNano())

	payload := syntheticPayload(sizeBytes)

	// Upload from memory.
	uploadStart := time.Now()
	if err := c.conn.Stor(remotePath, bytes.NewReader(payload)); err != nil {
		return 0, 0, fmt.Errorf("benchmark stor: %w", err)
	}
	uploadMs := time.Since(uploadStart).Milliseconds()

	// Download to discard.
	downloadStart := time.Now()
	resp, err := c.conn.Retr(remotePath)
	if err != nil {
		c.conn.Delete(remotePath) //nolint
		return uploadMs, 0, fmt.Errorf("benchmark retr: %w", err)
	}
	_, dlErr := io.Copy(io.Discard, resp)
	resp.Close()
	downloadMs := time.Since(downloadStart).Milliseconds()

	c.conn.Delete(remotePath) //nolint — best-effort cleanup

	if dlErr != nil {
		return uploadMs, 0, fmt.Errorf("benchmark read: %w", dlErr)
	}
	return uploadMs, downloadMs, nil
}

// countReader wraps a reader and counts bytes read.
type countReader struct {
	r io.Reader
	n int64
}

func (cr *countReader) Read(p []byte) (int, error) {
	n, err := cr.r.Read(p)
	cr.n += int64(n)
	return n, err
}
