package main

import (
	"fmt"
	"github.com/mitchellh/cli"
	"os"
)
var version string
func main() {
	var status int
	var err error

	c := cli.NewCLI("cstu", version)
	c.Args = os.Args[1:]
	c.Commands = cliCommands
	if status, err = c.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error executing CLI: %s\n", err)
	}

	os.Exit(status)

}
