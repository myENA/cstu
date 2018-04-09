package main

import (
	"github.com/myENA/cstu/cmd/initialize"
	"github.com/mitchellh/cli"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"github.com/myENA/cstu/cmd/upload"
)

// package global logger
var logger zerolog.Logger

// available commands
var cliCommands map[string]cli.CommandFactory

// init command factory
func init() {
	// init logger
	logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// register sub commands
	cliCommands = map[string]cli.CommandFactory{
		"upload": func() (cli.Command, error) {
			return &upload.Command{
				Self: os.Args[0],
				Log:  logger,
			}, nil
		},
		"init": func() (cli.Command, error) {
			return &initialize.Command{
				Self: os.Args[0],
				Log:  logger,
			}, nil
		},
	}
}
