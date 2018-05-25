package cmd

import (
	"fmt"
	"github.com/rs/zerolog"
	"net"
	"strings"
)

type CloudstackEnvironment struct {
	Name      string   `yaml:"name"`
	APIURL    string   `yaml:"apiURL"`
	APISecret string   `yaml:"apiSecret"`
	APIKey    string   `yaml:"apiKey"`
	Zones     []string `yaml:"zones"`
}

type TemplateYAML struct {
	Name            string                  `yaml:"name"`
	CSEnvironments  []CloudstackEnvironment `yaml:"environments"`
	HostIP          string                  `yaml:"hostIP"`
	TemplateFile    string                  `yaml:"templateFile"`
	TemplateID      string                  `yaml:"templateID"`
	OSType          string                  `yaml:"osType"`
	Format          string                  `yaml:"format"`
	HyperVisor      string                  `yaml:"hypervisor"`
	DisplayText     string                  `yaml:"displayText"`
	IsPublic        bool                    `yaml:"isPublic"`
	IsFeatured      bool                    `yaml:"isFeatured"`
	PasswordEnabled bool                    `yaml:"passwordEnabled"`
	IsDynamic       bool                    `yaml:"isDynamic"`
	IsExtractable   bool                    `yaml:"isExtractable"`
	IsRouting       bool                    `yaml:"isRouting"`
	RequiresHVM     bool                    `yaml:"requiresHVM"`
	SSHKeyEnabled   bool                    `yaml:"sshKeyEnabled"`
	ProjectID       string                  `yaml:"projectID,omitempty"`
	TemplateTag     string                  `yaml:"templateTag"`
}

// Get preferred outbound ip of this machine
// https://stackoverflow.com/questions/23558425/how-do-i-get-the-local-ip-address-in-go
func GetOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		logger := zerolog.Logger{}
		logger.Fatal().Msgf("%s", err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String()
}

func (t *TemplateYAML) CheckRequired() error {

	for _, e := range t.CSEnvironments {
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

	if t.Name == "" {
		return fmt.Errorf("--Name must  be passed")
	}

	if t.DisplayText == "" {
		return fmt.Errorf("--DisplayText must be passed")
	}

	if t.Format == "" {
		return fmt.Errorf("--Format must be passed")
	}

	if !strings.ContainsAny("QCOW2 RAW VHD OVA", strings.ToUpper(t.Format)) {
		return fmt.Errorf("supported formats are QCOW2, RAW, VHD and OVA")
	}

	if t.HyperVisor == "" {
		return fmt.Errorf("--hypervisor must be passed")
	}

	if t.OSType == "" {
		return fmt.Errorf("--os must be passed")
	}

	if t.HostIP == "" {
		return fmt.Errorf("--host-ip must be passed and must be reachable by cloudstack")
	}

	return nil
}
