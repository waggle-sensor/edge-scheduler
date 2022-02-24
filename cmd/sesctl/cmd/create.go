package cmd

import (
	"fmt"

	"github.com/sagecontinuum/ses/pkg/datatype"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"
)

var plugins []string

func init() {
	flags := cmdCreate.Flags()
	flags.StringSliceVarP(&plugins, "plugin", "p", []string{}, "Plugin Docker image and version")
	// flags.StringVarP(&deployment.Name, "name", "n", "", "Specify plugin name")
	// flags.StringVar(&deployment.Node, "node", "", "run plugin on node")
	// // flags.StringVarP(&job, "job", "j", "sage", "Specify job name")
	// flags.StringVar(&deployment.SelectorString, "selector", "", "Specify where plugin can run")
	// flags.StringVar(&deployment.Entrypoint, "entrypoint", "", "Specify command to run inside plugin")
	// flags.BoolVarP(&deployment.Privileged, "privileged", "p", false, "Deploy as privileged plugin")
	// flags.BoolVar(&deployment.DevelopMode, "develop", false, "Enable the following development time features: access to wan network")
	rootCmd.AddCommand(cmdCreate)
}

var cmdCreate = &cobra.Command{
	Use:              "create [FLAGS] OUTPUT_FILE",
	Short:            "Create a job template for submission",
	TraverseChildren: true,
	Args:             cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(plugins) < 1 {
			return fmt.Errorf("No plugin is selected. Please specify at least one plugin.")
		}
		fmt.Printf("%v", plugins)
		node := &datatype.Node{
			Name: "Node01",
		}
		job := &datatype.Job{
			Name:       "hello world",
			PluginTags: []string{"hello", "and", "world"},
			Nodes: []*datatype.Node{
				node,
			},
		}
		jsonblob, _ := yaml.Marshal(job)
		fmt.Printf("%s", string(jsonblob))
		return nil
	},
}
