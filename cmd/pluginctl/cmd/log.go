package cmd

import (
	"fmt"
	"io"

	"github.com/sagecontinuum/ses/pkg/logger"
	"github.com/sagecontinuum/ses/pkg/nodescheduler"
	"github.com/spf13/cobra"
)

func init() {
	cmdLog.Flags().BoolVarP(&followLog, "follow", "f", false, "Specified if logs should be streamed")
	rootCmd.AddCommand(cmdLog)
}

var cmdLog = &cobra.Command{
	Use:              "log APP_NAME",
	Short:            "Print logs of a plugin",
	TraverseChildren: true,
	Args:             cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		logger.Debug.Printf("kubeconfig: %s", kubeconfig)
		name = args[0]
		logger.Debug.Printf("args: %v", args)

		resourceManager, err := nodescheduler.NewK3SResourceManager("", false, kubeconfig, nil, false)
		if err != nil {
			fmt.Printf("ERROR: %s", err.Error())
		}
		resourceManager.Namespace = "default"
		podLog, err := resourceManager.GetPodLog(name, true)
		if err != nil {
			fmt.Printf("ERROR: %s", err.Error())
		} else {
			defer podLog.Close()
			buf := make([]byte, 2000)
			for {
				numBytes, err := podLog.Read(buf)
				if numBytes == 0 {
					break
				}
				if err == io.EOF {
					break
				} else if err != nil {
					// return err
				}
				fmt.Println(string(buf[:numBytes]))
			}
		}
	},
}
