//go:build darwin

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
	"os"
	"strings"

	"github.com/chaosblade-io/chaosblade-spec-go/log"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/stdcopy"
)

// execContainer with command which does not contain "sh -c" in the target container
func execContainerWithConf(ctx context.Context, containerId, command string, config types.ExecConfig, c *Client) (output string, err error) {
	log.Infof(ctx, "execute command: %s", strings.Join(config.Cmd, " "))
	id, err := c.client.ContainerExecCreate(ctx, containerId, config)
	if err != nil {
		log.Warnf(ctx, "Create exec for container: %s, err: %s", containerId, err.Error())
		return "", err
	}
	resp, err := c.client.ContainerExecAttach(ctx, id.ID, types.ExecStartCheck{})
	if err != nil {
		log.Warnf(ctx, "Attach exec for container: %s, err: %s", containerId, err.Error())
		return "", err
	}
	defer resp.Close()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	_, err = stdcopy.StdCopy(stdout, stderr, resp.Reader)
	if err != nil {
		log.Warnf(ctx, "Attach exec for container: %s, err: %s", containerId, err.Error())
		return "", err
	}
	result := stdout.String()
	errorMsg := stderr.String()
	log.Debugf(ctx, "execute result: %s, error msg: %s", result, errorMsg)
	if errorMsg != "" {
		return "", errors.New(errorMsg)
	} else {
		return result, nil
	}
}

func (c *Client) ExecContainer(ctx context.Context, containerId, command string) (output string, err error) {
	return execContainerWithConf(ctx, containerId, command, types.ExecConfig{
		AttachStderr: true,
		AttachStdout: true,
		Cmd:          []string{"sh", "-c", command},
		Privileged:   true,
		User:         "root",
	}, c)
}

// CopyToContainer copies a tar file to the dstPath.
// If the same file exits in the dstPath, it will be override if the override arg is true, otherwise not
func (c *Client) CopyToContainer(ctx context.Context, containerId, srcFile, dstPath, extractDirName string, override bool) error {
	// must be a tar file
	options := types.CopyToContainerOptions{
		AllowOverwriteDirWithFile: override,
		CopyUIDGID:                true,
	}
	_, err := c.ExecContainer(ctx, containerId, fmt.Sprintf("mkdir -p %s", dstPath))
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
