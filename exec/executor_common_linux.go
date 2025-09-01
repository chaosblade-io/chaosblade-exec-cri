//go:build linux

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
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
	"time"

	osexec "github.com/chaosblade-io/chaosblade-exec-os/exec"
	"github.com/chaosblade-io/chaosblade-exec-os/exec/model"
	"github.com/chaosblade-io/chaosblade-spec-go/log"
	"github.com/chaosblade-io/chaosblade-spec-go/spec"
	"github.com/chaosblade-io/chaosblade-spec-go/util"
	"github.com/containerd/cgroups"
	cgroupsv2 "github.com/containerd/cgroups/v2"
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
		log.Errorf(ctx, spec.ContainerExecFailed.Sprintf("GetClient", err))
		return spec.ResponseFailWithFlags(spec.ContainerExecFailed, "GetClient", err)
	}
	containerId := expModel.ActionFlags[ContainerIdFlag.Name]
	containerName := expModel.ActionFlags[ContainerNameFlag.Name]
	containerLabelSelector := parseContainerLabelSelector(expModel.ActionFlags[ContainerLabelSelectorFlag.Name])
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

	cgroupRoot := os.Getenv("CGROUP_ROOT")
	if cgroupRoot != "" && expModel.ActionProcessHang {
		expModel.ActionFlags["cgroup-root"] = cgroupRoot
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
		fmt.Sprintf("--%s=%s", model.NsMntFlag.Name, spec.True),
	)

	if !isDestroy && expModel.ActionProcessHang {
		return execForHangAction(uid, ctx, expModel, pid, args)
	}

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

func (r *CommonExecutor) SetChannel(channel spec.Channel) {
}

func (r *CommonExecutor) DeployChaosBlade(ctx context.Context, containerId string,
	srcFile, extractDirName string, override bool) error {
	return nil
}

func execForHangAction(uid string, ctx context.Context, expModel *spec.ExpModel, pid int32, args string) *spec.Response {

	chaosOsBin := path.Join(util.GetProgramPath(), spec.BinPath, spec.ChaosOsBin)

	args = fmt.Sprintf("-s -t %d -p -n -- %s %s", pid, chaosOsBin, args)

	argsArray := strings.Split(args, " ")

	bin := path.Join(util.GetProgramPath(), spec.BinPath, spec.NSExecBin)
	log.Debugf(ctx, "run command, %s %s", bin, args)

	command := exec.CommandContext(ctx, bin, argsArray...)
	command.SysProcAttr = &syscall.SysProcAttr{}

	cgroupRoot := os.Getenv("CGROUP_ROOT")
	if cgroupRoot == "" {
		cgroupRoot = expModel.ActionFlags["cgroup-root"]
		if cgroupRoot == "" {
			cgroupRoot = "/sys/fs/cgroup/"
		}
	}

	log.Debugf(ctx, "cgroup root path %s", cgroupRoot)

	isCgroupV2 := false
	if _, err := os.Stat(fmt.Sprintf("%s/cgroup.controllers", cgroupRoot)); err == nil {
		isCgroupV2 = true
	}
	if isCgroupV2 {
		g, err := cgroupsv2.PidGroupPath(int(pid))
		if err != nil {
			sprintf := fmt.Sprintf("loading cgroup2 for %d, err ", pid, err.Error())
			return spec.ReturnFail(spec.OsCmdExecFailed, sprintf)
		}
		cg, err := cgroupsv2.LoadManager("/sys/fs/cgroup/", g)
		if err != nil {
			if err != cgroupsv2.ErrCgroupDeleted {
				sprintf := fmt.Sprintf("cgroups V2 load failed, %s", err.Error())
				return spec.ReturnFail(spec.OsCmdExecFailed, sprintf)
			}
			if cg, err = cgroupsv2.NewManager("/sys/fs/cgroup", cgroupRoot, nil); err != nil {
				sprintf := fmt.Sprintf("cgroups V2 new manager failed, %s", err.Error())
				return spec.ReturnFail(spec.OsCmdExecFailed, sprintf)
			}
		}
		if err := command.Start(); err != nil {
			sprintf := fmt.Sprintf("command start failed, %s", err.Error())
			return spec.ReturnFail(spec.OsCmdExecFailed, sprintf)
		}
		if err := cg.AddProc(uint64(command.Process.Pid)); err != nil {
			if err := command.Process.Kill(); err != nil {
				sprintf := fmt.Sprintf("add process to cgroups V2 failed, %s", err.Error())
				return spec.ReturnFail(spec.OsCmdExecFailed, sprintf)
			}
		}
	} else {
		control, err := cgroups.Load(osexec.Hierarchy(cgroupRoot), osexec.PidPath(int(pid)))
		if err != nil {
			sprintf := fmt.Sprintf("cgroups V1 load failed, %s", err.Error())
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
	}

	signal := make(chan bool, 1)
	go func() {
		for {
			if comm, err := getProcessComm(command.Process.Pid); err != nil {
				log.Errorf(ctx, "get process comm failed, %s", err.Error())
			} else {
				if cmdline, err := getProcessCmdline(command.Process.Pid); err != nil {
					log.Errorf(ctx, "get process cmdline failed, %s", err.Error())
				} else {
					if cmdline == "" {
						log.Errorf(ctx, "unknown err, process exit.")
						signal <- true
						break
					}
				}

				log.Infof(ctx, "wait nsexec process pasue, current comm: %s, pid: %d", comm, command.Process.Pid)
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
				log.Errorf(ctx, "get process comm failed, %s", err.Error())
			} else {
				if cmdline, err := getProcessCmdline(command.Process.Pid); err != nil {
					log.Errorf(ctx, "get process cmdline failed, %s", err.Error())
				} else {
					if cmdline == "" {
						log.Errorf(ctx, "unknown err, process exit.")
						break
					}
				}

				log.Infof(ctx, "wait nsexec process resume, current comm: %s, pid: %d", comm, command.Process.Pid)
				if comm == "nsexec\n" {
					break
				}
			}
		}
	}
	return spec.ReturnSuccess(command.Process.Pid)
}

func getProcessComm(pid int) (string, error) {
	f, err := os.Open(fmt.Sprintf("%s/%d/comm", "/proc", pid))
	if err != nil {
		return "", err
	}
	defer f.Close()

	b, err := io.ReadAll(f)
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

	b, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}

	return string(b), nil
}
