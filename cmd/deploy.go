package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/jesusjhoel/beam/internal/hook"
	beamsync "github.com/jesusjhoel/beam/internal/sync"
	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Full deploy: pre-hooks → sync → post-hooks",
	RunE: func(cmd *cobra.Command, args []string) error {
		p, client, err := dial(flagProject)
		if err != nil {
			return err
		}
		defer client.Close()

		ctx := context.Background()
		hooks := hook.NewEngine()
		start := time.Now()
		hooksRun := []string{}

		// ── pre-deploy hooks ──────────────────────────────────────────
		results, err := hooks.Fire(ctx, hook.EventPreDeploy, p)
		for _, r := range results {
			hooksRun = append(hooksRun, fmt.Sprintf("pre-deploy: %s", r.Label))
			if !flagQuiet {
				icon := "✓"
				if !r.OK() {
					icon = "✗"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "  %s hook [%s] %s (%s)\n", icon, r.Type, r.Label, r.Duration.Round(time.Millisecond))
			}
		}
		if err != nil {
			return fmt.Errorf("pre-deploy hook failed: %w", err)
		}

		// ── sync ──────────────────────────────────────────────────────
		if !flagQuiet {
			fmt.Fprintln(cmd.OutOrStdout(), "  → syncing...")
		}
		plan, err := beamsync.Build(ctx, client, p)
		if err != nil {
			return fmt.Errorf("sync plan: %w", err)
		}
		if err := beamsync.Execute(ctx, client, plan, p, func(e beamsync.PlanEntry) {
			if !flagQuiet {
				verb := "+"
				if e.Action == beamsync.ActionUpdate {
					verb = "↑"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "    %s %s (%s)\n", verb, e.RelPath, formatBytes(e.LocalSize))
			}
		}); err != nil {
			return fmt.Errorf("sync: %w", err)
		}

		// ── post-deploy hooks ─────────────────────────────────────────
		results, err = hooks.Fire(ctx, hook.EventPostDeploy, p)
		for _, r := range results {
			hooksRun = append(hooksRun, fmt.Sprintf("post-deploy: %s", r.Label))
			if !flagQuiet {
				icon := "✓"
				if !r.OK() {
					icon = "✗"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "  %s hook [%s] %s (%s)\n", icon, r.Type, r.Label, r.Duration.Round(time.Millisecond))
			}
		}
		if err != nil {
			return fmt.Errorf("post-deploy hook failed: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "✓ Deployed %q — %d added, %d updated, %d skipped, %d hooks (%s)\n",
			p.Name, plan.AddCount, plan.UpdateCount, plan.SkipCount,
			len(hooksRun), time.Since(start).Round(time.Millisecond))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)
}
