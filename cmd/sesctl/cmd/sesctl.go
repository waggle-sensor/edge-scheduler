package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/waggle-sensor/edge-scheduler/pkg/logger"
)

var (
	Version string
	debug   bool
)

var jobRequest = &JobRequest{}

func getenv(key string, def string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return def
}

func init() {
	// To prevent printing the usage when commands end with an error
	// rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
	rootCmd.PersistentFlags().StringVar(&jobRequest.ServerHostString, "server", getenv("SES_HOST", "http://localhost:9770"), "Path to the kubeconfig file")
	rootCmd.PersistentFlags().StringVar(&jobRequest.UserToken, "token", getenv("SES_USER_TOKEN", ""), "User token")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "flag to debug")
}

var rootCmd = &cobra.Command{
	Use: "sesctl [FLAGS] [COMMANDS]",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if !debug {
			logger.Debug.SetOutput(io.Discard)
		}
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if jobRequest.UserToken == "" {
			return fmt.Errorf("Must provide a valid token")
		}
		return nil
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
