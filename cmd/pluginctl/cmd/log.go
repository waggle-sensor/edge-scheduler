package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/sagecontinuum/ses/pkg/logger"
	"github.com/sagecontinuum/ses/pkg/pluginctl"
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
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		logger.Debug.Printf("kubeconfig: %s", kubeconfig)
		name = args[0]
		logger.Debug.Printf("args: %v", args)
		pluginCtl, err := pluginctl.NewPluginCtl(kubeconfig)
		if err != nil {
			return
		}
		printLogFunc, terminateLog, err := pluginCtl.PrintLog(name, followLog)
		if err != nil {
			return
		} else {
			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt, syscall.SIGTERM)
			go printLogFunc()
			for {
				select {
				case <-c:
					logger.Debug.Println("Log terminated from user")
					return
				case <-terminateLog:
					logger.Debug.Println("Log terminated from handler")
					return
				}
			}
		}

		// podLog, err := pluginctl.Log(name, followLog)
		// if err != nil {
		// 	logger.Error.Println("%s", err.Error())
		// } else {
		// 	c := make(chan os.Signal, 1)
		// 	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		// 	go func() {
		// 		buf := make([]byte, 2000)
		// 		for {
		// 			numBytes, err := podLog.Read(buf)
		// 			if numBytes == 0 {
		// 				break
		// 			}
		// 			if err == io.EOF {
		// 				break
		// 			} else if err != nil {
		// 				// return err
		// 			}
		// 			fmt.Println(string(buf[:numBytes]))
		// 		}
		// 		c <- nil
		// 	}()
		// 	<-c
		// 	podLog.Close()
		// }
	},
}
