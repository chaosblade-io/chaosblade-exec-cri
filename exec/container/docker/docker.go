/*
 * Copyright 1999-2020 Alibaba Group Holding Ltd.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package docker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/chaosblade-io/chaosblade-spec-go/log"
	"github.com/chaosblade-io/chaosblade-spec-go/spec"
	"github.com/docker/docker/api/types"
	containertype "github.com/docker/docker/api/types/container"
	dockercontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/versions"
	"github.com/docker/docker/client"

	"github.com/chaosblade-io/chaosblade-exec-cri/exec/container"
)

var cli *Client

type Client struct {
	client *client.Client
	Ctx    context.Context
}

// GetClient returns the docker client
func NewClient(endpoint string) (*Client, error) {
	var oldClient *client.Client
	if cli != nil {
		oldClient = cli.client
	}
	client, err := checkAndCreateClient(endpoint, oldClient)
	if err != nil {
		return nil, err
	}
	cli = &Client{
		client: client,
		Ctx:    context.TODO(),
	}
	return cli, nil
}

// checkAndCreateClient
func checkAndCreateClient(endpoint string, cli *client.Client) (*client.Client, error) {
	if cli == nil {
		var err error
		if endpoint == "" {
			cli, err = client.NewClientWithOpts(client.FromEnv, client.WithVersion("1.24"))
		} else {
			cli, err = client.NewClientWithOpts(client.FromEnv, client.WithVersion("1.24"), client.WithHost(endpoint))
		}
		if err != nil {
			return nil, err
		}
	}
	return ping(cli)
}

// ping
func ping(cli *client.Client) (*client.Client, error) {
	if cli == nil {
		return nil, errors.New("client is nil")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	p, err := cli.Ping(ctx)
	if err == nil {
		return cli, nil
	}
	if p.APIVersion == "" {
		return nil, err
	}
	// if server version is lower than the client version, downgrade
	if versions.LessThan(p.APIVersion, cli.ClientVersion()) {
		client.WithVersion(p.APIVersion)(cli)
		_, err = cli.Ping(ctx)
		if err == nil {
			return cli, nil
		}
		return nil, err
	}
	return nil, err
}

func (c *Client) GetPidById(ctx context.Context, containerId string) (int32, error, int32) {
	inspect, err := c.client.ContainerInspect(context.Background(), containerId)
	if err != nil {
		return -1, errors.New(spec.ContainerExecFailed.Sprintf("GetContainerList", err.Error())), spec.ContainerExecFailed.Code
	}

	return int32(inspect.State.Pid), nil, spec.OK.Code
}

func (c *Client) GetContainerById(ctx context.Context, containerId string) (container.ContainerInfo, error, int32) {
	option := dockercontainer.ListOptions{
		Filters: filters.NewArgs(
			filters.Arg("id", containerId),
		),
	}
	return c.GetContainerFromDocker(option)
}

// getContainerByName returns the container object by container name
func (c *Client) GetContainerByName(ctx context.Context, containerName string) (container.ContainerInfo, error, int32) {
	option := dockercontainer.ListOptions{
		All: true,
		Filters: filters.NewArgs(
			filters.Arg("name", containerName),
		),
	}
	return c.GetContainerFromDocker(option)
}

func (c *Client) GetContainerByLabelSelector(labels map[string]string) (container.ContainerInfo, error, int32) {
	args := make([]filters.KeyValuePair, 0)

	for k, v := range labels {
		args = append(args, filters.Arg("label", fmt.Sprintf("%s=%s", k, v)))
	}

	return c.GetContainerFromDocker(dockercontainer.ListOptions{
		All:     true,
		Filters: filters.NewArgs(args...),
	})
}

func (c *Client) GetContainerFromDocker(option dockercontainer.ListOptions) (container.ContainerInfo, error, int32) {
	containers, err := c.client.ContainerList(context.Background(), option)
	if err != nil {
		return container.ContainerInfo{}, errors.New(spec.ContainerExecFailed.Sprintf("GetContainerList", err.Error())), spec.ContainerExecFailed.Code
	}
	if containers == nil || len(containers) == 0 {
		return container.ContainerInfo{}, errors.New(spec.ParameterInvalidDockContainerId.Sprintf("container-id")), spec.ParameterInvalidDockContainerId.Code
	}
	containerInfo := convertContainerInfo(containers[0])
	return containerInfo, nil, spec.OK.Code
}

func convertContainerInfo(container2 types.Container) container.ContainerInfo {
	return container.ContainerInfo{
		ContainerId:   container2.ID,
		ContainerName: container2.Names[0],
		Labels:        container2.Labels,
	}
}

// RemoveContainer
func (c *Client) RemoveContainer(ctx context.Context, containerId string, force bool) error {
	err := c.client.ContainerRemove(context.Background(), containerId, dockercontainer.RemoveOptions{
		Force: force,
	})
	if err != nil {
		log.Warnf(ctx, "Remove container: %s, err: %s", containerId, err)
		return err
	}
	return nil
}

// ExecuteAndRemove: create and start a container for executing a command, and remove the container
func (c *Client) ExecuteAndRemove(ctx context.Context, config *containertype.Config, hostConfig *containertype.HostConfig,
	networkConfig *network.NetworkingConfig, containerName string, removed bool,
	timeout time.Duration,
	command string, containerInfo container.ContainerInfo,
) (containerId string, output string, err error, code int32) {
	log.Debugf(ctx, "command: '%s', image: %s, containerName: %s", command, config.Image, containerName)
	// check image exists or not
	_, err = c.getImageByRef(ctx, config.Image)
	if err != nil {
		// pull image if not exists
		_, err := c.pullImage(config.Image)
		if err != nil {
			return "", "", errors.New(spec.ImagePullFailed.Sprintf(config.Image, err)), spec.ImagePullFailed.Code
		}
	}
	containerId, err = c.createAndStartContainer(ctx, config, hostConfig, networkConfig, containerName)
	if err != nil {
		c.RemoveContainer(ctx, containerId, true)
		return containerId, "", errors.New(spec.ContainerExecFailed.Sprintf("CreateAndStartContainer", err)), spec.ContainerExecFailed.Code
	}

	output, err = c.ExecContainer(ctx, containerId, command)
	if err != nil {
		if removed {
			c.RemoveContainer(ctx, containerId, true)
		}
		return containerId, "", errors.New(spec.ContainerExecFailed.Sprintf("ContainerExecCmd", err)), spec.ContainerExecFailed.Code
	}
	log.Infof(ctx, "Execute output in container: %s", output)
	if removed {
		c.RemoveContainer(ctx, containerId, true)
	}
	return containerId, output, nil, spec.OK.Code
}

// ImageExists
func (c *Client) getImageByRef(ctx context.Context, ref string) (image.Summary, error) {
	args := filters.NewArgs(filters.Arg("reference", ref))
	list, err := c.client.ImageList(context.Background(), image.ListOptions{
		All:     false,
		Filters: args,
	})
	if err != nil {
		log.Warnf(ctx, "Get image by name failed. name: %s, err: %s", ref, err)
		return image.Summary{}, err
	}
	if len(list) == 0 {
		log.Warnf(ctx, "Cannot find the image by name: %s", ref)
		return image.Summary{}, errors.New("image not found")
	}
	return list[0], nil
}

// PullImage
func (c *Client) pullImage(ref string) (string, error) {
	reader, err := c.client.ImagePull(context.Background(), ref, image.PullOptions{})
	if err != nil {
		return "", err
	}
	defer reader.Close()
	bytes, err := io.ReadAll(reader)
	return string(bytes), nil
}

// createAndStartContainer
func (c *Client) createAndStartContainer(ctx context.Context, config *containertype.Config, hostConfig *containertype.HostConfig,
	networkConfig *network.NetworkingConfig, containerName string,
) (string, error) {
	body, err := c.client.ContainerCreate(
		context.Background(),
		config,
		hostConfig,
		networkConfig,
		nil,
		containerName,
	)
	if err != nil {
		log.Warnf(ctx, "Create container: %s, err: %s", containerName, err.Error())
		return "", err
	}
	containerId := body.ID
	err = c.startContainer(ctx, containerId)
	return containerId, err
}

// startContainer
func (c *Client) startContainer(ctx context.Context, containerId string) error {
	err := c.client.ContainerStart(context.Background(), containerId, dockercontainer.StartOptions{})
	if err != nil {
		log.Warnf(ctx, "Start container: %s, err: %s", containerId, err.Error())
		return err
	}
	return nil
}
