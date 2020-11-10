package cloudscheduler

import (
	"github.com/sagecontinuum/ses/pkg/logger"
)

// RunCloudScheduler initializes itself and runs the main routine
func RunCloudScheduler() {
	logger.Info.Printf("initializing...")
	InitializeValidator()
	InitializeJobManager()

	// dryRun := flag.Bool("dry-run", false, "To emulate scheduler")
	// flag.Parse()
	go RunValidator()
	go RunJobManager()
	CreateRouter()
}
