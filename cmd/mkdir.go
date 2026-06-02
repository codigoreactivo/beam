package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var mkdirCmd = &cobra.Command{
	Use:   "mkdir <remote-path>",
	Short: "Create a remote directory",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		_, client, err := dial(flagProject)
		if err != nil {
			return err
		}
		defer client.Close()

		if err := client.Mkdir(context.Background(), args[0]); err != nil {
			return err
		}
		if !flagQuiet {
			fmt.Fprintf(cmd.OutOrStdout(), "✓ %s created\n", args[0])
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(mkdirCmd)
}
