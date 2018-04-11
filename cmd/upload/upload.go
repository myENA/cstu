package upload

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/xanzy/go-cloudstack/cloudstack"
	"gopkg.in/yaml.v2"
	"io/ioutil"
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

var err error
var ctx context.Context

type Options struct {
	Name            string `yaml:"name"`
	APIURL          string `yaml:"apiURL"`
	APISecret       string `yaml:"apiSecret"`
	APIKey          string `yaml:"apiKey"`
	HostIP          string `yaml:"hostIP"`
	TemplateFile    string `yaml:"templateFile"`
	OSType          string `yaml:"osType"`
	Zone            string `yaml:"zone"`
	Format          string `yaml:"format"`
	HyperVisor      string `yaml:"hypervisor"`
	DisplayText     string `yaml:"displayText"`
	IsPublic        bool   `yaml:"isPublic"`
	IsFeatured      bool   `yaml:"isFeatured"`
	PasswordEnabled bool   `yaml:"passwordEnabled"`
	IsDynamic       bool   `yaml:"isDynamic"`
	IsExtractable   bool   `yaml:"isExtractable"`
	IsRouting       bool   `yaml:"isRouting"`
	RequiresHVM     bool   `yaml:"requiresHVM"`
	SSHKeyEnabled   bool   `yaml:"sshKeyEnabled"`
	ProjectID       string `yaml:"projectID"`
	TemplateTag     string `yaml:"templateTag"`
	osID            string
	zoneID          string
	configFile      string
	debug           bool
	cleanup         bool
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
	c.cfs.StringVar(&c.args.APIURL, "url", os.Getenv("CLOUDSTACK_API_URL"), "CloudStack API URL [$CLOUDSTACK_API_URL]")
	c.cfs.StringVar(&c.args.APIKey, "api-key", os.Getenv("CLOUDSTACK_API_KEY"), "CloudStack API KEY [$CLOUDSTACK_API_KEY]")
	c.cfs.StringVar(&c.args.APISecret, "api-secret", os.Getenv("CLOUDSTACK_SECRET_KEY"), "CloudStack API URL [$CLOUDSTACK_SECRET_KEY]")
	c.cfs.StringVar(&c.args.HostIP, "host-ip", "", "The host IP address, must be reachable by CloudStack")
	c.cfs.StringVar(&c.args.configFile, "configFile", "", "Template yaml file")
	c.cfs.StringVar(&c.args.TemplateFile, "template-path", "", "Path to the template file to upload to CloudStack")
	c.cfs.StringVar(&c.args.Name, "Name", "", "CloudStack template Name")
	c.cfs.StringVar(&c.args.DisplayText, "DisplayText", "", "CloudStack display text")
	c.cfs.StringVar(&c.args.Format, "Format", "", "CloudStack template Format ex. QCOW2")
	c.cfs.StringVar(&c.args.HyperVisor, "hypervisor", "", "CloudStack hypervisor ex: KVM")
	c.cfs.StringVar(&c.args.OSType, "os", "", "Template OS-type")
	c.cfs.StringVar(&c.args.TemplateTag, "tag", "", "Template tag")
	c.cfs.StringVar(&c.args.Zone, "zone", "", "CloudStack Zone Name")
	c.cfs.StringVar(&c.args.ProjectID, "projectID", "", "CloudStack ProjectID")

	c.cfs.BoolVar(&c.args.IsPublic, "isPublic", false, "Set the template to public default: false")
	c.cfs.BoolVar(&c.args.IsDynamic, "isDynamic", false, "Set if template contains XS/VMWare tools inorder to support dynamic scaling of VM cpu/memory default: false")
	c.cfs.BoolVar(&c.args.SSHKeyEnabled, "sshKeyEnabled", false, "Set if template supports the sshkey upload feature default: false")
	c.cfs.BoolVar(&c.args.RequiresHVM, "hvm", false, "Set if this template requires HVM default: false")
	c.cfs.BoolVar(&c.args.IsExtractable, "extractable", true, "Set if the template or its derivatives are extractable default: false")
	c.cfs.BoolVar(&c.args.IsRouting, "isRouting", false, "Set if the template type is routing i.e., if template is used to deploy router default: false")
	c.cfs.BoolVar(&c.args.IsFeatured, "isFeatured", false, "Set the template to featured default: false")
	c.cfs.BoolVar(&c.args.debug, "debug", false, "Enable debugging logs")
	c.cfs.BoolVar(&c.args.cleanup, "cleanup", false, "Remove the template from webroot after cleanup")
	c.cfs.BoolVar(&c.args.PasswordEnabled, "passwdEnabled", false, "Set the template to Password Enabled default: false")

	if c.args.debug {
		c.Log.Level(zerolog.DebugLevel)
	}

	// always okay
	return c.cfs.Parse(args)

}

func (c *Command) requiredPassed() error {

	if c.args.Zone == "" {
		return fmt.Errorf("--zone must be passed")
	}

	if c.args.Name == "" {
		return fmt.Errorf("--Name must  be passed")
	}

	if c.args.DisplayText == "" {
		return fmt.Errorf("--DisplayText must be passed")
	}

	if c.args.Format == "" {
		return fmt.Errorf("--Format must be passed")
	} else {
		if !strings.ContainsAny("QCOW2 RAW VHD OVA", strings.ToUpper(c.args.Format)) {
			return fmt.Errorf("supported formats are QCOW2, RAW, VHD and OVA")
		}
	}

	if c.args.HyperVisor == "" {
		return fmt.Errorf("--hypervisor must be passed")
	}

	if c.args.OSType == "" {
		return fmt.Errorf("--os must be passed")
	}

	if c.args.APIURL == "" {
		return fmt.Errorf("--url must be either passed or set with envar $CLOUDSTACK_API_URL")
	}

	if c.args.APIKey == "" {
		return fmt.Errorf("--api-key must be either passed or set with envar $CLOUDSTACK_API_KEY")
	}
	if c.args.APISecret == "" {
		return fmt.Errorf("--secret-key must be either passed or set with envar $CLOUDSTACK_SECRET_KEY")
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
	signal.Notify(sigChan, syscall.SIGKILL)

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

	var exists = false
	var existingID string

	c.Log.Info().Msgf("Config File: %s", c.args.configFile)
	if c.args.configFile != "" {

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
	}

	// Make sure required Options are set
	if err = c.requiredPassed(); err != nil {
		log.Error().Msg(err.Error())
		return 1
	}

	if _, err := os.Stat(c.args.TemplateFile); os.IsNotExist(err) {
		c.Log.Error().Msgf("Cannot find %s: %s", c.args.TemplateFile, err)
		return 1
	}

	if err := c.mvTemplateForWeb(); err != nil {
		c.Log.Error().Msgf("%s", err)
		return 1
	}

	c.urlPath = fmt.Sprintf("http://%s", c.args.HostIP)

	c.Log.Debug().Msgf("URL: %s, APIKEY: %s, SECRET: %s", c.args.APIURL, c.args.APIKey, c.args.APISecret)
	cs := cloudstack.NewClient(c.args.APIURL, c.args.APIKey, c.args.APISecret, false)

	c.Log.Info().Msgf("Getting os id for %s", c.args.OSType)
	c.args.osID, err = c.getOSID(cs, c.args.OSType)

	if err != nil {
		c.Log.Error().Msgf("%s", err)
		return 1
	}
	c.Log.Info().Msgf("Getting Zone id for %s", c.args.Zone)
	c.args.zoneID, _, err = cs.Zone.GetZoneID(c.args.Zone)

	if err != nil {
		c.Log.Error().Msgf("%s", err)
		return 1
	}

	c.Log.Info().Msgf("Checking if template %s exists", c.args.Name)
	templ := c.checkTemplateExists(cs, c.args.Name, c.args.zoneID)

	if templ != nil {
		c.Log.Info().Msg("Found a template with the same Name, saving ID for deletion later")
		exists = true
		existingID = templ.Id
	}

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

	newTempl, err := c.registerTemplate(cs)

	if err != nil {
		c.Log.Error().Msgf("%s", err)
		c.Log.Info().Msg("Cleaning up docker container")
		if err := c.deleteContainer(c.cID); err != nil {
			c.Log.Error().Msgf("%s", err)
			return 1
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
		c.Log.Info().Msg("Cleaning up docker container")
		if err := c.deleteContainer(c.cID); err != nil {
			c.Log.Error().Msgf("%s", err)
			return 1
		}
		return 1
	}

	if exists {
		c.Log.Info().Msgf("Deleting old template id %s", existingID)
		c.deleteExistingTemplate(cs, existingID)
	}

	c.Log.Info().Msg("Stopping the httpd container")
	if err := c.deleteContainer(c.cID); err != nil {
		c.Log.Error().Msgf("%s", err)
		return 1
	}

	c.Log.Info().Msgf("Your new Template %s with ID %s is ready for use", c.args.Name, newID)

	if c.args.cleanup {
		if err := os.Remove(fmt.Sprintf("%s/%s.qcow2", webPath, c.args.Name)); err != nil {
			c.Log.Error().Msgf("Could not remove the template from the %s, please remove manually: %s",
				fmt.Sprintf("%s/%s.qcow2", webPath, c.args.Name), err)
			return 0
		}
	}

	return 0
}
