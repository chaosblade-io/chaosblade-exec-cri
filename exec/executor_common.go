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
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/chaosblade-io/chaosblade-exec-os/exec/model"
	"github.com/chaosblade-io/chaosblade-spec-go/spec"
	"github.com/chaosblade-io/chaosblade-spec-go/util"
	"github.com/containerd/cgroups"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
	"time"
)

// CommonExecutor is an executor implementation which used copy chaosblade tool to the target container and executed
type CommonExecutor struct {
	BaseClientExecutor
}

func NewCommonExecutor() *CommonExecutor {
	return &CommonExecutor{
		BaseClientExecutor{
			CommandFunc: CommonFunc,
		},
	}
}

func (r *CommonExecutor) Name() string {
	return "CommonExecutor"
}

func (r *CommonExecutor) Exec(uid string, ctx context.Context, expModel *spec.ExpModel) *spec.Response {
	if err := r.SetClient(expModel); err != nil {
		util.Errorf(uid, util.GetRunFuncName(), spec.ContainerExecFailed.Sprintf("GetClient", err))
		return spec.ResponseFailWithFlags(spec.ContainerExecFailed, "GetClient", err)
	}
	containerId := expModel.ActionFlags[ContainerIdFlag.Name]
	containerName := expModel.ActionFlags[ContainerNameFlag.Name]
	container, response := GetContainer(r.Client, uid, containerId, containerName)
	if !response.Success {
		return response
	}
	pid, err, code := r.Client.GetPidById(container.ContainerId)
	if err != nil {
		util.Errorf(uid, util.GetRunFuncName(), err.Error())
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
		if v == "" {
			continue
		}
		if m[k] != "" {
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
		fmt.Sprintf("--%s=%s", model.NsMntFlag.Name, spec.True),
	)

	if !isDestroy && expModel.ActionProcessHang {
		return execForHangAction(uid, ctx, expModel, pid, args)
	}

	chaosOsBin := path.Join(util.GetProgramPath(), spec.BinPath, spec.ChaosOsBin)
	argsArray := strings.Split(args, " ")

	command := exec.CommandContext(ctx, chaosOsBin, argsArray...)

	buf := new(bytes.Buffer)
	command.Stdout = buf
	command.Stderr = buf
	logrus.Debugf("run command, %s %s", chaosOsBin, args)
	if err := command.Start(); err != nil {
		sprintf := fmt.Sprintf("command start failed, %s", err.Error())
		return spec.ReturnFail(spec.OsCmdExecFailed, sprintf)
	}

	if err := command.Wait(); err != nil {
		sprintf := fmt.Sprintf("command wait failed, %s", err.Error())
		logrus.Debugf("command result: %s", buf.String())
		return spec.ReturnFail(spec.OsCmdExecFailed, sprintf)
	}
	logrus.Debugf("command result: %s", buf.String())
	return spec.Decode(buf.String(), nil)
}

func (r *CommonExecutor) SetChannel(channel spec.Channel) {
}

func (r *CommonExecutor) DeployChaosBlade(ctx context.Context, containerId string,
	srcFile, extractDirName string, override bool) error {
	return nil
}

func execForHangAction(uid string, ctx context.Context, expModel *spec.ExpModel, pid int32, args string) *spec.Response {

	chaosOsBin := path.Join(util.GetProgramPath(), spec.BinPath, spec.ChaosOsBin)

	args = fmt.Sprintf("-s %s %d %s -- %s %s", "-t", pid, "-p", chaosOsBin, args)

	argsArray := strings.Split(args, " ")

	bin := path.Join(util.GetProgramPath(), spec.BinPath, spec.NSExecBin)
	logrus.Debugf("run command, %s %s", bin, args)

	command := exec.CommandContext(ctx, bin, argsArray...)
	command.SysProcAttr = &syscall.SysProcAttr{}

	control, err := cgroups.Load(cgroups.V1, pidPath(int(pid)))
	if err != nil {
		sprintf := fmt.Sprintf("cgroups load failed, %s", err.Error())
		return spec.ReturnFail(spec.OsCmdExecFailed, sprintf)
	}

	if err := command.Start(); err != nil {
		sprintf := fmt.Sprintf("command start failed, %s", err.Error())
		return spec.ReturnFail(spec.OsCmdExecFailed, sprintf)
	}

	// add target cgroups
	if err = control.Add(cgroups.Process{Pid: command.Process.Pid}); err != nil {
		if err := command.Process.Kill(); err != nil {
			sprintf := fmt.Sprintf("create experiment failed, %v", err)
			return spec.ReturnFail(spec.OsCmdExecFailed, sprintf)
		}
	}

	signal := make(chan bool, 1)
	go func() {
		for {
			if comm, err := getProcessComm(command.Process.Pid); err != nil {
				logrus.Errorf("get process comm failed, %s", err.Error())
			} else {
				if cmdline, err := getProcessCmdline(command.Process.Pid); err != nil {
					logrus.Errorf("get process cmdline failed, %s", err.Error())
				} else {
					if cmdline == "" {
						logrus.Errorln("unknown err, process exit.")
						signal <- true
						break
					}
				}

				logrus.Infof("wait nasexec process pasue, current comm: %s, pid: %d", comm, command.Process.Pid)
				if comm == "pause\n" {
					signal <- true
					break
				}
			}
			time.Sleep(time.Millisecond)
		}
	}()

	if <-signal {
		for {
			if err := command.Process.Signal(syscall.SIGCONT); err != nil {
				sprintf := fmt.Sprintf("send signal failed, %s", err.Error())
				return spec.ReturnFail(spec.OsCmdExecFailed, sprintf)
			}
			time.Sleep(time.Millisecond)

			if comm, err := getProcessComm(command.Process.Pid); err != nil {
				logrus.Errorf("get process comm failed, %s", err.Error())
			} else {
				if cmdline, err := getProcessCmdline(command.Process.Pid); err != nil {
					logrus.Errorf("get process cmdline failed, %s", err.Error())
				} else {
					if cmdline == "" {
						logrus.Errorln("unknown err, process exit.")
						break
					}
				}

				logrus.Infof("wait nasexec process resume, current comm: %s, pid: %d", comm, command.Process.Pid)
				if comm == "nsexec\n" {
					break
				}
			}
		}
	}
	return spec.ReturnSuccess(uid)
}

func pidPath(pid int) cgroups.Path {
	p := fmt.Sprintf("/proc/%d/cgroup", pid)
	paths, err := cgroups.ParseCgroupFile(p)
	if err != nil {
		return func(_ cgroups.Name) (string, error) {
			return "", fmt.Errorf("failed to parse cgroup file %s: %s", p, err.Error())
		}
	}

	return func(name cgroups.Name) (string, error) {
		root, ok := paths[string(name)]
		if !ok {
			if root, ok = paths["name="+string(name)]; !ok {
				return "", errors.New("controller is not supported")
			}

		}
		return root, nil
	}
}

func getProcessComm(pid int) (string, error) {
	f, err := os.Open(fmt.Sprintf("%s/%d/comm", "/proc", pid))
	if err != nil {
		return "", err
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func getProcessCmdline(pid int) (string, error) {
	f, err := os.Open(fmt.Sprintf("%s/%d/cmdline", "/proc", pid))
	if err != nil {
		return "", err
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return "", err
	}

	return string(b), nil
}
