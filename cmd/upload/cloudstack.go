package upload

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/xanzy/go-cloudstack/cloudstack"
	"strings"
	"time"
)

func (c *Command) getOSID(cs *cloudstack.CloudStackClient, osName string) (string, error) {
	osID := ""
	typesParams := &cloudstack.ListOsTypesParams{}
	typesParams.SetDescription(osName)
	osTypes, err := cs.GuestOS.ListOsTypes(typesParams)

	if err != nil {
		return "", err
	}

	if osTypes.Count == 0 {
		return "", errors.New("No OS Types were returned")
	}

	for _, o := range osTypes.OsTypes {
		if o.Description == osName {
			osID = o.Id
			break
		}
	}

	if osID == "" {
		osTypes, err := cs.GuestOS.ListOsTypes(nil)

		if err != nil {
			return "", err
		}
		return "", fmt.Errorf("availabe os types are %v", osTypes.OsTypes)
	}

	return osID, nil
}

func (c *Command) checkTemplateExists(cs *cloudstack.CloudStackClient, templateName, zoneID string) *cloudstack.Template {

	templates, count, err := cs.Template.GetTemplateByName(templateName, "all", zoneID)

	if err != nil {
		c.Log.Info().Msgf("%s", err)
		return nil
	}

	if count == 0 {
		return nil
	}

	return templates
}

func (c *Command) registerTemplate(cs *cloudstack.CloudStackClient) (*cloudstack.RegisterTemplateResponse, error) {

	templateURL := fmt.Sprintf("%s/%s.qcow2", c.urlPath, c.args.Name)

	c.Log.Info().Msgf("Registering template at url: %s", templateURL)

	regParams := cs.Template.NewRegisterTemplateParams(c.args.DisplayText, strings.ToUpper(c.args.Format),
		c.args.HyperVisor, c.args.Name, c.args.osID, templateURL, c.args.zoneID)
	regParams.SetIspublic(c.args.IsPublic)
	regParams.SetIsfeatured(c.args.IsFeatured)
	regParams.SetPasswordenabled(c.args.PasswordEnabled)
	regParams.SetIsrouting(c.args.IsRouting)
	regParams.SetRequireshvm(c.args.RequiresHVM)
	regParams.SetIsdynamicallyscalable(c.args.IsDynamic)
	regParams.SetIsextractable(c.args.IsExtractable)
	if c.args.ProjectID != "" {
		regParams.SetProjectid(c.args.ProjectID)
	}
	regParams.SetSshkeyenabled(c.args.SSHKeyEnabled)

	regParams.SetIspublic(true)

	return cs.Template.RegisterTemplate(regParams)

}

func (c *Command) watchRegisteredTemplate(cs *cloudstack.CloudStackClient, templateID string) error {
	var watch = true

	for watch {

		templ, _, err := cs.Template.GetTemplateByID(templateID, "all")

		if err != nil {
			return err
		}

		c.Log.Info().Msgf("Checking if template %s is ready: %t", c.args.Name, templ.Isready)

		if !templ.Isready {

			if strings.Contains(templ.Status, "refused") {
				c.Log.Info().Msgf("Connection refused to %s, please check the url and try again", c.urlPath)
				return fmt.Errorf("connection refused to %s", c.urlPath)
			}

			watch = true
			time.Sleep(sleepTimer * time.Second)
		} else {
			watch = false
		}

	}

	return nil
}

func (c *Command) deleteExistingTemplate(cs *cloudstack.CloudStackClient, existing string) error {

	delParams := cs.Template.NewDeleteTemplateParams(existing)

	delResp, err := cs.Template.DeleteTemplate(delParams)

	if err != nil {
		return err
	}

	if !delResp.Success {
		return fmt.Errorf("deleting the existing template failed, please maually delete it from CloudStack: %s", delResp.Displaytext)
	}

	c.Log.Info().Msgf("Template deleted: %s", delResp.Displaytext)

	return nil

}
