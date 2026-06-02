package cmd

import (
	"context"
	"fmt"

	beamsync "github.com/codigoreactivo/beam/internal/sync"
	"github.com/spf13/cobra"
)

var syncDryRun bool

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync local directory → remote (incremental, skips unchanged files)",
	RunE: func(cmd *cobra.Command, args []string) error {
		p, client, err := dial(flagProject)
		if err != nil {
			return err
		}
		defer client.Close()

		ctx := context.Background()
		if !flagQuiet {
			fmt.Fprintf(cmd.OutOrStdout(), "Building sync plan for %q ...\n", p.Name)
		}

		plan, err := beamsync.Build(ctx, client, p)
		if err != nil {
			return fmt.Errorf("build plan: %w", err)
		}

		if plan.AddCount+plan.UpdateCount == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "✓ Already in sync — nothing to upload")
			return nil
		}

		if syncDryRun {
			return printPlan(cmd.OutOrStdout(), plan)
		}

		if !flagQuiet {
			printPlan(cmd.OutOrStdout(), plan) //nolint:errcheck
			fmt.Fprintln(cmd.OutOrStdout())
		}

		err = beamsync.Execute(ctx, client, plan, p, func(e beamsync.PlanEntry) {
			if !flagQuiet {
				verb := "+"
				if e.Action == beamsync.ActionUpdate {
					verb = "↑"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "  %s %s (%s)\n", verb, e.RelPath, formatBytes(e.LocalSize))
			}
		})
		if err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "✓ %d added, %d updated, %d skipped (%s total)\n",
			plan.AddCount, plan.UpdateCount, plan.SkipCount, formatBytes(plan.TotalBytes()))
		return nil
	},
}

func init() {
	syncCmd.Flags().BoolVar(&syncDryRun, "dry-run", false, "preview what would be synced without transferring")
	rootCmd.AddCommand(syncCmd)
}
