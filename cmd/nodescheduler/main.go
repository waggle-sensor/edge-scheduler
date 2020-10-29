package main

func main() {
	InfoLogger.Printf("initializing...")
	InitializeValidator()
	InitializeJobManager()

	// dryRun := flag.Bool("dry-run", false, "To emulate scheduler")
	// flag.Parse()
	go RunValidator()
	go RunJobManager()
	createRouter()
}
