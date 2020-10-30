package logger

import (
	"log"
	"os"
)

var (
	// Info logs messages verbosely
	Info = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)
	// Error logs errors
	Error = log.New(os.Stdout, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
)
