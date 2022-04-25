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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/chaosblade-io/chaosblade-spec-go/spec"
	"github.com/docker/docker/api/types"
	containertype "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/versions"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/sirupsen/logrus"

	"github.com/chaosblade-io/chaosblade-exec-cri/exec/container"
)

var cli *Client

type Client struct {
	client *client.Client
	Ctx    context.Context
}

//GetClient returns the docker client
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

//checkAndCreateClient
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

func (c *Client) GetContainerById(containerId string) (container.ContainerInfo, error, int32) {
	option := types.ContainerListOptions{
		Filters: filters.NewArgs(
			filters.Arg("id", containerId),
		),
	}
	return c.GetContainerFromDocker(option)
}

//getContainerByName returns the container object by container name
func (c *Client) GetContainerByName(containerName string) (container.ContainerInfo, error, int32) {
	option := types.ContainerListOptions{
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

	return c.GetContainerFromDocker(types.ContainerListOptions{
		All:     true,
		Filters: filters.NewArgs(args...),
	})
}

func (c *Client) GetContainerFromDocker(option types.ContainerListOptions) (container.ContainerInfo, error, int32) {
	containers, err := c.client.ContainerList(context.Background(), option)

	if err != nil {
		return container.ContainerInfo{}, fmt.Errorf(spec.ContainerExecFailed.Sprintf("GetContainerList", err.Error())), spec.ContainerExecFailed.Code
	}
	if containers == nil || len(containers) == 0 {
		return container.ContainerInfo{}, fmt.Errorf(spec.ParameterInvalidDockContainerId.Sprintf("container-id")), spec.ParameterInvalidDockContainerId.Code
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

//RemoveContainer
func (c *Client) RemoveContainer(containerId string, force bool) error {
	err := c.client.ContainerRemove(context.Background(), containerId, types.ContainerRemoveOptions{
		Force: force,
	})
	if err != nil {
		logrus.Warningf("Remove container: %s, err: %s", containerId, err)
		return err
	}
	return nil
}

// CopyToContainer copies a tar file to the dstPath.
// If the same file exits in the dstPath, it will be override if the override arg is true, otherwise not
func (c *Client) CopyToContainer(containerId, srcFile, dstPath, extractDirName string, override bool) error {
	// must be a tar file
	options := types.CopyToContainerOptions{
		AllowOverwriteDirWithFile: override,
		CopyUIDGID:                true,
	}
	_, err := c.execContainerPrivileged(containerId, fmt.Sprintf("mkdir -p %s", dstPath))
	if err != nil {
		return err
	}
	file, err := os.OpenFile(srcFile, os.O_RDONLY, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	return c.client.CopyToContainer(c.Ctx, containerId, dstPath, file, options)
}
func (c *Client) execContainerPrivileged(containerId, command string) (output string, err error) {
	return c.execContainerWithConf(containerId, command, types.ExecConfig{
		AttachStderr: true,
		AttachStdout: true,
		Cmd:          []string{"sh", "-c", command},
		Privileged:   true,
		User:         "root",
	})
}

//execContainer with command which does not contain "sh -c" in the target container
func (c *Client) execContainerWithConf(containerId, command string, config types.ExecConfig) (output string, err error) {
	logrus.Infof("execute command: %s", strings.Join(config.Cmd, " "))
	ctx := context.Background()
	id, err := c.client.ContainerExecCreate(ctx, containerId, config)
	if err != nil {
		logrus.Warningf("Create exec for container: %s, err: %s", containerId, err.Error())
		return "", err
	}
	resp, err := c.client.ContainerExecAttach(ctx, id.ID, types.ExecStartCheck{})
	if err != nil {
		logrus.Warningf("Attach exec for container: %s, err: %s", containerId, err.Error())
		return "", err
	}
	defer resp.Close()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	_, err = stdcopy.StdCopy(stdout, stderr, resp.Reader)
	if err != nil {
		logrus.Warningf("Attach exec for container: %s, err: %s", containerId, err.Error())
		return "", err
	}
	result := stdout.String()
	errorMsg := stderr.String()
	logrus.Debugf("execute result: %s, error msg: %s", result, errorMsg)
	if errorMsg != "" {
		return "", fmt.Errorf(errorMsg)
	} else {
		return result, nil
	}
}

func (c *Client) ExecContainer(containerId, command string) (output string, err error) {
	return c.execContainerWithConf(containerId, command, types.ExecConfig{
		AttachStderr: true,
		AttachStdout: true,
		Cmd:          []string{"sh", "-c", command},
	})
}

func (c *Client) ExecContainerPrivileged(containerId, command string) (output string, err error) {
	return c.execContainerWithConf(containerId, command, types.ExecConfig{
		AttachStderr: true,
		AttachStdout: true,
		Cmd:          []string{"sh", "-c", command},
		Privileged:   true,
		User:         "root",
	})
}

//ExecuteAndRemove: create and start a container for executing a command, and remove the container
func (c *Client) ExecuteAndRemove(config *containertype.Config, hostConfig *containertype.HostConfig,
	networkConfig *network.NetworkingConfig, containerName string, removed bool, timeout time.Duration,
	command string, containerInfo container.ContainerInfo) (containerId string, output string, err error, code int32) {

	logrus.Debugf("command: '%s', image: %s, containerName: %s", command, config.Image, containerName)
	// check image exists or not
	_, err = c.getImageByRef(config.Image)
	if err != nil {
		// pull image if not exists
		_, err := c.pullImage(config.Image)
		if err != nil {
			return "", "", fmt.Errorf(spec.ImagePullFailed.Sprintf(config.Image, err)), spec.ImagePullFailed.Code
		}
	}
	containerId, err = c.createAndStartContainer(config, hostConfig, networkConfig, containerName)
	if err != nil {
		c.RemoveContainer(containerId, true)
		return containerId, "", fmt.Errorf(spec.ContainerExecFailed.Sprintf("CreateAndStartContainer", err)), spec.ContainerExecFailed.Code
	}

	output, err = c.ExecContainer(containerId, command)
	if err != nil {
		if removed {
			c.RemoveContainer(containerId, true)
		}
		return containerId, "", fmt.Errorf(spec.ContainerExecFailed.Sprintf("ContainerExecCmd", err)), spec.ContainerExecFailed.Code
	}
	logrus.Infof("Execute output in container: %s", output)
	if removed {
		c.RemoveContainer(containerId, true)
	}
	return containerId, output, nil, spec.OK.Code
}

//ImageExists
func (c *Client) getImageByRef(ref string) (types.ImageSummary, error) {
	args := filters.NewArgs(filters.Arg("reference", ref))
	list, err := c.client.ImageList(context.Background(), types.ImageListOptions{
		All:     false,
		Filters: args,
	})
	if err != nil {
		logrus.Warningf("Get image by name failed. name: %s, err: %s", ref, err)
		return types.ImageSummary{}, err
	}
	if len(list) == 0 {
		logrus.Warningf("Cannot find the image by name: %s", ref)
		return types.ImageSummary{}, errors.New("image not found")
	}
	return list[0], nil
}

//PullImage
func (c *Client) pullImage(ref string) (string, error) {
	reader, err := c.client.ImagePull(context.Background(), ref, types.ImagePullOptions{})
	if err != nil {
		return "", err
	}
	defer reader.Close()
	bytes, err := ioutil.ReadAll(reader)
	return string(bytes), nil
}

//createAndStartContainer
func (c *Client) createAndStartContainer(config *containertype.Config, hostConfig *containertype.HostConfig,
	networkConfig *network.NetworkingConfig, containerName string) (string, error) {
	body, err := c.client.ContainerCreate(context.Background(), config, hostConfig, networkConfig, containerName)
	if err != nil {
		logrus.Warningf("Create container: %s, err: %s", containerName, err.Error())
		return "", err
	}
	containerId := body.ID
	err = c.startContainer(containerId)
	return containerId, err
}

//startContainer
func (c *Client) startContainer(containerId string) error {
	err := c.client.ContainerStart(context.Background(), containerId, types.ContainerStartOptions{})
	if err != nil {
		logrus.Warningf("Start container: %s, err: %s", containerId, err.Error())
		return err
	}
	return nil
}
