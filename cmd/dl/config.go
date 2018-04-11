package dl

import (
	"fmt"
	"github.com/myENA/cstu/cmd/upload"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

func (c *Command) writeTemplateYAML() error {
	templ := &upload.Options{
		Name:            c.args.templateName,
		APIURL:          c.args.apiURL,
		APIKey:          c.args.apiKey,
		APISecret:       c.args.apiSecret,
		HostIP:          "",
		TemplateFile:    fmt.Sprintf("./%s.qcow2", c.args.templateName),
		OSType:          c.tData.Ostypename,
		Zone:            c.tData.Zonename,
		Format:          c.tData.Format,
		HyperVisor:      c.tData.Hypervisor,
		DisplayText:     c.tData.Displaytext,
		IsPublic:        c.tData.Ispublic,
		IsFeatured:      c.tData.Isfeatured,
		PasswordEnabled: c.tData.Passwordenabled,
		IsDynamic:       c.tData.Isdynamicallyscalable,
		IsExtractable:   c.tData.Isextractable,
		IsRouting:       false,
		RequiresHVM:     false,
		SSHKeyEnabled:   c.tData.Sshkeyenabled,
		ProjectID:       c.tData.Projectid,
		TemplateTag:     c.tData.Templatetag,
	}

	templateYAML, err := yaml.Marshal(templ)

	if err != nil {
		return err
	}

	if err := ioutil.WriteFile("template.yml", templateYAML, 0766); err != nil {
		return err
	}

	return nil
}
