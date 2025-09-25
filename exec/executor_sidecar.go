/*
 * Copyright 1999-2019 Alibaba Group Holding Ltd.
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

package exec

import (
	"context"
	"fmt"
	"time"

	"github.com/chaosblade-io/chaosblade-spec-go/log"
	"github.com/chaosblade-io/chaosblade-spec-go/spec"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"

	execContainer "github.com/chaosblade-io/chaosblade-exec-cri/exec/container"
)

type RunInSidecarContainerExecutor struct {
	BaseClientExecutor
	runConfigFunc func(container string) (container.HostConfig, network.NetworkingConfig)
	isResident    bool
}

func (*RunInSidecarContainerExecutor) Name() string {
	return "runAndExecSidecar"
}

func (r *RunInSidecarContainerExecutor) Exec(uid string, ctx context.Context, expModel *spec.ExpModel) *spec.Response {
	if err := r.SetClient(expModel); err != nil {
		log.Errorf(ctx, "%s", spec.ContainerExecFailed.Sprintf("GetClient", err))
		return spec.ResponseFailWithFlags(spec.ContainerExecFailed, "GetClient", err)
	}
	containerId := expModel.ActionFlags[ContainerIdFlag.Name]
	containerName := expModel.ActionFlags[ContainerNameFlag.Name]
	containerLabelSelector := parseContainerLabelSelector(expModel.ActionFlags[ContainerLabelSelectorFlag.Name])
	containerInfo, response := GetContainer(ctx, r.Client, uid, containerId, containerName, containerLabelSelector)
	if !response.Success {
		return response
	}
	hostConfig, networkingConfig := r.runConfigFunc(containerInfo.ContainerId)
	sidecarName := createSidecarContainerName(containerInfo.ContainerName, expModel.Target, expModel.ActionName)
	return r.startAndExecInContainer(uid, ctx, expModel, &hostConfig, &networkingConfig, sidecarName, containerInfo)
}

func NewNetWorkSidecarExecutor() *RunInSidecarContainerExecutor {
	runConfigFunc := func(containerId string) (container.HostConfig, network.NetworkingConfig) {
		hostConfig := container.HostConfig{
			NetworkMode: container.NetworkMode(fmt.Sprintf("container:%s", containerId)),
			CapAdd:      []string{"NET_ADMIN"},
		}
		networkConfig := network.NetworkingConfig{}
		return hostConfig, networkConfig
	}
	return &RunInSidecarContainerExecutor{
		// set the client when invoking
		runConfigFunc: runConfigFunc,
		isResident:    false,
		BaseClientExecutor: BaseClientExecutor{
			CommandFunc: CommonFunc,
		},
	}
}

func createSidecarContainerName(containerName, target, injectType string) string {
	return fmt.Sprintf("%s-%s-%s", containerName, target, injectType)
}

func (*RunInSidecarContainerExecutor) SetChannel(channel spec.Channel) {
}

func (r *RunInSidecarContainerExecutor) getContainerConfig(expModel *spec.ExpModel) *container.Config {
	return &container.Config{
		// detach
		AttachStdout: false,
		AttachStderr: false,
		Tty:          true,
		Cmd:          []string{"/bin/sh"},
		Image: execContainer.GetChaosBladeImageRef(expModel.ActionFlags[ImageRepoFlag.Name],
			expModel.ActionFlags[ImageVersionFlag.Name]),
		Labels: map[string]string{
			"chaosblade": "chaosblade-sidecar",
		},
	}
}

func (r *RunInSidecarContainerExecutor) startAndExecInContainer(uid string, ctx context.Context, expModel *spec.ExpModel,
	hostConfig *container.HostConfig, networkConfig *network.NetworkingConfig, containerName string, containerInfo execContainer.ContainerInfo,
) *spec.Response {
	config := r.getContainerConfig(expModel)
	var defaultResponse *spec.Response
	command := r.CommandFunc(uid, ctx, expModel)
	sidecarContainerId, output, err, code := r.Client.ExecuteAndRemove(ctx,
		config, hostConfig, networkConfig, containerName, true, time.Second, command, containerInfo)

	if err != nil {
		log.Errorf(ctx, "%s", err.Error())
		return spec.ResponseFail(code, err.Error(), nil)
	}
	returnedResponse := ConvertContainerOutputToResponse(output, err, defaultResponse)
	log.Infof(ctx, "sidecarContainerId for experiment %s is %s, output is %s, err is %v", uid, sidecarContainerId, output, err)
	return returnedResponse
}
