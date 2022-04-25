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
	"strings"

	"github.com/chaosblade-io/chaosblade-spec-go/spec"
	"github.com/chaosblade-io/chaosblade-spec-go/util"

	"github.com/chaosblade-io/chaosblade-exec-cri/exec/container"
	"github.com/chaosblade-io/chaosblade-exec-cri/exec/container/containerd"
	"github.com/chaosblade-io/chaosblade-exec-cri/exec/container/docker"
)

const DstChaosBladeDir = "/opt"

// BladeBin is the blade path in the chaosblade-tool image
const BladeBin = "/opt/chaosblade/blade"

// BaseDockerClientExecutor
type BaseDockerClientExecutor struct {
	Client      container.Container
	CommandFunc func(uid string, ctx context.Context, model *spec.ExpModel) string
}

// SetClient to the executor
func (b *BaseDockerClientExecutor) SetClient(expModel *spec.ExpModel) error {
	cli, err := GetClientByRuntime(expModel)
	if err != nil {
		return err
	}
	b.Client = cli
	return nil
}

// commonFunc is the command created function
var commonFunc = func(uid string, ctx context.Context, model *spec.ExpModel) string {
	matchers := spec.ConvertExpMatchersToString(model, func() map[string]spec.Empty {
		return GetAllDockerFlagNames()
	})
	if _, ok := spec.IsDestroy(ctx); ok {
		// UPDATE: https://github.com/chaosblade-io/chaosblade/issues/334
		return fmt.Sprintf("%s destroy %s %s %s", BladeBin, model.Target, model.ActionName, matchers)
	}
	return fmt.Sprintf("%s create %s %s %s --uid %s", BladeBin, model.Target, model.ActionName, matchers, uid)
}

func ConvertContainerOutputToResponse(output string, err error, defaultResponse *spec.Response) *spec.Response {
	if err != nil {
		response := spec.Decode(err.Error(), defaultResponse)
		if response.Success {
			return response
		}
		return spec.ResponseFailWithFlags(spec.ContainerExecFailed, "execContainer", err)
	}
	output = strings.TrimSpace(output)
	if output == "" {
		return spec.ResponseFailWithFlags(spec.ContainerExecFailed, "execContainer",
			"cannot get result message from container, please execute recovery and try again")
	}
	return spec.Decode(output, defaultResponse)
}

func GetClientByRuntime(expModel *spec.ExpModel) (container.Container, error) {
	switch expModel.ActionFlags[ContainerRuntime.Name] {
	case container.ContainerdRuntime:
		return containerd.NewClient(expModel.ActionFlags[EndpointFlag.Name], expModel.ActionFlags[ContainerNamespace.Name])
	default:
		return docker.NewClient(expModel.ActionFlags[EndpointFlag.Name])
		//default:
		//	return nil,errors.New(fmt.Sprintf("`%s`, the container runtime not support", expModel.ActionFlags[ContainerRuntime.Name]))
	}
}

// GetContainer return container by container flag, such as container id or container name.
func GetContainer(client container.Container, uid string, containerId, containerName string, containerLabelSelector map[string]string) (container.ContainerInfo, *spec.Response) {
	if containerId == "" && containerName == "" && containerLabelSelector == nil {
		tips := fmt.Sprintf("%s or %s or %s", ContainerIdFlag.Name, ContainerNameFlag.Name, ContainerLabelSelectorFlag.Name)
		util.Errorf(uid, util.GetRunFuncName(), spec.ParameterLess.Sprintf(tips))
		return container.ContainerInfo{}, spec.ResponseFailWithFlags(spec.ParameterLess, tips)
	}
	var container container.ContainerInfo
	var code int32
	var err error
	if containerId != "" {
		container, err, code = client.GetContainerById(containerId)
	} else if containerName != "" {
		container, err, code = client.GetContainerByName(containerName)
	} else {
		container, err, code = client.GetContainerByLabelSelector(containerLabelSelector)
	}
	if err != nil {
		util.Errorf(uid, util.GetRunFuncName(), err.Error())
		return container, spec.ResponseFail(code, err.Error(), nil)
	}
	return container, spec.ReturnSuccess(container)
}

func parseContainerLabelSelector(raw string) map[string]string {
	labels := make(map[string]string, 0)

	if raw != "" {
		for _, label := range strings.Split(raw, ",") {
			keyAndValue := strings.Split(label, "=")
			if len(keyAndValue) == 2 {
				labels[keyAndValue[0]] = keyAndValue[1]
			}
		}
	}

	return labels
}
