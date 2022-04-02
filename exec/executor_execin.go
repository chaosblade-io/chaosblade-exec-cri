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
	"github.com/chaosblade-io/chaosblade-spec-go/log"
	"path"
	"strconv"
	"strings"

	"github.com/chaosblade-io/chaosblade-exec-cri/version"
	"github.com/chaosblade-io/chaosblade-spec-go/channel"
	"github.com/chaosblade-io/chaosblade-spec-go/spec"
)

var defaultBladeTarFilePath = fmt.Sprintf("/opt/chaosblade-%s.tar.gz", version.BladeVersion)

// RunCmdInContainerExecutor is an executor interface which executes command in the target container directly
type RunCmdInContainerExecutor interface {
	spec.Executor
	DeployChaosBlade(ctx context.Context, containerId string, srcFile, extractDirName string, override bool) error
}

// RunCmdInContainerExecutorByCP is an executor implementation which used copy chaosblade tool to the target container and executed
type RunCmdInContainerExecutorByCP struct {
	BaseClientExecutor
}

func NewRunCmdInContainerExecutorByCP() RunCmdInContainerExecutor {
	return &RunCmdInContainerExecutorByCP{
		BaseClientExecutor{
			CommandFunc: CommonFunc,
		},
	}
}

func (r *RunCmdInContainerExecutorByCP) Name() string {
	return "runCmdInContainerExecutorByCP"
}

func (r *RunCmdInContainerExecutorByCP) Exec(uid string, ctx context.Context, expModel *spec.ExpModel) *spec.Response {
	if err := r.SetClient(expModel); err != nil {
		log.Errorf(ctx, spec.ContainerExecFailed.Sprintf("GetClient", err))
		return spec.ResponseFailWithFlags(spec.ContainerExecFailed, "GetClient", err)
	}
	containerId := expModel.ActionFlags[ContainerIdFlag.Name]
	containerName := expModel.ActionFlags[ContainerNameFlag.Name]
	container, response := GetContainer(ctx, r.Client, uid, containerId, containerName)
	if !response.Success {
		return response
	}
	command := r.CommandFunc(uid, ctx, expModel)
	if _, ok := spec.IsDestroy(ctx); !ok {
		// Create
		chaosbladeReleaseFile := expModel.ActionFlags[ChaosBladeReleaseFlag.Name]
		if chaosbladeReleaseFile == "" {
			chaosbladeReleaseFile = defaultBladeTarFilePath
		}
		overrideValue := expModel.ActionFlags[ChaosBladeOverrideFlag.Name]
		override, err := strconv.ParseBool(overrideValue)
		if err != nil {
			override = false
		}
		if resp, ok := channel.NewLocalChannel().IsAllCommandsAvailable(ctx, []string{"tar"}); !ok {
			log.Errorf(ctx, resp.Err)
			return resp
		}

		response := channel.NewLocalChannel().Run(context.Background(), "tar",
			fmt.Sprintf("tf %s| head -1 | cut -f1 -d/", chaosbladeReleaseFile))
		if !response.Success {
			log.Errorf(ctx, "`%s`: chaosblade-release parameter is invalid, err: %s", chaosbladeReleaseFile, response.Err)
			return spec.ResponseFailWithFlags(spec.ParameterInvalid, ChaosBladeReleaseFlag.Name, chaosbladeReleaseFile, response.Err)
		}
		if response.Result == nil {
			log.Errorf(ctx, "`%s`: chaosblade-release parameter is invalid, extract directory failed", chaosbladeReleaseFile)
			return spec.ResponseFailWithFlags(spec.ParameterInvalid, ChaosBladeReleaseFlag.Name, chaosbladeReleaseFile, "the obtained directory name is nil")
		}
		extractedDirName := strings.TrimSpace(response.Result.(string))
		if extractedDirName == "" {
			log.Errorf(ctx, "`%s`: chaosblade-release parameter is invalid, extract empty directory failed", chaosbladeReleaseFile)
			return spec.ResponseFailWithFlags(spec.ParameterInvalid, ChaosBladeReleaseFlag.Name, chaosbladeReleaseFile, "the obtained directory name is empty")

		}
		err = r.DeployChaosBlade(ctx, container.ContainerId, chaosbladeReleaseFile, extractedDirName, override)
		if err != nil {
			log.Errorf(ctx,"DeployChaosBlade err: %v", err)
			return spec.ResponseFailWithFlags(spec.ContainerExecFailed, "DeployChaosBlade", err)
		}
	}
	output, err := r.Client.ExecContainer(ctx, container.ContainerId, command)
	var defaultResponse *spec.Response
	if err != nil {
		log.Errorf(ctx,"execContainer err: %v", err)
		return spec.ResponseFailWithFlags(spec.ContainerExecFailed, "execContainer", err)
	}
	return ConvertContainerOutputToResponse(output, err, defaultResponse)
	//return spec.Success()
}

func (r *RunCmdInContainerExecutorByCP) SetChannel(channel spec.Channel) {
}

func (r *RunCmdInContainerExecutorByCP) DeployChaosBlade(ctx context.Context, containerId string,
	srcFile, extractDirName string, override bool) error {
	// check if the blade tool exists
	// todo for test
	output, err := r.Client.ExecContainer(ctx, containerId, fmt.Sprintf("[ -e %s ] && echo True || echo False", BladeBin))
	if err == nil && strings.Contains(output, "True") && !override {
		return nil
	}

	err = r.Client.CopyToContainer(ctx, containerId, srcFile, DstChaosBladeDir, extractDirName, override)
	if err != nil {
		return err
	}

	dstBladeDir := path.Join(DstChaosBladeDir, extractDirName)
	expectBladeDir := path.Join(DstChaosBladeDir, "chaosblade")
	rmCmd := fmt.Sprintf("rm -rf %s", expectBladeDir)
	_, err = r.Client.ExecContainer(ctx, containerId, rmCmd)
	if err != nil {
		return err
	}

	renameCmd := fmt.Sprintf("mv %s %s", dstBladeDir, expectBladeDir)
	_, err = r.Client.ExecContainer(ctx, containerId, renameCmd)
	return err
}
