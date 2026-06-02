package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/codigoreactivo/beam/internal/config"
	"github.com/spf13/cobra"
)

var (
	logsAll    bool
	logsExport bool
	logsLines  int
	logsFollow bool
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View project logs",
	RunE: func(cmd *cobra.Command, args []string) error {
		if logsAll {
			return tailAll(cmd.OutOrStdout())
		}
		if flagProject == "" {
			return fmt.Errorf("--project / -p is required (or use --all)")
		}
		logPath := filepath.Join(config.LogsDir(), flagProject+".log")
		if logsExport {
			return exportLog(logPath, flagProject)
		}
		return tailLog(cmd.OutOrStdout(), logPath, logsFollow)
	},
}

func tailLog(w io.Writer, logPath string, follow bool) error {
	f, err := os.Open(logPath)
	if os.IsNotExist(err) {
		fmt.Fprintln(w, "No logs yet.")
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()

	// seek to last N lines
	if logsLines > 0 {
		if err := seekLastN(f, logsLines); err != nil {
			f.Seek(0, io.SeekStart) //nolint:errcheck
		}
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fmt.Fprintln(w, scanner.Text())
	}

	if !follow {
		return scanner.Err()
	}

	// simple poll-based tail -f
	for {
		for scanner.Scan() {
			fmt.Fprintln(w, scanner.Text())
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func tailAll(w io.Writer) error {
	logsDir := config.LogsDir()
	entries, err := os.ReadDir(logsDir)
	if os.IsNotExist(err) {
		fmt.Fprintln(w, "No logs yet.")
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".log" {
			continue
		}
		fmt.Fprintf(w, "\n=== %s ===\n", e.Name())
		tailLog(w, filepath.Join(logsDir, e.Name()), false) //nolint:errcheck
	}
	return nil
}

func exportLog(logPath, project string) error {
	dst := project + "-" + time.Now().Format("20060102-150405") + ".log"
	src, err := os.Open(logPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("no logs for %q", project)
	}
	if err != nil {
		return err
	}
	defer src.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, src); err != nil {
		return err
	}
	fmt.Printf("✓ Logs exported to %s\n", dst)
	return nil
}

// seekLastN positions f so that reading it forward yields (at most) n lines.
func seekLastN(f *os.File, n int) error {
	const blockSize = 4096
	fi, err := f.Stat()
	if err != nil {
		return err
	}
	size := fi.Size()
	if size == 0 {
		return nil
	}

	var lines int
	offset := size
	buf := make([]byte, blockSize)

	for offset > 0 && lines <= n {
		read := int64(blockSize)
		if read > offset {
			read = offset
		}
		offset -= read
		f.Seek(offset, io.SeekStart) //nolint:errcheck
		nr, _ := f.Read(buf[:read])
		for i := nr - 1; i >= 0; i-- {
			if buf[i] == '\n' {
				lines++
				if lines > n {
					f.Seek(offset+int64(i)+1, io.SeekStart) //nolint:errcheck
					return nil
				}
			}
		}
	}
	f.Seek(0, io.SeekStart) //nolint:errcheck
	return nil
}

func init() {
	logsCmd.Flags().BoolVar(&logsAll, "all", false, "show logs for all projects")
	logsCmd.Flags().BoolVar(&logsExport, "export", false, "export logs to a file")
	logsCmd.Flags().IntVarP(&logsLines, "lines", "n", 50, "number of lines to show")
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "follow log output (tail -f)")
	rootCmd.AddCommand(logsCmd)
}
