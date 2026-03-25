package main

import (
	"os"
	"time"

	"github.com/bytetwiddler/witti"
)

func main() {
	exitCode := witti.Run(os.Args[1:], os.Stdout, os.Stderr, time.Now, time.Local)
	os.Exit(exitCode)
}
