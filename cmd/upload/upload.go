package upload

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/myENA/cstu/cmd"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/xanzy/go-cloudstack/cloudstack"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

const (
	synopsisMessage = "Uploads templates to cloudstack"
	webPath         = "/opt/cows"
	containerName   = "templateWeb"
	sleepTimer      = 15
)

var ctx context.Context

type Options struct {
	cmd.TemplateYAML `yaml:",inline"`
	osID             string
	zoneID           string
	configFile       string
	debug            bool
	cleanup          bool
	system           bool
	setResourceTags  bool
	resourceTags     string
}

// Command represents the upload subcommand
type Command struct {
	Self    string
	Log     zerolog.Logger
	args    *Options
	urlPath string
	cID     string
	cfs     *flag.FlagSet
}

func init() {
	ctx = context.Background()
}

func (c *Command) setupFlags(args []string) error {
	// init config if needed
	if c.args == nil {
		c.args = new(Options)
	}

	c.cfs = flag.NewFlagSet("upload", flag.ExitOnError)
	c.cfs.StringVar(&c.args.configFile, "configFile", "", "Template yaml file")
	c.cfs.BoolVar(&c.args.cleanup, "cleanup", false, "Remove the template from webroot after cleanup")
	c.cfs.BoolVar(&c.args.debug, "debug", false, "Enable debug logs")
	c.cfs.BoolVar(&c.args.system, "system-service", false, "Use the system httpd service. Must still have a directory at /opt/cows")

	// always okay
	return c.cfs.Parse(args)

}

func (c *Command) requiredPassed() error {

	for _, e := range c.args.CSEnvironments {
		if len(e.Zones) == 0 {
			return fmt.Errorf("no zones for %s has not been set, please either specify a list of zones", e.Name)
		}
		if e.APIURL == "" {
			return fmt.Errorf("api url must was not passed for %s environment", e.Name)
		}

		if e.APIKey == "" {
			return fmt.Errorf("api key must was not passed for %s environment", e.Name)
		}
		if e.APISecret == "" {
			return fmt.Errorf("api secret must was not passed for %s environment", e.Name)
		}
	}

	if c.args.Name == "" {
		return fmt.Errorf("--Name must  be passed")
	}

	if c.args.DisplayText == "" {
		return fmt.Errorf("--DisplayText must be passed")
	}

	if c.args.Format == "" {
		return fmt.Errorf("--Format must be passed")
	}

	if !strings.ContainsAny("QCOW2 RAW VHD OVA", strings.ToUpper(c.args.Format)) {
		return fmt.Errorf("supported formats are QCOW2, RAW, VHD and OVA")
	}

	if c.args.HyperVisor == "" {
		return fmt.Errorf("--hypervisor must be passed")
	}

	if c.args.OSType == "" {
		return fmt.Errorf("--os must be passed")
	}

	if c.args.HostIP == "" {
		return fmt.Errorf("--host-ip must be passed and must be reachable by cloudstack")
	}

	return nil
}

// Run the upload command
func (c *Command) Run(args []string) int {

	if err := c.setupFlags(args); err != nil {
		c.Log.Error().Msgf("%s", err)
		return 1
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	signal.Notify(sigChan, syscall.SIGTERM)

	errChan := make(chan int, 10)

	go func() {
		errChan <- c.register(args)
	}()

	select {
	case sig := <-sigChan:
		c.Log.Info().Msgf("Received interrupt signal: %v", sig)
		c.Log.Info().Msg("Cleaning up container")
		if err := c.deleteContainer(c.cID); err != nil {
			return 1
		}

		c.Log.Info().Msg("Moving template back")
		if err := os.Rename(fmt.Sprintf("%s/%s.qcow2", webPath, c.args.Name), c.args.TemplateFile); err != nil {
			c.Log.Error().Msgf("Failed moving %s.qcow2 back to %s", c.args.Name, c.args.TemplateFile)
			return 1
		}

	case err := <-errChan:
		if err != 0 {
			log.Printf("Error received: %v", err)
			os.Exit(1)
		}
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

func (c *Command) register(args []string) int {
	var err error

	if c.args.configFile == "" {
		c.Log.Error().Msg("A config file must be passed")
		return 1
	}

	c.Log.Info().Msgf("Reading config file at %s", c.args.configFile)
	configFile, err := ioutil.ReadFile(c.args.configFile)

	if err != nil {
		c.Log.Error().Msgf("%s", err)
		return 1
	}

	if err := yaml.Unmarshal(configFile, &c.args); err != nil {
		c.Log.Error().Msgf("%s", err)
		return 1
	}

	// Make sure required Options are set
	if err = c.requiredPassed(); err != nil {
		c.Log.Error().Msg(err.Error())
		return 1
	}

	if c.args.debug {
		c.Log.Level(zerolog.DebugLevel)
	}

	if _, err := os.Stat(c.args.TemplateFile); os.IsNotExist(err) {
		c.Log.Error().Msgf("Cannot find %s: %s", c.args.TemplateFile, err)
		return 1
	}

	if err := c.templateToWebRoot(); err != nil {
		c.Log.Error().Msgf("%s", err)
		return 1
	}

	hostIP := cmd.GetOutboundIP()

	if hostIP != c.args.HostIP {
		c.args.HostIP = hostIP
	}

	c.urlPath = fmt.Sprintf("http://%s", c.args.HostIP)

	for _, e := range c.args.CSEnvironments {

		var exists = false
		var existingID string

		c.Log.Debug().Msgf("APIUrl: %s", e.APIURL)
		c.Log.Debug().Msgf("APIKey: %s", e.APIKey)
		c.Log.Debug().Msgf("SecretKey: %s", e.APISecret)

		cs := cloudstack.NewAsyncClient(e.APIURL, e.APIKey, e.APISecret, false)

		c.Log.Info().Msgf("Getting os id for %s", c.args.OSType)
		c.args.osID, err = c.getOSID(cs, c.args.OSType)

		if err != nil {
			c.Log.Error().Msgf("%s", err)
			return 1
		}

		c.Log.Info().Msgf("Zones %d", len(e.Zones))

		for _, z := range e.Zones {

			c.Log.Info().Msgf("Getting Zone id for %s", z)
			c.args.zoneID, _, err = cs.Zone.GetZoneID(z)

			if err != nil {
				c.Log.Error().Msgf("%s", err)
				return 1
			}

			c.Log.Info().Msgf("Checking if template %s exists", c.args.Name)
			templ := c.checkTemplateExists(cs, c.args.Name, c.args.zoneID)

			if templ != nil {
				c.Log.Info().Msgf("Found a template with the same Name, saving ID %s for deletion later", templ.Id)
				c.args.TemplateTag = templ.Templatetag
				exists = true
				existingID = templ.Id
			}

			if !c.args.system {
				if errCode := c.startWebContainer(); errCode != 0 {
					return errCode
				}
			} else {
				if err := checkSystemHTTPPort(); err != nil {
					c.Log.Error().Msgf("Error checking host http service port: %s. Trying to start docker container")
					c.args.system = false
					if errCode := c.startWebContainer(); errCode != 0 {
						return errCode
					}
				}
			}

			newTempl, err := c.registerTemplate(cs)

			if err != nil {
				c.Log.Error().Msgf("%s", err)

				if !c.args.system {
					c.Log.Info().Msg("Cleaning up docker container")
					if err := c.deleteContainer(c.cID); err != nil {
						c.Log.Error().Msgf("%s", err)
						return 1
					}
				}
				return 1
			}

			var newID string

			c.Log.Info().Msg("Grabbing new template ID")
			for _, t := range newTempl.RegisterTemplate {
				if t.Name == c.args.Name && t.Id != existingID {
					newID = t.Id
					break
				}
			}

			c.Log.Info().Msgf("Waiting for new template to be ready")
			if err := c.watchRegisteredTemplate(cs, newID); err != nil {
				c.Log.Error().Msgf("%s", err)
				if !c.args.system {
					c.Log.Info().Msg("Cleaning up docker container")
					if err := c.deleteContainer(c.cID); err != nil {
						c.Log.Error().Msgf("%s", err)
						return 1
					}
				}
				return 1
			}

			if c.args.ResourceTags != nil {
				c.Log.Info().Msgf("Creating resource tags for the new template: %s", c.args.Name)
				if err := c.createResourceTags(cs, newID); err != nil {
					c.Log.Error().Msgf("%s", err)
					return 1
				}
			}

			if exists {
				c.Log.Info().Msgf("Deleting old template id %s", existingID)
				c.deleteExistingTemplate(cs, existingID)
			}

			if !c.args.system {
				c.Log.Info().Msg("Stopping the httpd container")
				if err := c.deleteContainer(c.cID); err != nil {
					c.Log.Error().Msgf("%s", err)
					return 1
				}
			}
			c.Log.Info().Msgf("Your new Template %s with ID %s is ready for use", c.args.Name, newID)

		}

	}

	if c.args.cleanup {
		if err := os.Remove(fmt.Sprintf("%s/%s.qcow2", webPath, c.args.Name)); err != nil {
			c.Log.Error().Msgf("Could not cleanup the template from the %s, please remove manually: %s",
				fmt.Sprintf("%s/%s.qcow2", webPath, c.args.Name), err)
			return 0
		}
	}

	return 0
}

func checkSystemHTTPPort() error {
	client := http.Client{}

	resp, err := client.Get(fmt.Sprintf("http://%s", cmd.GetOutboundIP()))

	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return errors.New("unable to verify http service is running on port 80")
	}

	return nil

}

func (c *Command) startWebContainer() int {
	if err := c.pullHttpd(); err != nil {
		return 1
	}

	if err := c.runWebContainer(); err != nil {
		c.Log.Error().Msgf("%s", err)
		return 1
	}

	c.Log.Info().Msg("Waiting for container to be active")
	if err := c.containerActive(); err != nil {
		return 1
	}

	return 0
}
