package main

import (
	"github.com/sagecontinuum/ses/pkg/cloudscheduler"
	"github.com/sagecontinuum/ses/pkg/util"
)

func main() {
	util.InfoLogger.Printf("initializing...")
	cloudscheduler.InitializeValidator()
	cloudscheduler.InitializeJobManager()

	// dryRun := flag.Bool("dry-run", false, "To emulate scheduler")
	// flag.Parse()
	go cloudscheduler.RunValidator()
	go cloudscheduler.RunJobManager()
	cloudscheduler.CreateRouter()
}
