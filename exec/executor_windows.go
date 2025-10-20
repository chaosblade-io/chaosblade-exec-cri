//go:build windows

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
	"fmt"

	"github.com/chaosblade-io/chaosblade-spec-go/spec"

	"github.com/chaosblade-io/chaosblade-exec-cri/exec/container"
)

// DockerExpModelSpec type definition for Windows
type DockerExpModelSpec struct {
	ScopeName     string
	ExpModelSpecs map[string]spec.ExpModelCommandSpec
}

func (b *DockerExpModelSpec) ExpModels() map[string]spec.ExpModelCommandSpec {
	return b.ExpModelSpecs
}

func GetClientByRuntime(expModel *spec.ExpModel) (container.Container, error) {
	// Windows implementation - return error as container runtime support is limited on Windows
	return nil, fmt.Errorf("container runtime not fully supported on Windows platform")
}

func NewCriExpModelSpec() *DockerExpModelSpec {
	// Windows implementation - return empty model spec
	return &DockerExpModelSpec{
		ScopeName:     "cri",
		ExpModelSpecs: make(map[string]spec.ExpModelCommandSpec, 0),
	}
}

func NewDockerExpModelSpec() *DockerExpModelSpec {
	// Windows implementation - return empty model spec
	return &DockerExpModelSpec{
		ScopeName:     "docker",
		ExpModelSpecs: make(map[string]spec.ExpModelCommandSpec, 0),
	}
}

// Windows implementations of flag variables
var ContainerIdFlag = &spec.ExpFlag{
	Name:                  "container-id",
	Desc:                  "Container id, when used with container-name, container-id is preferred",
	NoArgs:                false,
	Required:              false,
	RequiredWhenDestroyed: false,
}

var ContainerNameFlag = &spec.ExpFlag{
	Name:                  "container-name",
	Desc:                  "Container name, when used with container-id, container-id is preferred",
	NoArgs:                false,
	Required:              false,
	RequiredWhenDestroyed: false,
}

var ContainerLabelSelectorFlag = &spec.ExpFlag{
	Name:                  "container-label-selector",
	Desc:                  "Container label selector, when used with container-id or container-name, container-id or container-name is preferred",
	NoArgs:                false,
	Required:              false,
	RequiredWhenDestroyed: false,
}

var EndpointFlag = &spec.ExpFlag{
	Name:     "cri-endpoint",
	Desc:     "Cri container socket endpoint",
	NoArgs:   false,
	Required: false,
}

var ContainerRuntime = &spec.ExpFlag{
	Name:     "container-runtime",
	Desc:     "container runtime, support cri and containerd, default value is docker",
	NoArgs:   false,
	Required: false,
}

var ContainerNamespace = &spec.ExpFlag{
	Name:     "container-namespace",
	Desc:     "container namespace, If container-runtime is containerd it will be used, default value is k8s.io",
	NoArgs:   false,
	Required: false,
}

var ChaosBladeReleaseFlag = &spec.ExpFlag{
	Name: "chaosblade-release",
	Desc: "The pull path of the chaosblade tar package, for example, --chaosblade-release /opt/chaosblade-0.4.0.tar.gz",
}

var ChaosBladeOverrideFlag = &spec.ExpFlag{
	Name:   "chaosblade-override",
	Desc:   "Override the exists chaosblade tool in the target container or not, default value is false",
	NoArgs: true,
}

var ImageRepoFlag = &spec.ExpFlag{
	Name:     "image-repo",
	Desc:     "Image repository of the chaosblade-tool",
	NoArgs:   false,
	Required: false,
}

var ImageVersionFlag = &spec.ExpFlag{
	Name:     "image-version",
	Desc:     "Image version of the chaosblade-tool",
	NoArgs:   false,
	Required: false,
}

func GetNSExecFlags() []spec.ExpFlagSpec {
	return []spec.ExpFlagSpec{
		ContainerIdFlag,
		ContainerNameFlag,
		EndpointFlag,
		ContainerRuntime,
		ContainerNamespace,
		ContainerLabelSelectorFlag,
	}
}

func GetAllDockerFlagNames() map[string]spec.Empty {
	flagNames := make(map[string]spec.Empty, 0)
	allFlags := []spec.ExpFlagSpec{
		ContainerIdFlag,
		ContainerNameFlag,
		EndpointFlag,
		ContainerRuntime,
		ContainerNamespace,
		ContainerLabelSelectorFlag,
		ImageRepoFlag,
		ImageVersionFlag,
		ChaosBladeReleaseFlag,
		ChaosBladeOverrideFlag,
	}
	for _, flag := range allFlags {
		flagNames[flag.FlagName()] = spec.Empty{}
	}
	return flagNames
}
