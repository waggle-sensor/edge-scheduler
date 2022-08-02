package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "build PLUGIN_DIR",
		Short: "Build plugin from directory",
		Args:  cobra.ExactArgs(1),
	}

	flags := cmd.Flags()
	tag := flags.String("t", "", "image tag")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		path, err := filepath.Abs(args[0])
		if err != nil {
			return fmt.Errorf("invalid path: %s", err.Error())
		}

		name := filepath.Base(path)

		if *tag == "" {
			*tag = fmt.Sprintf("10.31.81.1:5000/local/%s", name)
		}

		// build and push to local registry. all output goes to stderr to allow easy piping
		ctx, _ := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)

		if err := runCommandContextStderr(ctx, "docker", "build", "-t", *tag, path); err != nil {
			return err
		}

		if err := runCommandContextStderr(ctx, "docker", "push", *tag); err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "successfully build image!\n\n")

		// print tag to stdout to allow piping into other commands: pluginctl run -n my-plugin $(pluginctl build .)
		fmt.Printf("%s\n", *tag)

		return nil
	}

	rootCmd.AddCommand(cmd)
}

func runCommandContextStderr(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
