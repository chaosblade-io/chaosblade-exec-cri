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

	"github.com/chaosblade-io/chaosblade-exec-os/exec"
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

func NewCriExpModelSpec() *dockerExpModelSpec {
	modelSpec := &dockerExpModelSpec{
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
		newDiskCommandSpecForDocker(),
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

func newNetworkCommandModelSpecForDocker() spec.ExpModelCommandSpec {
	networkCommandModelSpec := exec.NewNetworkCommandSpec()
	for _, action := range networkCommandModelSpec.Actions() {
		v := interface{}(action)
		switch v.(type) {
		case *exec.DelayActionSpec:
			action.SetExample(
				`# Access to native 8080 and 8081 ports is delayed by 3 seconds, and the delay time fluctuates by 1 second
blade create cri network delay --time 3000 --offset 1000 --interface eth0 --local-port 8080,8081 --container-id ee54f1e61c08

# Local access to external 14.215.177.39 machine (ping www.baidu.com obtained IP) port 80 delay of 3 seconds
blade create cri network delay --time 3000 --interface eth0 --remote-port 80 --destination-ip 14.215.177.39 --container-id ee54f1e61c08

# Do a 5 second delay for the entire network card eth0, excluding ports 22 and 8000 to 8080
blade create cri network delay --time 5000 --interface eth0 --exclude-port 22,8000-8080 --container-id ee54f1e61c08`)
		case *exec.DropActionSpec:
			action.SetExample(
				`# Block incoming connection from the port 80
blade create cri network drop --source-port 80 --network-traffic in --container-id ee54f1e61c08`)
		case *exec.DnsActionSpec:
			action.SetExample(
				`# The domain name www.baidu.com is not accessible
blade create cri network dns --domain www.baidu.com --ip 10.0.0.0 --container-id ee54f1e61c08`)
		case *exec.LossActionSpec:
			action.SetExample(`# Access to native 8080 and 8081 ports lost 70% of packets
blade create cri network loss --percent 70 --interface eth0 --local-port 8080,8081 --container-id ee54f1e61c08

# The machine accesses external 14.215.177.39 machine (ping www.baidu.com) 80 port packet loss rate 100%
blade create cri network loss --percent 100 --interface eth0 --remote-port 80 --destination-ip 14.215.177.39 --container-id ee54f1e61c08

# Do 60% packet loss for the entire network card Eth0, excluding ports 22 and 8000 to 8080
blade create cri network loss --percent 60 --interface eth0 --exclude-port 22,8000-8080 --container-id ee54f1e61c08

# Realize the whole network card is not accessible, not accessible time 20 seconds. After executing the following command, the current network is disconnected and restored in 20 seconds. Remember!! Don't forget -timeout parameter
blade create cri network loss --percent 100 --interface eth0 --timeout 20 --container-id ee54f1e61c08`)
		case *exec.DuplicateActionSpec:
			action.SetExample(`# Specify the network card eth0 and repeat the packet by 10%
blade create cri network duplicate --percent=10 --interface=eth0 --container-id ee54f1e61c08`)
		case *exec.CorruptActionSpec:
			action.SetExample(`# Access to the specified IP request packet is corrupted, 80% of the time
blade create cri network corrupt --percent 80 --destination-ip 180.101.49.12 --interface eth0 --container-id ee54f1e61c08`)
		case *exec.ReorderActionSpec:
			action.SetExample(`# Access the specified IP request packet disorder
blade create cri network reorder --correlation 80 --percent 50 --gap 2 --time 500 --interface eth0 --destination-ip 180.101.49.12 --container-id ee54f1e61c08`)
		case *exec.OccupyActionSpec:
			action.SetExample(`#Specify port 8080 occupancy
blade create cri network occupy --port 8080 --force --container-id ee54f1e61c08

# The machine accesses external 14.215.177.39 machine (ping www.baidu.com) 80 port packet loss rate 100%
blade create cri network loss --percent 100 --interface eth0 --remote-port 80 --destination-ip 14.215.177.39 --container-id ee54f1e61c08`)
		}
	}
	return networkCommandModelSpec
}

func newFileCommandSpecForDocker() spec.ExpModelCommandSpec {
	fileCommandSpec := exec.NewFileCommandSpec()
	for _, action := range fileCommandSpec.Actions() {
		v := interface{}(action)
		switch v.(type) {
		case *exec.FileAppendActionSpec:
			action.SetLongDesc("The file append experiment scenario in container")
			action.SetExample(
				`# Appends the content "HELLO WORLD" to the /home/logs/nginx.log file
blade create cri file append --filepath=/home/logs/nginx.log --content="HELL WORLD" --chaosblade-release /root/chaosblade-0.6.0.tar.gz --container-id ee54f1e61c08

# Appends the content "HELLO WORLD" to the /home/logs/nginx.log file, interval 10 seconds
blade create cri file append --filepath=/home/logs/nginx.log --content="HELL WORLD" --interval 10 --chaosblade-release /root/chaosblade-0.6.0.tar.gz --container-id ee54f1e61c08

# Appends the content "HELLO WORLD" to the /home/logs/nginx.log file, enable base64 encoding
blade create cri file append --filepath=/home/logs/nginx.log --content=SEVMTE8gV09STEQ= --chaosblade-release /root/chaosblade-0.6.0.tar.gz --container-id ee54f1e61c08

# mock interface timeout exception
blade create cri file append --filepath=/home/logs/nginx.log --content="@{DATE:+%Y-%m-%d %H:%M:%S} ERROR invoke getUser timeout [@{RANDOM:100-200}]ms abc  mock exception" --chaosblade-release /root/chaosblade-0.6.0.tar.gz --container-id ee54f1e61c08
`)
		case *exec.FileAddActionSpec:
			action.SetLongDesc("The file add experiment scenario in container")
			action.SetExample(
				`# Create a file named nginx.log in the /home directory
blade create cri file add --filepath /home/nginx.log --chaosblade-release /root/chaosblade-0.6.0.tar.gz --container-id ee54f1e61c08

# Create a file named nginx.log in the /home directory with the contents of HELLO WORLD
blade create cri file add --filepath /home/nginx.log --content "HELLO WORLD" --chaosblade-release /root/chaosblade-0.6.0.tar.gz --container-id ee54f1e61c08

# Create a file named nginx.log in the /temp directory and automatically create directories that don't exist
blade create cri file add --filepath /temp/nginx.log --auto-create-dir --chaosblade-release /root/chaosblade-0.6.0.tar.gz --container-id ee54f1e61c08

# Create a directory named /nginx in the /temp directory and automatically create directories that don't exist
blade create cri file add --directory --filepath /temp/nginx --auto-create-dir --chaosblade-release /root/chaosblade-0.6.0.tar.gz --container-id ee54f1e61c08
`)

		case *exec.FileChmodActionSpec:
			action.SetLongDesc("The file permission modification scenario in container")
			action.SetExample(`# Modify /home/logs/nginx.log file permissions to 777
blade create cri file chmod --filepath /home/logs/nginx.log --mark=777 --chaosblade-release /root/chaosblade-0.6.0.tar.gz --container-id ee54f1e61c08
`)
		case *exec.FileDeleteActionSpec:
			action.SetLongDesc("The file delete scenario in container")
			action.SetExample(
				`# Delete the file /home/logs/nginx.log
blade create cri file delete --filepath /home/logs/nginx.log --chaosblade-release /root/chaosblade-0.6.0.tar.gz --container-id ee54f1e61c08

# Force delete the file /home/logs/nginx.log unrecoverable
blade create cri file delete --filepath /home/logs/nginx.log --force --chaosblade-release /root/chaosblade-0.6.0.tar.gz --container-id ee54f1e61c08
`)
		case *exec.FileMoveActionSpec:
			action.SetExample("The file move scenario in container")
			action.SetExample(`# Move the file /home/logs/nginx.log to /tmp
blade create cri file delete --filepath /home/logs/nginx.log --target /tmp --chaosblade-release /root/chaosblade-0.6.0.tar.gz --container-id ee54f1e61c08

# Force Move the file /home/logs/nginx.log to /temp
blade create cri file delete --filepath /home/logs/nginx.log --target /tmp --force --chaosblade-release /root/chaosblade-0.6.0.tar.gz --container-id ee54f1e61c08

# Move the file /home/logs/nginx.log to /temp/ and automatically create directories that don't exist
blade create cri file delete --filepath /home/logs/nginx.log --target /temp --auto-create-dir --chaosblade-release /root/chaosblade-0.6.0.tar.gz --container-id ee54f1e61c08
`)
		}
	}
	return fileCommandSpec
}

func newMemCommandModelSpecForDocker() spec.ExpModelCommandSpec {
	memCommandModelSpec := exec.NewMemCommandModelSpec()
	for _, action := range memCommandModelSpec.Actions() {
		v := interface{}(action)
		switch v.(type) {
		case *exec.MemLoadActionCommand:
			action.SetLongDesc("The memory fill experiment scenario in container")
			action.SetExample(
				`# The execution memory footprint is 50%
blade create cri mem load --mode ram --mem-percent 50 --chaosblade-release /root/chaosblade-0.6.0.tar.gz --container-id ee54f1e61c08

# The execution memory footprint is 50%, cache model
blade create cri mem load --mode cache --mem-percent 50 --chaosblade-release /root/chaosblade-0.6.0.tar.gz --container-id ee54f1e61c08

# The execution memory footprint is 50%, usage contains buffer/cache
blade create cri mem load --mode ram --mem-percent 50 --include-buffer-cache --chaosblade-release /root/chaosblade-0.6.0.tar.gz --container-id ee54f1e61c08

# The execution memory footprint is 50% for 200 seconds
blade create cri mem load --mode ram --mem-percent 50 --timeout 200 --chaosblade-release /root/chaosblade-0.6.0.tar.gz --container-id ee54f1e61c08

# 200M memory is reserved
blade create cri mem load --mode ram --reserve 200 --rate 100 --chaosblade-release /root/chaosblade-0.6.0.tar.gz --container-id ee54f1e61c08`)
		}
	}
	return memCommandModelSpec
}

func newDiskCommandSpecForDocker() spec.ExpModelCommandSpec {
	commandSpec := exec.NewDiskCommandSpec()
	for _, action := range commandSpec.Actions() {
		v := interface{}(action)
		switch v.(type) {
		case *exec.FillActionSpec:
			action.SetLongDesc("The disk fill scenario experiment in the container")
			action.SetExample(
				`
# Fill the /home directory with 40G of disk space in the container
blade create cri disk fill --path /home --size 40000 --chaosblade-release /root/chaosblade-0.6.0.tar.gz --container-id ee54f1e61c08

# Fill the /home directory with 80% of the disk space in the container and retains the file handle that populates the disk
blade create cri disk fill --path /home --percent 80 --retain-handle --chaosblade-release /root/chaosblade-0.6.0.tar.gz --container-id ee54f1e61c08

# Perform a fixed-size experimental scenario in the container
blade c cri disk fill --path /home --reserve 1024 --chaosblade-release /root/chaosblade-0.6.0.tar.gz --container-id ee54f1e61c08
`)
		case *exec.BurnActionSpec:
			action.SetLongDesc("Disk read and write IO load experiment in the container")
			action.SetExample(
				`# The data of rkB/s, wkB/s and % Util were mainly observed. Perform disk read IO high-load scenarios
blade create cri disk burn --read --path /home --chaosblade-release /root/chaosblade-0.6.0.tar.gz --container-id ee54f1e61c08

# Perform disk write IO high-load scenarios
blade create cri disk burn --write --path /home --chaosblade-release /root/chaosblade-0.6.0.tar.gz --container-id ee54f1e61c08

# Read and write IO load scenarios are performed at the same time. Path is not specified. The default is /
blade create cri disk burn --read --write --chaosblade-release /root/chaosblade-0.6.0.tar.gz --container-id ee54f1e61c08`)
		}
	}
	return commandSpec
}

func newCpuCommandModelSpecForDocker() spec.ExpModelCommandSpec {
	cpuCommandModelSpec := exec.NewCpuCommandModelSpec()
	for _, action := range cpuCommandModelSpec.Actions() {
		v := interface{}(action)
		switch v.(type) {
		case *exec.FullLoadActionCommand:
			action.SetLongDesc("The CPU load experiment scenario in container is the same as the CPU scenario of basic resources")
			action.SetExample(
				`# Create a CPU full load experiment in the container
blade create cri cpu load --chaosblade-release /root/chaosblade-0.6.0.tar.gz --container-id ee54f1e61c08

#Specifies two random kernel's full load in the container
blade create cri cpu load --cpu-percent 60 --cpu-count 2 --chaosblade-release /root/chaosblade-0.6.0.tar.gz --container-id ee54f1e61c08

# Specifies that the kernel is full load with index 0, 3, and that the kernel's index starts at 0
blade create cri cpu load --cpu-list 0,3 --chaosblade-release /root/chaosblade-0.6.0.tar.gz --container-id ee54f1e61c08

# Specify the kernel full load of indexes 1-3
blade create cri cpu load --cpu-list 1-3 --chaosblade-release /root/chaosblade-0.6.0.tar.gz --container-id ee54f1e61c08

# Specified percentage load in the container
blade create cri cpu load --cpu-percent 60 --chaosblade-release /root/chaosblade-0.6.0.tar.gz --container-id ee54f1e61c08`)
		}
	}
	return cpuCommandModelSpec
}

func newProcessCommandModelSpecForDocker() spec.ExpModelCommandSpec {
	commandModelSpec := exec.NewProcessCommandModelSpec()
	for _, action := range commandModelSpec.Actions() {
		v := interface{}(action)
		switch v.(type) {
		case *exec.KillProcessActionCommandSpec:
			action.SetLongDesc("The process scenario in container is the same as the basic resource process scenario")
			action.SetExample(
				`# Kill the nginx process in the container
blade create cri process kill --process nginx --chaosblade-release /root/chaosblade-0.6.0.tar.gz --container-id ee54f1e61c08

# Specifies the signal and local port to kill the process in the container
blade create cri process kill --local-port 8080 --signal 15 --chaosblade-release /root/chaosblade-0.6.0.tar.gz --container-id ee54f1e61c08`)

		case *exec.StopProcessActionCommandSpec:
			action.SetLongDesc("The process scenario in container is the same as the basic resource process scenario")
			action.SetExample(
				`# Pause the process that contains the "nginx" keyword in the container
blade create cri process stop --process nginx --chaosblade-release /root/chaosblade-0.6.0.tar.gz --container-id ee54f1e61c08

# Pause the Java process in the container
blade create cri process stop --process-cmd java --chaosblade-release /root/chaosblade-0.6.0.tar.gz --container-id ee54f1e61c08`)

		}
	}
	return commandModelSpec
}

type dockerExpModelSpec struct {
	ScopeName     string
	ExpModelSpecs map[string]spec.ExpModelCommandSpec
}

func (b *dockerExpModelSpec) Scope() string {
	return b.ScopeName
}

func (b *dockerExpModelSpec) ExpModels() map[string]spec.ExpModelCommandSpec {
	return b.ExpModelSpecs
}

func (b *dockerExpModelSpec) GetExpActionModelSpec(target, actionName string) spec.ExpActionCommandSpec {
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

func (b *dockerExpModelSpec) addExpModels(expModel ...spec.ExpModelCommandSpec) {
	for _, model := range expModel {
		b.ExpModelSpecs[model.Name()] = model
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
	Desc: "The pull path of the chaosblade tar package, for example, --chaosblade-release /opt/chaosblade-0.4.0.tar.gz",
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
		ContainerLabelSelectorFlag,
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
