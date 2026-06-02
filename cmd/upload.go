package cmd

import (
	"context"
	"fmt"
	"path"
	"path/filepath"

	"github.com/spf13/cobra"
)

var uploadCmd = &cobra.Command{
	Use:   "upload <local> [remote]",
	Short: "Upload a file or directory to remote",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		p, client, err := dial(flagProject)
		if err != nil {
			return err
		}
		defer client.Close()

		local := args[0]
		remote := p.Remote
		if len(args) > 1 {
			remote = args[1]
		} else {
			// default: remote root / local basename
			remote = path.Join(p.Remote, filepath.Base(local))
		}

		if !flagQuiet {
			fmt.Fprintf(cmd.OutOrStdout(), "Uploading %s → %s ...\n", local, remote)
		}

		n, err := client.Upload(context.Background(), local, remote)
		if err != nil {
			return err
		}

		if !flagQuiet {
			fmt.Fprintf(cmd.OutOrStdout(), "✓ %s uploaded (%s)\n", local, formatBytes(n))
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(uploadCmd)
}

func formatBytes(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}
