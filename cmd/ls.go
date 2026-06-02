package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var lsCmd = &cobra.Command{
	Use:   "ls [remote-path]",
	Short: "List remote directory",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		_, client, err := dial(flagProject)
		if err != nil {
			return err
		}
		defer client.Close()

		remotePath := "/"
		if len(args) > 0 {
			remotePath = args[0]
		}

		entries, err := client.List(context.Background(), remotePath)
		if err != nil {
			return err
		}

		if flagJSON {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(entries)
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		for _, e := range entries {
			typ := "-"
			name := e.Name
			if e.IsDir {
				typ = "d"
				name += "/"
			}
			fmt.Fprintf(w, "%s\t%s\t%8d\t%s\t%s\n",
				typ,
				e.Mode.String(),
				e.Size,
				e.ModTime.Format("Jan 02 15:04"),
				name,
			)
		}
		return w.Flush()
	},
}

func init() {
	rootCmd.AddCommand(lsCmd)
}
