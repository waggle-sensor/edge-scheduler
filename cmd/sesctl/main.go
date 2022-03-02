package main

import (
	"github.com/sagecontinuum/ses/cmd/sesctl/cmd"
)

var Version = "0.0.0"

func main() {
	cmd.Version = Version
	// k := kubectl.NewDefaultKubectlCommand()
	// err := k.Execute()
	// f := util.NewFactory(genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag())
	// cmd := exec.NewCmdExec(f, genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr})
	// opt := exec.ExecOptions{}
	// args := []string{"pod/node-influxdb-77bb74f689-nr5zm", "/bin/date"}
	// err := opt.Complete(f, cmd, args, 1)
	// if err != nil {
	// 	panic(err)
	// }
	// err = opt.Validate()
	// if err != nil {
	// 	panic(err)
	// }
	// err = opt.Run()
	// // exec.NewCmdExec()
	// if err != nil {
	// 	panic(err)
	// }
	// os.Args = []string{os.Args[0], "--kubeconfig", "/Users/yongho.kim/tmp/pirate_kubeconfig", "exec", "-n", "default", "-it", "node-influxdb-77bb74f689-nr5zm", "--", "/bin/date"}

	cmd.Execute()
}
