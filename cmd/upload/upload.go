package upload

import (
	"context"
	"flag"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/xanzy/go-cloudstack/cloudstack"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"strings"
)

const (
	synopsisMessage = "Uploads templates to cloudstack"
	helpMessage     = `CloudStack template updater
	--url			CloudStack API URL [$CLOUDSTACK_API_URL] [required]
	--api-key		CloudStack API KEY [$CLOUDSTACK_API_KEY] [required]
	--secret-key	CloudStack API URL [$CLOUDSTACK_SECRET_KEY] [required]
	--host-ip		The host IP address, must be reachable by CloudStack. Used for the registration URL [required]
	--template-path	File path to the template file to upload to CloudStack [required]
	--Name 			Cloudstack template Name [required]
	--DisplayText	CloudStack display test [required]
	--Format		CloudStack template Format (QCOW2, RAW, VHD and OVA) [required]
	--hypervisor	The target hypervisor for the template [required]
	--os			The OS for the template, searches for OS to get the os type id returns error if not found [required]
 	--tag			Cloudstack template tag
	--Zone			Cloudstack template Zone, searches CloudStack to get Zone ID
	--isPublic		Set the template to public
	--isFeatured	Set the template to featured
	--passwdEnabled	Set the template to Password Enabled
	--isDynamic		Set the template as dynamically scalable
	--extractable	Set the template as extractable
	--isRouting		Set the template as a routing template, ie. if template is used to deploy router
	--isRouting		Set the template as a routing template, ie. if template is used to deploy router
	--hvm			Set if template requires HVM
	--sshKeyEnabled	Set if template supports the sshkey upload feature
	--projectID		Register template for the project
	--configFile	Path to a config file instead of passing Options
	--debug		Enable debugging
`
	webPath       = "/opt/cows"
	containerName = "templateWeb"
	sleepTimer    = 15
)

var err error

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
	debug bool
}

// Command represents the upload subcommand
type Command struct {
	Self    string
	Log     zerolog.Logger
	args    *Options
	urlPath string
	cID     string
}

func (c *Command) setupFlags(args []string) error {
	var cmdFlags *flag.FlagSet
	// init config if needed
	if c.args == nil {
		c.args = new(Options)
	}
	cmdFlags = flag.NewFlagSet("upload", flag.ExitOnError)
	cmdFlags.Usage = func() { fmt.Fprint(os.Stdout, c.Help()); os.Exit(0) }

	cmdFlags.StringVar(&c.args.APIURL, "url", os.Getenv("CLOUDSTACK_API_URL"), "CloudStack API URL [$CLOUDSTACK_API_URL]")
	cmdFlags.StringVar(&c.args.APIKey, "api-key", os.Getenv("CLOUDSTACK_API_KEY"), "CloudStack API KEY [$CLOUDSTACK_API_KEY]")
	cmdFlags.StringVar(&c.args.APISecret, "api-secret", os.Getenv("CLOUDSTACK_SECRET_KEY"), "CloudStack API URL [$CLOUDSTACK_SECRET_KEY]")
	cmdFlags.StringVar(&c.args.HostIP, "host-ip", "", "The host IP address, must be reachable by CloudStack")
	cmdFlags.StringVar(&c.args.configFile, "configFile", "", "Template yaml file")
	cmdFlags.StringVar(&c.args.TemplateFile, "template-path", "", "Path to the template file to upload to CloudStack")
	cmdFlags.StringVar(&c.args.Name, "Name", "", "CloudStack template Name")
	cmdFlags.StringVar(&c.args.DisplayText, "DisplayText", "", "CloudStack display text")
	cmdFlags.StringVar(&c.args.Format, "Format", "", "CloudStack template Format ex. QCOW2")
	cmdFlags.StringVar(&c.args.HyperVisor, "hypervisor", "", "CloudStack hypervisor ex: KVM")
	cmdFlags.StringVar(&c.args.OSType, "os", "", "Template OS-type")
	cmdFlags.StringVar(&c.args.TemplateTag, "tag", "", "Template tag")
	cmdFlags.StringVar(&c.args.Zone, "zone", "", "CloudStack Zone Name")
	cmdFlags.StringVar(&c.args.ProjectID, "projectID", "", "CloudStack ProjectID")

	cmdFlags.BoolVar(&c.args.IsPublic, "isPublic", false, "Set the template to public default: false")
	cmdFlags.BoolVar(&c.args.IsDynamic, "isDynamic", false, "Set if template contains XS/VMWare tools inorder to support dynamic scaling of VM cpu/memory default: false")
	cmdFlags.BoolVar(&c.args.SSHKeyEnabled, "sshKeyEnabled", false, "Set if template supports the sshkey upload feature default: false")
	cmdFlags.BoolVar(&c.args.RequiresHVM, "hvm", false, "Set if this template requires HVM default: false")
	cmdFlags.BoolVar(&c.args.IsExtractable, "extractable", false, "Set if the template or its derivatives are extractable default: false")
	cmdFlags.BoolVar(&c.args.IsRouting, "isRouting", false, "Set if the template type is routing i.e., if template is used to deploy router default: false")
	cmdFlags.BoolVar(&c.args.IsFeatured, "isFeatured", false, "Set the template to featured default: false")
	cmdFlags.BoolVar(&c.args.debug, "debug", false, "Enable debugging logs")
	cmdFlags.BoolVar(&c.args.PasswordEnabled, "passwdEnabled", false, "Set the template to Password Enabled default: false")

	// parse Options and ignore error
	if err := cmdFlags.Parse(args); err != nil {
		return nil
	}

	// check for remaining garbage
	if cmdFlags.NArg() > 0 {
		return fmt.Errorf("unknown non-flag argument sent to command")
	}

	if c.args.debug {
		c.Log.Level(zerolog.InfoLevel)
	}
	// always okay
	return nil

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
	var exists = false

	var existingID string

	c.Log.Info().Msg("Checking Options")

	if err = c.setupFlags(args); err != nil {
		c.Log.Error().Msgf("[Error] Setup failed: %s", err.Error())
		return 1
	}

	c.Log.Info().Msgf("Config File: %s", c.args.configFile)
	if c.args.configFile != "" {

		c.Log.Info().Msgf("Reading config file at %s", c.args.configFile)
		configFile, err := ioutil.ReadFile(c.args.configFile)

		if err != nil {
			c.Log.Error().Msgf("%s",err)
			return 1
		}

		if err := yaml.Unmarshal(configFile, &c.args); err != nil {
			c.Log.Error().Msgf("%s",err)
			return 1
		}
	} else {

		// Make sure required Options are set
		if err = c.requiredPassed(); err != nil {
			log.Error().Msg(err.Error())
			return 1
		}
	}

	if _, err := os.Stat(c.args.TemplateFile); os.IsNotExist(err) {
		c.Log.Error().Msgf("Cannot find %s: %s", c.args.TemplateFile, err)
		return 1
	}

	if err := c.mvTemplateForWeb(); err != nil {
		c.Log.Error().Msgf("%s",err)
		return 1
	}

	c.urlPath = fmt.Sprintf("http://%s", c.args.HostIP)

	c.Log.Debug().Msgf("URL: %s, APIKEY: %s, SECRET: %s", c.args.APIURL, c.args.APIKey, c.args.APISecret)
	cs := cloudstack.NewClient(c.args.APIURL, c.args.APIKey, c.args.APISecret, false)

	c.Log.Info().Msgf("Getting os id for %s", c.args.OSType)
	c.args.osID, err = c.getOSID(cs, c.args.OSType)

	if err != nil {
		c.Log.Error().Msgf("%s",err)
		return 1
	}
	c.Log.Info().Msgf("Getting Zone id for %s", c.args.Zone)
	c.args.zoneID, _, err = cs.Zone.GetZoneID(c.args.Zone)

	if err != nil {
		c.Log.Error().Msgf("%s",err)
		return 1
	}

	if err := c.runWebContainer(); err != nil {
		c.Log.Error().Msgf("%s",err)
		return 1
	}

	c.Log.Info().Msgf("Checking if template %s exists", c.args.Name)

	templ := c.checkTemplateExists(cs, c.args.Name, c.args.zoneID)

	if templ != nil {
		c.Log.Info().Msg("Found a template with the same Name, saving ID for deletion later")
		exists = true
		existingID = templ.Id
	}

	newTempl, err := c.registerTemplate(cs)

	if err != nil {
		c.Log.Error().Msgf("%s", err)
		c.Log.Info().Msg("Cleaning up docker container")
		if err := c.deleteContainer(context.Background(), c.cID); err != nil {
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
		c.Log.Error().Msgf("%s",err)
		c.Log.Info().Msg("Cleaning up docker container")
		if err := c.deleteContainer(context.Background(), c.cID); err != nil {
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
	if err := c.deleteContainer(context.Background(), c.cID); err != nil {
		c.Log.Error().Msgf("%s",err)
		return 1
	}

	c.Log.Info().Msgf("Your new Template %s with ID %s is ready for use", c.args.Name, newID)
	return 0
}

func (c *Command) Synopsis() string {
	return synopsisMessage
}

func (c *Command) Help() string {
	return helpMessage

}
