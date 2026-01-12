//go:build linux || darwin

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
	"strings"

	"github.com/chaosblade-io/chaosblade-exec-os/exec/cpu"
	"github.com/chaosblade-io/chaosblade-exec-os/exec/disk"
	"github.com/chaosblade-io/chaosblade-exec-os/exec/file"
	"github.com/chaosblade-io/chaosblade-exec-os/exec/mem"
	"github.com/chaosblade-io/chaosblade-exec-os/exec/network"
	"github.com/chaosblade-io/chaosblade-exec-os/exec/process"
	"github.com/chaosblade-io/chaosblade-spec-go/spec"
)

type ResourceExpModelSpec interface {
	// Scope
	Scope() string
	// ExpModels returns the map of the experiment name and the model
	ExpModels() map[string]spec.ExpModelCommandSpec
	// GetExpActionModelSpec returns the action spec
	GetExpActionModelSpec(target, action string) spec.ExpActionCommandSpec
}

func newNetworkCommandModelSpecForDocker() spec.ExpModelCommandSpec {
	return network.NewNetworkCommandSpec()
}

func newFileCommandSpecForDocker() spec.ExpModelCommandSpec {
	return file.NewFileCommandSpec()
}

func newMemCommandModelSpecForDocker() spec.ExpModelCommandSpec {
	return mem.NewMemCommandModelSpec()
}

func newDiskFillCommandSpecForDocker() spec.ExpModelCommandSpec {
	return disk.NewDiskCommandSpec()
}

func newDiskCommandSpecForDocker() spec.ExpModelCommandSpec {
	return disk.NewDiskCommandSpec()
}

func newCpuCommandModelSpecForDocker() spec.ExpModelCommandSpec {
	return cpu.NewCpuCommandModelSpec()
}

func newProcessCommandModelSpecForDocker() spec.ExpModelCommandSpec {
	return process.NewProcessCommandModelSpec()
}

type DockerExpModelSpec struct {
	ScopeName     string
	ExpModelSpecs map[string]spec.ExpModelCommandSpec
}

func (b *DockerExpModelSpec) Scope() string {
	return b.ScopeName
}

func (b *DockerExpModelSpec) ExpModels() map[string]spec.ExpModelCommandSpec {
	return b.ExpModelSpecs
}

func (b *DockerExpModelSpec) GetExpActionModelSpec(target, actionName string) spec.ExpActionCommandSpec {
	commandSpec := b.ExpModelSpecs[target]
	if commandSpec == nil {
		return nil
	}
	actions := commandSpec.Actions()
	if actions == nil {
		return nil
	}
	for _, action := range actions {
		if action.Name() == actionName {
			return action
		}
		for _, alias := range action.Aliases() {
			if alias == actionName {
				return action
			}
		}
	}
	return nil
}

func (b *DockerExpModelSpec) addExpModels(expModel ...spec.ExpModelCommandSpec) {
	for _, model := range expModel {
		b.ExpModelSpecs[model.Name()] = model
	}
}

func addActionExamples(modelSpec *DockerExpModelSpec) {
	scopeName := modelSpec.ScopeName
	for _, expModelSpec := range modelSpec.ExpModels() {
		for _, action := range expModelSpec.Actions() {
			// For actions without specific examples, use default string replacement
			example := action.Example()
			if example != "" {
				example = strings.Replace(example,
					fmt.Sprintf("blade create %s %s", expModelSpec.Name(), action.Name()),
					fmt.Sprintf("blade create %s %s %s --container-id ee54f1e61c08 --container-runtime docker", scopeName, expModelSpec.Name(), action.Name()),
					-1,
				)
				example = strings.Replace(example,
					fmt.Sprintf("blade c %s %s", expModelSpec.Name(), action.Name()),
					fmt.Sprintf("blade c %s %s %s --container-id ee54f1e61c08 --container-runtime docker", scopeName, expModelSpec.Name(), action.Name()),
					-1,
				)
				example = strings.Replace(example,
					fmt.Sprintf("blade create docker %s %s", expModelSpec.Name(), action.Name()),
					fmt.Sprintf("blade create %s %s %s --container-id ee54f1e61c08 --container-runtime docker", scopeName, expModelSpec.Name(), action.Name()),
					-1,
				)
				action.SetExample(example)
			}
		}
	}
}

func GetAllExecutors() map[string]spec.Executor {
	executors := make(map[string]spec.Executor, 0)
	dockerModelSpecs := NewCriExpModelSpec()
	for _, expModel := range dockerModelSpecs.ExpModels() {
		executorMap := extractExecutorFromExpModel(expModel)
		for key, value := range executorMap {
			executors[key] = value
		}
	}
	return executors
}

func extractExecutorFromExpModel(expModel spec.ExpModelCommandSpec) map[string]spec.Executor {
	executors := make(map[string]spec.Executor)
	for _, actionModel := range expModel.Actions() {
		executors[GetExecutorKey(expModel.Name(), actionModel.Name())] = actionModel.Executor()
	}
	return executors
}

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

var EndpointFlag = &spec.ExpFlag{
	Name:     "cri-endpoint",
	Desc:     "Cri container socket endpoint",
	NoArgs:   false,
	Required: false,
}

var ChaosBladeReleaseFlag = &spec.ExpFlag{
	Name: "chaosblade-release",
	Desc: "The pull path of the chaosblade tar package, for example, --chaosblade-release /opt/chaosblade-v1.8.0-linux_amd64.tar.gz. Required on macOS/Darwin platform, optional on Linux platform (uses namespace execution). But if it's a Java scenario, you must add this parameter.",
}

var ChaosBladeOverrideFlag = &spec.ExpFlag{
	Name:   "chaosblade-override",
	Desc:   "Override the exists chaosblade tool in the target container or not, default value is false",
	NoArgs: true,
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

func GetContainerSelfFlags() []spec.ExpFlagSpec {
	return []spec.ExpFlagSpec{
		ContainerIdFlag,
		ContainerNameFlag,
		EndpointFlag,
		ContainerRuntime,
		ContainerNamespace,
	}
}

func GetExecSidecarFlags() []spec.ExpFlagSpec {
	return []spec.ExpFlagSpec{
		ContainerIdFlag,
		ContainerNameFlag,
		ImageRepoFlag,
		ImageVersionFlag,
		EndpointFlag,
		ContainerRuntime,
		ContainerNamespace,
	}
}

func GetExecInContainerFlags() []spec.ExpFlagSpec {
	return []spec.ExpFlagSpec{
		ContainerIdFlag,
		ContainerNameFlag,
		ImageRepoFlag,
		ImageVersionFlag,
		EndpointFlag,
		ChaosBladeReleaseFlag,
		ChaosBladeOverrideFlag,
		ContainerRuntime,
		ContainerNamespace,
	}
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

func getAllDockerFlags() []spec.ExpFlagSpec {
	allFlags := make([]spec.ExpFlagSpec, 0)
	allFlags = append(allFlags, GetContainerSelfFlags()...)
	allFlags = append(allFlags, GetExecSidecarFlags()...)
	allFlags = append(allFlags, GetExecInContainerFlags()...)

	set := make(map[spec.ExpFlagSpec]bool, 0)
	flags := make([]spec.ExpFlagSpec, 0)

	for i := range allFlags {
		if !set[allFlags[i]] {
			flags = append(flags, allFlags[i])
			set[allFlags[i]] = true
		}
	}

	return flags
}

func GetAllDockerFlagNames() map[string]spec.Empty {
	flagNames := make(map[string]spec.Empty, 0)
	for _, flag := range getAllDockerFlags() {
		flagNames[flag.FlagName()] = spec.Empty{}
	}
	return flagNames
}

func GetExecutorKey(target, action string) string {
	return fmt.Sprintf("%s-%s", target, action)
}
