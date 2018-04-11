package main

import (
	"github.com/mitchellh/cli"
	"github.com/myENA/cstu/cmd/dl"
	"github.com/myENA/cstu/cmd/initialize"
	"github.com/myENA/cstu/cmd/upload"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
)

// package global logger
var logger zerolog.Logger

// available commands
var cliCommands map[string]cli.CommandFactory

// init command factory
func init() {
	// init logger
	logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	logger.Level(zerolog.InfoLevel)

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
		"dl": func() (cli.Command, error) {
			return &dl.Command{
				Self: os.Args[0],
				Log:  logger,
			}, nil
		},
	}
}
