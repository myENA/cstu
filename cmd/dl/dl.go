package dl

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/xanzy/go-cloudstack/cloudstack"
	"os"
)

const (
	synopsisMessage = "Download a CloudStack template"
)

var err error
var cs *cloudstack.CloudStackClient

type options struct {
	apiURL, apiKey, apiSecret, templateName, templateID, zoneID string
	debug                                                       bool
}

type Command struct {
	Self  string
	Log   zerolog.Logger
	args  *options
	cfs   *flag.FlagSet
	tData *cloudstack.Template
}

func (c *Command) setupFlags(args []string) error {
	// init config if needed
	if c.args == nil {
		c.args = new(options)
	}
	c.cfs = flag.NewFlagSet("upload", flag.ExitOnError)
	c.cfs.StringVar(&c.args.apiURL, "url", os.Getenv("CLOUDSTACK_API_URL"), "CloudStack API URL [$CLOUDSTACK_API_URL]")
	c.cfs.StringVar(&c.args.apiKey, "api-key", os.Getenv("CLOUDSTACK_API_KEY"), "CloudStack API KEY [$CLOUDSTACK_API_KEY]")
	c.cfs.StringVar(&c.args.apiSecret, "api-secret", os.Getenv("CLOUDSTACK_SECRET_KEY"), "CloudStack API URL [$CLOUDSTACK_SECRET_KEY]")
	c.cfs.StringVar(&c.args.templateName, "template", "", "CloudStack template to download")
	c.cfs.StringVar(&c.args.templateID, "templateID", "", "CloudStack template id to download")
	c.cfs.StringVar(&c.args.zoneID, "zoneID", "", "CloudStack zone id")
	c.cfs.BoolVar(&c.args.debug, "debug", false, "Enable debug logs")

	if c.args.debug {
		c.Log.Level(zerolog.DebugLevel)
	}

	if c.cfs.NArg() == 0 {

	}

	return c.cfs.Parse(args)
}

func (c *Command) Run(args []string) int {
	if err := c.setupFlags(args); err != nil {
		c.Log.Error().Msgf("%s", err)
		return 1
	}

	if err := c.checkRequired(); err != nil {
		c.Log.Error().Msgf("%s", err)
		return 1
	}

	cs = cloudstack.NewAsyncClient(c.args.apiURL, c.args.apiKey, c.args.apiSecret, false)

	if c.args.templateName != "" {

		if err := c.setTemplateID(); err != nil {
			c.Log.Error().Msgf("%s", err)
			return 1
		}
	}

	if c.args.templateID != "" {
		c.tData, err = c.getTemplateData()

		if err != nil {
			c.Log.Error().Msgf("%s", err)
			return 1
		}

		c.args.templateName = c.tData.Name

	}

	if err := c.extractRequest(); err != nil {
		c.Log.Error().Msgf("%s", err)
		return 1
	}

	if err := c.writeTemplateYAML(); err != nil {
		c.Log.Error().Msgf("%s", err)
		return 1
	}

	return 0
}

func (c *Command) Synopsis() string {
	return synopsisMessage
}

func (c *Command) Help() string {

	if c.cfs == nil {
		c.setupFlags(nil)
	}

	b := &bytes.Buffer{}
	c.cfs.SetOutput(b)
	c.cfs.Usage()
	return b.String()
}
func (c *Command) checkRequired() error {
	if c.args.apiURL == "" {
		c.Help()
		return fmt.Errorf("--url must be either passed or set with envar $CLOUDSTACK_API_URL")
	}

	if c.args.apiKey == "" {
		return fmt.Errorf("--api-key must be either passed or set with envar $CLOUDSTACK_API_KEY")
	}
	if c.args.apiSecret == "" {
		return fmt.Errorf("--secret-key must be either passed or set with envar $CLOUDSTACK_SECRET_KEY")
	}

	if c.args.templateName == "" && c.args.templateID == "" {
		c.Log.Error().Msg(c.Help())
		return fmt.Errorf("--template name or --templateID must be passed")
	}

	if c.args.zoneID == "" {
		return fmt.Errorf("--zoneID must be passed")
	}

	return nil
}
