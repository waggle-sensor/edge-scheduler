package main

import (
	"github.com/sagecontinuum/ses/pkg/nodescheduler"
)

func main() {
	// dryRun := flag.Bool("dry-run", false, "To emulate scheduler")
	// flag.Parse()

	nodescheduler.RunNodeScheduler()
}
