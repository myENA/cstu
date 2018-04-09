package upload

import (
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"os"
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
	ctx := context.Background()

	c.Log.Info().Msgf("Creating httpd container")
	createResp, err := c.createContainer(ctx)

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

func (c *Command) createContainer(ctx context.Context) (container.ContainerCreateCreatedBody, error) {
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

func (c *Command) deleteContainer(ctx context.Context, containerID string) error {
	cli := c.newDockerClient()

	return cli.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{Force: true})
}
