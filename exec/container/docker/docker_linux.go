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
	"github.com/chaosblade-io/chaosblade-exec-cri/exec/container"
)

func (c *Client) ExecContainer(containerId, command string) (output string, err error) {
	id, err, _ := c.GetPidById(containerId)
	if err != nil {
		return "", err
	}
	return container.ExecContainer(id, command)
}

// CopyToContainer copies a tar file to the dstPath.
// If the same file exits in the dstPath, it will be override if the override arg is true, otherwise not
func (c *Client) CopyToContainer(containerId, srcFile, dstPath, extractDirName string, override bool) error {
	id, err, _ := c.GetPidById(containerId)
	if err != nil {
		return err
	}
	return container.CopyToContainer(uint32(id), srcFile, dstPath, extractDirName, override)
}
