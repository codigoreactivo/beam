package cmd

import (
	"context"
	"fmt"
	"io"

	beamsync "github.com/codigoreactivo/beam/internal/sync"
	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Show diff between local and remote directories",
	RunE: func(cmd *cobra.Command, args []string) error {
		p, client, err := dial(flagProject)
		if err != nil {
			return err
		}
		defer client.Close()

		plan, err := beamsync.Build(context.Background(), client, p)
		if err != nil {
			return fmt.Errorf("diff: %w", err)
		}

		return printPlan(cmd.OutOrStdout(), plan)
	},
}

func init() {
	rootCmd.AddCommand(diffCmd)
}

func printPlan(w io.Writer, plan *beamsync.Plan) error {
	if plan.AddCount+plan.UpdateCount == 0 {
		fmt.Fprintln(w, "✓ No differences — local and remote are in sync")
		return nil
	}

	for _, e := range plan.Entries {
		switch e.Action {
		case beamsync.ActionAdd:
			fmt.Fprintf(w, "  + %-50s %s\n", e.RelPath, formatBytes(e.LocalSize))
		case beamsync.ActionUpdate:
			fmt.Fprintf(w, "  ↑ %-50s %s → %s\n", e.RelPath,
				formatBytes(e.RemoteSize), formatBytes(e.LocalSize))
		}
	}
	fmt.Fprintf(w, "\n  %d to add, %d to update, %d unchanged\n",
		plan.AddCount, plan.UpdateCount, plan.SkipCount)
	return nil
}
