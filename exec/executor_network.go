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
	"github.com/chaosblade-io/chaosblade-exec-os/exec/model"
	"github.com/chaosblade-io/chaosblade-spec-go/log"
	"github.com/chaosblade-io/chaosblade-spec-go/spec"
	"github.com/chaosblade-io/chaosblade-spec-go/util"
	"os/exec"
	"path"
	"strings"
)

// NetworkExecutor is an executor implementation which used copy chaosblade tool to the target container and executed
type NetworkExecutor struct {
	BaseClientExecutor
}

func NewNetworkExecutor() *NetworkExecutor {
	return &NetworkExecutor{
		BaseClientExecutor{
			CommandFunc: CommonFunc,
		},
	}
}

func (r *NetworkExecutor) Name() string {
	return "networkExecutor"
}

func (r *NetworkExecutor) Exec(uid string, ctx context.Context, expModel *spec.ExpModel) *spec.Response {
	if err := r.SetClient(expModel); err != nil {
		log.Errorf(ctx, spec.ContainerExecFailed.Sprintf("GetClient", err))
		return spec.ResponseFailWithFlags(spec.ContainerExecFailed, "GetClient", err)
	}
	containerId := expModel.ActionFlags[ContainerIdFlag.Name]
	containerName := expModel.ActionFlags[ContainerNameFlag.Name]
	containerLabelSelector := parseContainerLabelSelector(expModel.ActionFlags[ContainerNameFlag.Name])
	container, response := GetContainer(ctx, r.Client, uid, containerId, containerName, containerLabelSelector)
	if !response.Success {
		return response
	}
	pid, err, code := r.Client.GetPidById(ctx, container.ContainerId)
	if err != nil {
		log.Errorf(ctx, err.Error())
		return spec.ResponseFail(code, err.Error(), nil)
	}

	var args string
	var flags string

	nsFlags := GetNSExecFlags()
	m := make(map[string]string, len(nsFlags))
	for _, f := range nsFlags {
		m[f.FlagName()] = f.FlagName()
	}

	for k, v := range expModel.ActionFlags {
		if v == "" || m[k] != "" || k == "timeout" {
			continue
		}
		flags = fmt.Sprintf("%s --%s=%s", flags, k, v)
	}
	_, isDestroy := spec.IsDestroy(ctx)

	if isDestroy {
		args = fmt.Sprintf("%s %s %s%s --uid=%s", spec.Destroy, expModel.Target, expModel.ActionName, flags, uid)
	} else {
		args = fmt.Sprintf("%s %s %s%s --uid=%s", spec.Create, expModel.Target, expModel.ActionName, flags, uid)
	}

	args = fmt.Sprintf("%s %s %s %s %s",
		args,
		fmt.Sprintf("--%s=%s", model.ChannelFlag.Name, spec.NSExecBin),
		fmt.Sprintf("--%s=%d", model.NsTargetFlag.Name, pid),
		fmt.Sprintf("--%s=%s", model.NsPidFlag.Name, spec.True),
		fmt.Sprintf("--%s=%s", model.NsNetFlag.Name, spec.True),
	)

	chaosOsBin := path.Join(util.GetProgramPath(), spec.BinPath, spec.ChaosOsBin)

	argsArray := strings.Split(args, " ")

	command := exec.CommandContext(ctx, chaosOsBin, argsArray...)
	output, err := command.CombinedOutput()
	outMsg := string(output)
	log.Debugf(ctx, "Command Result, output: %v, err: %v", outMsg, err)
	if err != nil {
		return spec.ReturnFail(spec.OsCmdExecFailed, fmt.Sprintf("command exec failed, %s", err.Error()))
	}
	return spec.Decode(outMsg, nil)
}

func (r *NetworkExecutor) SetChannel(channel spec.Channel) {
}

func (r *NetworkExecutor) DeployChaosBlade(ctx context.Context, containerId string,
	srcFile, extractDirName string, override bool) error {
	return nil
}
