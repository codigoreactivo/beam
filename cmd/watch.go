package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	beamsync "github.com/jesusjhoel/beam/internal/sync"
	"github.com/jesusjhoel/beam/internal/transfer"
	"github.com/jesusjhoel/beam/internal/watcher"
	"github.com/spf13/cobra"
)

var watchOnce bool

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch local directory and auto-upload on change",
	RunE: func(cmd *cobra.Command, args []string) error {
		p, client, err := dial(flagProject)
		if err != nil {
			return err
		}

		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		if watchOnce {
			defer client.Close()
			plan, err := beamsync.Build(ctx, client, p)
			if err != nil {
				return err
			}
			return beamsync.Execute(ctx, client, plan, p, func(e beamsync.PlanEntry) {
				if !flagQuiet {
					fmt.Fprintf(cmd.OutOrStdout(), "  + %s (%s)\n", e.RelPath, formatBytes(e.LocalSize))
				}
			})
		}

		// Keep-alive: reconnect on each event since SFTP connections can time out
		client.Close()

		w, err := watcher.New(p)
		if err != nil {
			return fmt.Errorf("watcher: %w", err)
		}
		defer w.Close()

		if err := w.Start(ctx); err != nil {
			return fmt.Errorf("start watcher on %s: %w", p.Local, err)
		}

		if !flagQuiet {
			fmt.Fprintf(cmd.OutOrStdout(), "Watching %s → %s:%s\n", p.Local, p.Host, p.Remote)
			fmt.Fprintln(cmd.OutOrStdout(), "  Press Ctrl+C to stop")
		}

		for {
			select {
			case <-ctx.Done():
				fmt.Fprintln(cmd.OutOrStdout(), "\nStopped.")
				return nil

			case e := <-w.Events:
				start := time.Now()
				c, dialErr := transfer.Connect(p)
				if dialErr != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "  ✗ reconnect failed: %v\n", dialErr)
					continue
				}

				remotePath := path.Join(p.Remote, e.RelPath)
				n, uploadErr := c.Upload(ctx, e.Path, remotePath)
				c.Close()

				if uploadErr != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "  ✗ %s: %v\n", e.RelPath, uploadErr)
				} else if !flagQuiet {
					fmt.Fprintf(cmd.OutOrStdout(), "  ↑ %s (%s, %s)\n",
						e.RelPath, formatBytes(n), time.Since(start).Round(time.Millisecond))
				}
			}
		}
	},
}

func init() {
	watchCmd.Flags().BoolVar(&watchOnce, "once", false, "sync once then exit")
	rootCmd.AddCommand(watchCmd)
}
