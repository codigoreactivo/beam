package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var mvCmd = &cobra.Command{
	Use:   "mv <src> <dst>",
	Short: "Rename or move a remote file",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		_, client, err := dial(flagProject)
		if err != nil {
			return err
		}
		defer client.Close()

		if err := client.Rename(context.Background(), args[0], args[1]); err != nil {
			return err
		}
		if !flagQuiet {
			fmt.Fprintf(cmd.OutOrStdout(), "✓ %s → %s\n", args[0], args[1])
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(mvCmd)
}
