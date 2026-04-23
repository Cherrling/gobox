package main

import (
	"os"

	"gobox/applets"
	_ "gobox/applets/coreutils"
)

func main() {
	os.Exit(applets.Dispatch(os.Args))
}
