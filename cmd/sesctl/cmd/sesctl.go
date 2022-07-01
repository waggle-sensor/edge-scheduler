package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/waggle-sensor/edge-scheduler/pkg/logger"
)

var (
	Version          string
	debug            bool
	name             string
	node             string
	privileged       bool
	job              string
	selectorStr      string
	followLog        bool
	entrypoint       string
	stdin            bool
	tty              bool
	serverHostString string
)

func getenv(key string, def string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return def
}

func init() {
	var rootCmd = &cobra.Command{
		Use: "sesctl [FLAGS] [COMMANDS]",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if !debug {
				logger.Debug.SetOutput(io.Discard)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("SAGE edge scheduler client version: %s\n", Version)
			fmt.Printf("sesctl --help for more information\n")
		},
		// ValidArgs: []string{"deploy", "logs"},
	}
	// To prevent printing the usage when commands end with an error
	// rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
	rootCmd.PersistentFlags().StringVar(&serverHostString, "server", getenv("SES_HOST", "http://localhost:9770"), "path to the kubeconfig file")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "flag to debug")
}

var rootCmd = &cobra.Command{
	Use: "sesctl [FLAGS] [COMMANDS]",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if !debug {
			logger.Debug.SetOutput(io.Discard)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("SAGE edge scheduler client version: %s\n", Version)
		fmt.Printf("sesctl --help for more information\n")
	},
	// ValidArgs: []string{"deploy", "logs"},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
