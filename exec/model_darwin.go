//go:build darwin

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
	"github.com/chaosblade-io/chaosblade-spec-go/spec"
)

func NewCriExpModelSpec() *DockerExpModelSpec {
	modelSpec := &DockerExpModelSpec{
		ScopeName:     "cri",
		ExpModelSpecs: make(map[string]spec.ExpModelCommandSpec, 0),
	}
	networkCommandModelSpec := newNetworkCommandModelSpecForDocker()
	execSidecarModelSpecs := []spec.ExpModelCommandSpec{
		networkCommandModelSpec,
	}

	javaExpModelSpecs := getJvmModels()
	execInContainerModelSpecs := []spec.ExpModelCommandSpec{
		newProcessCommandModelSpecForDocker(),
		newCpuCommandModelSpecForDocker(),
		newDiskFillCommandSpecForDocker(),
		newMemCommandModelSpecForDocker(),
		newFileCommandSpecForDocker(),
	}
	execInContainerModelSpecs = append(execInContainerModelSpecs, javaExpModelSpecs...)
	containerSelfModelSpec := NewContainerCommandSpec()

	spec.AddExecutorToModelSpec(NewNetWorkSidecarExecutor(), networkCommandModelSpec)
	spec.AddExecutorToModelSpec(NewRunCmdInContainerExecutorByCP(), execInContainerModelSpecs...)
	spec.AddFlagsToModelSpec(GetExecSidecarFlags, execSidecarModelSpecs...)
	spec.AddFlagsToModelSpec(GetContainerSelfFlags, containerSelfModelSpec)
	spec.AddFlagsToModelSpec(GetExecInContainerFlags, execInContainerModelSpecs...)

	expModelCommandSpecs := append(execSidecarModelSpecs, execInContainerModelSpecs...)
	expModelCommandSpecs = append(expModelCommandSpecs, containerSelfModelSpec)
	modelSpec.addExpModels(expModelCommandSpecs...)
	return modelSpec
}

func NewDockerExpModelSpec() *DockerExpModelSpec {
	modelSpec := &DockerExpModelSpec{
		ScopeName:     "docker",
		ExpModelSpecs: make(map[string]spec.ExpModelCommandSpec, 0),
	}
	networkCommandModelSpec := newNetworkCommandModelSpecForDocker()
	execSidecarModelSpecs := []spec.ExpModelCommandSpec{
		networkCommandModelSpec,
	}

	javaExpModelSpecs := getJvmModels()
	execInContainerModelSpecs := []spec.ExpModelCommandSpec{
		newProcessCommandModelSpecForDocker(),
		newCpuCommandModelSpecForDocker(),
		newDiskFillCommandSpecForDocker(),
		newMemCommandModelSpecForDocker(),
		newFileCommandSpecForDocker(),
	}
	execInContainerModelSpecs = append(execInContainerModelSpecs, javaExpModelSpecs...)
	containerSelfModelSpec := NewContainerCommandSpec()

	spec.AddExecutorToModelSpec(NewNetWorkSidecarExecutor(), networkCommandModelSpec)
	spec.AddExecutorToModelSpec(NewRunCmdInContainerExecutorByCP(), execInContainerModelSpecs...)
	spec.AddFlagsToModelSpec(GetExecSidecarFlags, execSidecarModelSpecs...)
	spec.AddFlagsToModelSpec(GetContainerSelfFlags, containerSelfModelSpec)
	spec.AddFlagsToModelSpec(GetExecInContainerFlags, execInContainerModelSpecs...)

	expModelCommandSpecs := append(execSidecarModelSpecs, execInContainerModelSpecs...)
	expModelCommandSpecs = append(expModelCommandSpecs, containerSelfModelSpec)
	modelSpec.addExpModels(expModelCommandSpecs...)
	return modelSpec
}
