package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/waggle-sensor/edge-scheduler/pkg/logger"
	"github.com/waggle-sensor/edge-scheduler/pkg/pluginctl"
	"k8s.io/client-go/util/homedir"
)

const (
	rancherKubeconfigPath = "/etc/rancher/k3s/k3s.yaml"
)

var (
	Version    = "0.0.0"
	debug      bool
	kubeconfig string
	followLog  bool
	stdin      bool
	tty        bool
)

var deployment = &pluginctl.Deployment{}

func getenv(key string, def string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return def
}

func detectDefaultKubeconfig() string {
	if _, err := os.ReadFile(rancherKubeconfigPath); err == nil {
		return rancherKubeconfigPath
	}
	if home := homedir.HomeDir(); home != "" {
		return filepath.Join(home, ".kube", "config")
	}
	return ""
}

func init() {
	rootCmd.AddCommand(completionCmd)
	// To prevent printing the usage when commands end with an error
	rootCmd.SilenceUsage = true
	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", getenv("KUBECONFIG", detectDefaultKubeconfig()), "path to the kubeconfig file")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "flag to debug")
}

var rootCmd = &cobra.Command{
	Use: "pluginctl [FLAGS] [COMMANDS]",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if !debug {
			logger.Debug.SetOutput(io.Discard)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("SAGE plugin control for running plugins: %s\n", Version)
		fmt.Printf("pluginctl --help for more information\n")
	},
	ValidArgs: []string{"deploy", "logs"},
}

var completionCmd = &cobra.Command{
	Use:                   "completion [bash|zsh|fish|powershell]",
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.ExactValidArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			cmd.Root().GenPowerShellCompletion(os.Stdout)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
