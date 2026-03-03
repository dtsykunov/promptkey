//go:build debug

package main

import (
	"log"
	"os"
)

func init() {
	log.SetOutput(os.Stderr)
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Lshortfile)
}

func debugf(format string, args ...any) {
	log.Printf(format, args...)
}
