package cmd

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
)

var downloadCmd = &cobra.Command{
	Use:   "download <remote> [local]",
	Short: "Download a file or directory from remote",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		_, client, err := dial(flagProject)
		if err != nil {
			return err
		}
		defer client.Close()

		remote := args[0]
		local := filepath.Base(remote)
		if len(args) > 1 {
			local = args[1]
		}

		if !flagQuiet {
			fmt.Fprintf(cmd.OutOrStdout(), "Downloading %s → %s ...\n", remote, local)
		}

		n, err := client.Download(context.Background(), remote, local)
		if err != nil {
			return err
		}

		if !flagQuiet {
			fmt.Fprintf(cmd.OutOrStdout(), "✓ %s downloaded (%s)\n", remote, formatBytes(n))
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(downloadCmd)
}
