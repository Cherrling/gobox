package main

import (
	"os"

	"gobox/applets"
	// Import applet packages to register them
)

func main() {
	os.Exit(applets.Dispatch(os.Args))
}
