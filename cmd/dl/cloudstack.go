package dl

import (
	"fmt"
	"github.com/xanzy/go-cloudstack/cloudstack"
	"io"
	"net/http"
	"os"
)

func (c *Command) extractRequest() error {

	if !c.isExtractable() {
		return fmt.Errorf("%s is not extractable", c.args.templateName)
	}
	extractOpts := cs.Template.NewExtractTemplateParams(c.args.templateID, "HTTP_DOWNLOAD")

	extractResp, err := cs.Template.ExtractTemplate(extractOpts)

	if err != nil {
		c.Log.Error().Msgf("Error retrieving download URL: %s", err)
		return err
	}

	return c.DownloadFile(extractResp.Url, "./")
}

func (c *Command) isExtractable() bool {
	templ, _, err := cs.Template.GetTemplateByID(c.args.templateID, "all")

	if err != nil {
		c.Log.Fatal().Msgf("%s", err)
	}

	return templ.Isextractable
}

func (c *Command) setTemplateID() error {
	var err error
	c.args.templateID, _, err = cs.Template.GetTemplateID(c.args.templateName, "all", c.args.zoneID)

	return err
}

func (c *Command) getTemplateData() (*cloudstack.Template, error) {
	templ, _, err := cs.Template.GetTemplateByID(c.args.templateID, "all")

	if err != nil {
		return nil, err
	}

	return templ, nil
}

func (c *Command) DownloadFile(url string, dest string) error {
	webCLient := &http.Client{}
	var qcow string
	if c.args.templateName != "" {
		qcow = c.args.templateName
	} else {
		qcow = c.args.templateID
	}

	wr, err := os.Create(fmt.Sprintf("%s.qcow2", qcow))

	if err != nil {
		return err
	}

	defer wr.Close()

	resp, err := webCLient.Get(url)

	if err != nil {
		c.Log.Error().Msgf("Error getting template: %s", err)
		return err
	}
	defer resp.Body.Close()

	c.Log.Info().Msgf("Downloading template from %s", url)
	wrSize, err := io.Copy(wr, resp.Body)

	if err != nil {
		c.Log.Error().Msgf("Error downloading template: %s", err)
		return err
	}

	c.Log.Info().Msgf("Wrote template %s size: %d", c.args.templateName, wrSize)
	return nil
}
