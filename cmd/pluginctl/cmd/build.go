package cmd

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
)

func init() {
	const registryInterface = "lan0"
	const registryPort = "5000"
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

		registryIP, err := interfaceIPv4(registryInterface)
		if err != nil {
			return fmt.Errorf("failed to get registry address: %w", err)
		}
		registry := net.JoinHostPort(registryIP.String(), registryPort)
		image := fmt.Sprintf("%s/%s/%s", registry, namespace, name)

		// build and push to local registry. all output goes to stderr to allow easy piping
		ctx, _ := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		dockerArgs := []string{
			"build",
			"-t",
			image,
		}
		if len(args[1:]) > 0 {
			dockerArgs = append(dockerArgs, args[1:]...)
		}
		dockerArgs = append(dockerArgs, path)
		if err := runCommandContextStderr(ctx, "docker", dockerArgs...); err != nil {
			return err
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

// interfaceIPv4 returns the first IPv4 address assigned to the named network interface.
func interfaceIPv4(name string) (net.IP, error) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return nil, fmt.Errorf("interface %s: %w", name, err)
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, fmt.Errorf("interface %s: %w", name, err)
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
			if ip4 := ipnet.IP.To4(); ip4 != nil {
				return ip4, nil
			}
		}
	}
	return nil, fmt.Errorf("interface %s has no IPv4 address", name)
}
