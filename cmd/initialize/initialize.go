package initialize

import (
	"github.com/myENA/cstu/cmd/upload"
	"github.com/rs/zerolog"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

const (
	synopsisMessage = "Writes a blank template.yml"
	helpMessage     = "ctsu init creates a blank template config file for the upload command"
)

// Command represents the upload subcommand
type Command struct {
	Self string
	Log  zerolog.Logger
}

func (c *Command) Run(args []string) int {
	configYaml := &upload.Options{}
	configYaml.IsExtractable = true
	config, err := yaml.Marshal(configYaml)

	if err != nil {
		c.Log.Error().Err(err)
		return 1
	}

	if err := ioutil.WriteFile("template.yml", config, 0766); err != nil {
		c.Log.Error().Err(err)
		return 1
	}

	return 0
}

func (c *Command) Synopsis() string {
	return synopsisMessage
}

func (c *Command) Help() string {
	return helpMessage

}
