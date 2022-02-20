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
	"fmt"
	"github.com/chaosblade-io/chaosblade-spec-go/spec"
	"github.com/chaosblade-io/chaosblade-spec-go/util"
	"os"
	"os/exec"
	"path"
	"time"

	containertype "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/gogo/protobuf/types"
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
	GetPidById(containerId string) (int32, error, int32)
	GetContainerById(containerId string) (ContainerInfo, error, int32)
	GetContainerByName(containerName string) (ContainerInfo, error, int32)
	RemoveContainer(containerId string, force bool) error
	CopyToContainer(containerId, srcFile, dstPath, extractDirName string, override bool) error

	ExecContainer(containerId, command string) (output string, err error)
	ExecuteAndRemove(config *containertype.Config, hostConfig *containertype.HostConfig,
		networkConfig *network.NetworkingConfig, containerName string, removed bool, timeout time.Duration,
		command string, containerInfo ContainerInfo) (containerId string, output string, err error, code int32)
}

func CopyToContainer(pid, srcFile, dstPath, extractDirName string, override bool) error {

	command := exec.Command(path.Join(util.GetProgramPath(), "bin", spec.NSExecBin),
		"-t", pid,
		"-p", "-m",
		"--",
		"/bin/sh", "-c",
		fmt.Sprintf("cat > %s", path.Join(dstPath, srcFile)))

	open, err := os.Open(srcFile)
	defer open.Close()
	if err != nil {
		return err
	}
	command.Stdin = open
	if err := command.Start(); err != nil {
		return err
	}
	if err := command.Wait(); err != nil {
		return err
	}
	return nil
}

func ExecContainer(pid int32, command string) (output string, err error) {

	cmd := exec.Command("/bin/sh", "-c",
		fmt.Sprintf("%s -t %d -p -m -- %s",
			path.Join(util.GetBinPath(), "nsexec"),
			pid,
			command))

	if combinedOutput, err := cmd.CombinedOutput(); err != nil {
		return "", err
	} else {
		return string(combinedOutput), nil
	}
}

//ContainerInfo for server
type ContainerInfo struct {
	ContainerId   string
	ContainerName string
	Labels        map[string]string
	Spec          *types.Any
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
