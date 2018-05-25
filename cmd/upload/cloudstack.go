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

	template, _, err := cs.Template.GetTemplateByName(templateName, "all", zoneID)

	if err != nil {
		c.Log.Info().Msgf("%s", err)
		return nil
	}

	if template == nil {
		return nil
	}

	return template
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
	var watched int
	watchLimit := 20
	for watch {

		if watched == watchLimit {
			return errors.New("Template is taking longer than expected to upload, cancelling upload")
		}

		templ, _, err := cs.Template.GetTemplateByID(templateID, "all")

		if err != nil {
			return err
		}

		c.Log.Info().Msgf("Checking if template %s is ready: %t status: %s", c.args.Name, templ.Isready, templ.Status)

		if !templ.Isready {
			if !strings.Contains(templ.Status, "Downloaded") && templ.Status != "" && templ.Status != "Installing Template" && templ.Status != "Download Complete" {
				return fmt.Errorf("connection error to %s with status: %s, please check the url and try again", c.urlPath, templ.Status)
			}

			watch = true
			watched += 1
			time.Sleep(sleepTimer * time.Second)

		} else {
			watch = false
		}

	}

	return nil
}

func (c *Command) deleteExistingTemplate(cs *cloudstack.CloudStackClient, existing string) {

	delParams := cs.Template.NewDeleteTemplateParams(existing)

	delResp, err := cs.Template.DeleteTemplate(delParams)

	if err != nil {
		c.Log.Error().Msgf("Error deleting template id %s: %s", existing, err)
	}

	c.Log.Info().Msgf("Response: %+v", delResp.Success)

	if delResp.Success {

		c.Log.Error().Msgf("deleting the existing template failed, please manually delete it from CloudStack: %s", delResp.Displaytext)
	} else {
		c.Log.Info().Msgf("Template deleted: %s", delResp.Displaytext)
	}

}
