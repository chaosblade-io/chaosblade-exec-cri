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
package container

import (
	"context"
	"fmt"
	"time"

	"github.com/containerd/typeurl/v2"
	containertype "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
)

const (
	ContainerdRuntime = "containerd"
	DockerRuntime     = "docker"
)

const (
	ChaosBladeImageVersion = "latest"
	DefaultImageRepo       = "registry.cn-hangzhou.aliyuncs.com/chaosblade/chaosblade-tool"
)

type Container interface {
	GetPidById(ctx context.Context, containerId string) (int32, error, int32)
	GetContainerById(ctx context.Context, containerId string) (ContainerInfo, error, int32)
	GetContainerByName(ctx context.Context, containerName string) (ContainerInfo, error, int32)
	GetContainerByLabelSelector(containerLabelSelector map[string]string) (ContainerInfo, error, int32)
	RemoveContainer(ctx context.Context, containerId string, force bool) error
	CopyToContainer(ctx context.Context, containerId, srcFile, dstPath, extractDirName string, override bool) error

	ExecContainer(ctx context.Context, containerId, command string) (output string, err error)
	ExecuteAndRemove(ctx context.Context, config *containertype.Config, hostConfig *containertype.HostConfig,
		networkConfig *network.NetworkingConfig, containerName string, removed bool, timeout time.Duration,
		command string, containerInfo ContainerInfo) (containerId string, output string, err error, code int32)
}

// ContainerInfo for server
type ContainerInfo struct {
	ContainerId   string
	ContainerName string
	Labels        map[string]string
	Spec          typeurl.Any
}

func GetChaosBladeImageRef(repo, version string) string {
	if repo == "" {
		repo = DefaultImageRepo
	}
	if version == "" {
		version = ChaosBladeImageVersion
	}
	return fmt.Sprintf("%s:%s", repo, version)
}
