package main

import (
	"fmt"
	"os"

	"github.com/sagecontinuum/ses/pkg/runplugin"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("usage: %s plugin-image [plugin-args]", os.Args[0])
		os.Exit(1)
	}

	runplugin.RunPlugin(os.Args[1], os.Args[2:]...)
}
