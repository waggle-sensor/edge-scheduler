package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

func init() {
	const registry = "10.31.81.1:5000"
	const namespace = "local"

	cmd := &cobra.Command{
		Use:   "build PLUGIN_DIR [FLAGS] [-- BUILD ARGUMENTS]",
		Short: "Build plugin",
		Long:  "Build a plugin contained in a directory.",
		Example: `# clone plugin repo
git clone https://github.com/my-username/my-plugin

# build and run plugin in cloned directory
pluginctl run -n my-plugin $(pluginctl build my-plugin)

# build with parameters that work on container builder (i.e., Docker)
pluginctl build my-plugin -- --build-arg=MY_VAR="hello"`,
		Args: cobra.MinimumNArgs(1),
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		path, err := filepath.Abs(args[0])
		if err != nil {
			return fmt.Errorf("invalid plugin path: %s", err.Error())
		}
		name := filepath.Base(path)
		image := fmt.Sprintf("%s/%s/%s", registry, namespace, name)

		// build and push to local registry. all output goes to stderr to allow easy piping
		ctx, _ := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		if len(args[1:]) < 1 {
			if err := runCommandContextStderr(ctx, "docker", "build", "-t", image, path); err != nil {
				return err
			}
		} else {
			if err := runCommandContextStderr(ctx, "docker", "build", "-t", image, strings.Join(args[1:], " "), path); err != nil {
				return err
			}
		}

		if err := runCommandContextStderr(ctx, "docker", "push", image); err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "Successfully built plugin\n\n")

		// print tag to stdout to allow piping into other commands: pluginctl run -n my-plugin $(pluginctl build .)
		fmt.Printf("%s\n", image)

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
