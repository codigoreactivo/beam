package cmd

import (
	"fmt"

	beammcp "github.com/codigoreactivo/beam/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "MCP server commands",
}

var mcpPort int

var mcpServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the MCP server (stdio transport for Claude Desktop)",
	Long: `Starts Beam's MCP server using stdio transport.

Configure in Claude Desktop (~/Library/Application Support/Claude/claude_desktop_config.json):
  {
    "mcpServers": {
      "beam": {
        "command": "beam",
        "args": ["mcp", "serve"]
      }
    }
  }`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !flagQuiet {
			fmt.Fprintln(cmd.ErrOrStderr(), "beam mcp: listening on stdio")
		}
		return beammcp.Run(mcpPort)
	},
}

func init() {
	mcpServeCmd.Flags().IntVar(&mcpPort, "port", 7071, "port (reserved for future HTTP transport)")
	mcpCmd.AddCommand(mcpServeCmd)
	rootCmd.AddCommand(mcpCmd)
}
