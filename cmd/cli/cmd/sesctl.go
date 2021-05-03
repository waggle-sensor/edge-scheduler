package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "1.0.0"

var rootCmd = &cobra.Command{
	Use: "sesctl",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("SAGE edge scheduler client version: %s\n", version)
		fmt.Printf("sesctl --help for more information\n")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
