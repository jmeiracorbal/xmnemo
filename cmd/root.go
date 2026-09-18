package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "xmnemo",
	Short: "Build a custom mnemo binary with selected adapters",
	Long: `xmnemo is a developer tool for building a custom mnemo binary
that includes only the agent adapters you need.

Requires Go to be installed. Intended for developers extending mnemo
with their own adapters — most mnemo users never need this tool.

Example:
  xmnemo build --adapter github.com/acme/mnemo-zed@v1.2.0`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
