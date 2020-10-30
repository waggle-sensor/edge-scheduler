package main

import "flag"

func main() {
	dryRun := flag.Bool("dry-run", false, "To emulate scheduler")
	flag.Parse()

	InitializeKB()
	InitializeK3s()
	InitializeGoalManager()

	if !*dryRun {
		InitializeMeasureCollector("localhost:5672")
		go RunMeasureCollector(chanFromMeasure)
	}

	go RunScheduler(chanTriggerScheduler, dryRun)
	go RunKnowledgebase(chanFromMeasure, chanTriggerScheduler)
	createRouter()
}
