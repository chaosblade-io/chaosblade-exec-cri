//go:build linux

/*
 * Copyright 2025 The ChaosBlade Authors
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

	"github.com/chaosblade-io/chaosblade-exec-cri/exec/container"
)

func (c *Client) ExecContainer(ctx context.Context, containerId, command string) (output string, err error) {
	id, err, _ := c.GetPidById(ctx, containerId)
	if err != nil {
		return "", err
	}
	return container.ExecContainer(ctx, id, command)
}

// CopyToContainer copies a tar file to the dstPath.
// If the same file exits in the dstPath, it will be override if the override arg is true, otherwise not
func (c *Client) CopyToContainer(ctx context.Context, containerId, srcFile, dstPath, extractDirName string, override bool) error {
	id, err, _ := c.GetPidById(ctx, containerId)
	if err != nil {
		return err
	}
	return container.CopyToContainer(ctx, uint32(id), srcFile, dstPath, extractDirName, override)
}
