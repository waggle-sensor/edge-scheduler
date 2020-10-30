package util

import (
	"log"
	"os"
)

var (
	// InfoLogger logs messages verbosely
	InfoLogger = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)
	// ErrorLogger logs errors
	ErrorLogger = log.New(os.Stdout, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
)
