package upload

import (
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"net/http"
	"os"
	"time"
)

func (c *Command) newDockerClient() *client.Client {
	cli, err := client.NewEnvClient()

	if err != nil {
		c.Log.Error().Msgf("Error establishing docker client: %s", err)
		os.Exit(1)
	}

	return cli
}

func (c *Command) mvTemplateForWeb() error {
	return os.Rename(c.args.TemplateFile, fmt.Sprintf("%s/%s.qcow2", webPath, c.args.Name))
}

func (c *Command) runWebContainer() error {
	cli := c.newDockerClient()

	c.Log.Info().Msgf("Creating httpd container")
	createResp, err := c.createContainer()

	if err != nil {
		c.Log.Info().Msgf("%s", err)
		c.Log.Info().Msgf("%s", createResp.ID)
		return err
	}

	c.Log.Info().Msgf("Running web server for upload: %s", c.urlPath)
	if err := cli.ContainerStart(ctx, createResp.ID, types.ContainerStartOptions{}); err != nil {
		c.Log.Error().Err(err)
		return err
	}

	c.cID = createResp.ID

	return nil
}

func (c *Command) pullHttpd() error {
	cli := c.newDockerClient()

	c.Log.Info().Msg("Pulling httpd:alpine")

	pullOpts := types.ImagePullOptions{All: true}

	responseBody, err := cli.ImagePull(ctx, "httpd:alpine", pullOpts)
	defer responseBody.Close()

	if err != nil {
		c.Log.Error().Msgf("Error pulling httpd:alpine: %s", err)
		return err
	}

	return nil

}

func (c *Command) createContainer() (container.ContainerCreateCreatedBody, error) {
	cli := c.newDockerClient()

	config := &container.Config{
		Image: "httpd:alpine",
		ExposedPorts: nat.PortSet{
			"80/tcp": struct{}{},
		},
	}

	hostConfig := &container.HostConfig{
		Binds: []string{fmt.Sprintf("%s:/usr/local/apache2/htdocs/", webPath)},
		PortBindings: nat.PortMap{
			"80/tcp": []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: "80",
				},
			},
		},
	}

	return cli.ContainerCreate(ctx, config, hostConfig, nil, containerName)
}

func (c *Command) deleteContainer(containerID string) error {
	cli := c.newDockerClient()

	return cli.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{Force: true})
}

func (c *Command) containerActive() error {
	wait := true

	for wait {
		h := &http.Client{}
		checkURL := fmt.Sprintf("http://%s", c.args.HostIP)
		_, err := h.Get(checkURL)

		// The web container will refuse connection until it is ready
		if err != nil {
			c.Log.Info().Msg("Web container not ready, if this is taking unusually long please Ctrl+c and retry")
			wait = true
			time.Sleep(2 * time.Second)
		} else {
			c.Log.Info().Msg("Web container ready! Continuing")
			wait = false
		}
	}

	return nil
}
