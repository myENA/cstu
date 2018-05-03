package cmd

import (
	"github.com/rs/zerolog"
	"net"
)

type TemplateYAML struct {
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
	ProjectID       string `yaml:"projectID,omitempty"`
	TemplateTag     string `yaml:"templateTag"`
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
