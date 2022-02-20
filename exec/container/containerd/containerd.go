/*
 * Copyright 1999-2020 Alibaba Group Holding Ltd.
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
package containerd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/chaosblade-io/chaosblade-spec-go/spec"
	"github.com/chaosblade-io/chaosblade-spec-go/util"
	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/containers"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
	ctrdutil "github.com/containerd/containerd/pkg/cri/util"
	"github.com/containerd/containerd/runtime/v2/runc/options"
	"github.com/containerd/containerd/snapshots"
	containertype "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"

	"github.com/chaosblade-io/chaosblade-exec-cri/exec/container"
)

const (
	connectionTimeout = 2 * time.Second
	baseBackoffDelay  = 100 * time.Millisecond
	maxBackoffDelay   = 3 * time.Second
)

const (
	DefaultStateDir = "/run/containerd"
	// DefaultAddress is the default unix socket address
	DefaultUinxAddress = DefaultStateDir + "/containerd.sock"
	DefaultRuntime     = "io.containerd.runc.v2"
	DefaultSnapshotter = "overlayfs"

	DefaultContainerdNS = "k8s.io"

	NetworkNsType = "network"
)

var cli *Client

type Client struct {
	cclient *containerd.Client

	Ctx    context.Context
	Cancel context.CancelFunc
	connMu sync.Mutex
}

func NewClient(endpoint, namespace string) (*Client, error) {
	if cli != nil {
		if ok, _ := cli.cclient.IsServing(cli.Ctx); ok {
			return cli, nil
		}
	}

	if endpoint == "" {
		endpoint = DefaultUinxAddress
	}
	if namespace == "" {
		namespace = DefaultContainerdNS
	}
	cclient, err := containerd.New(endpoint, containerd.WithDefaultNamespace(namespace))
	if err != nil {
		return nil, err
	}
	var (
		ctx    = context.Background()
		cancel context.CancelFunc
	)
	ctx = namespaces.WithNamespace(ctx, namespace)
	ctx, cancel = context.WithCancel(ctx)
	cli = &Client{
		cclient: cclient,
		connMu:  sync.Mutex{},
		Ctx:     ctx,
		Cancel:  cancel,
	}
	return cli, nil
}

func (c *Client) GetPidById(containerId string) (int32, error, int32) {

	ctx := context.Background()
	container, err := c.cclient.LoadContainer(ctx, containerId)
	if err != nil {
		return -1, fmt.Errorf(spec.ContainerExecFailed.Sprintf("GetContainerList", err.Error())), spec.ContainerExecFailed.Code
	}
	task, err := container.Task(ctx, nil)
	if err != nil {
		return -1, fmt.Errorf(spec.ContainerExecFailed.Sprintf("GetContainerList", err.Error())), spec.ContainerExecFailed.Code
	}

	return int32(task.Pid()), nil, spec.OK.Code
}

func (c *Client) GetContainerById(containerId string) (container.ContainerInfo, error, int32) {
	if c.cclient == nil {
		return container.ContainerInfo{}, errors.New("containerd client is not available"), spec.ContainerExecFailed.Code
	}

	containerDetail, err := c.cclient.ContainerService().Get(c.Ctx, containerId)
	if err != nil {
		return container.ContainerInfo{}, err, spec.ContainerExecFailed.Code
	}

	return convertContainerInfo(containerDetail), nil, spec.OK.Code
}

func (c *Client) GetContainerByName(containerName string) (container.ContainerInfo, error, int32) {
	// containerd have not name. so maybe it is not usefull
	filters := []string{fmt.Sprintf("runtime,name==%s", containerName)}
	containerDetails, err := c.cclient.ContainerService().List(c.Ctx, filters...)
	if err != nil {
		return container.ContainerInfo{}, err, spec.ContainerExecFailed.Code
	}

	return convertContainerInfo(containerDetails[0]), nil, spec.OK.Code
}

func convertContainerInfo(containerDetail containers.Container) container.ContainerInfo {
	return container.ContainerInfo{
		ContainerId:   containerDetail.ID,
		ContainerName: containerDetail.Labels["io.kubernetes.container.name"],
		//Env:             spec.Process.Env,
		Labels: containerDetail.Labels,
		Spec:   containerDetail.Spec,
	}
}
func (c *Client) RemoveContainer(containerId string, force bool) error {
	err := c.cclient.ContainerService().Delete(c.Ctx, containerId)
	if err == nil {
		return nil
	}

	if errdefs.IsNotFound(err) {
		return nil
	}

	return err
}

func (c *Client) CopyToContainer(containerId, srcFile, dstPath, extractDirName string, override bool) error {

	containerDetail, err := c.cclient.LoadContainer(c.Ctx, containerId)
	if err != nil {
		return err
	}

	task, err := containerDetail.Task(c.Ctx, nil)
	if err != nil {
		return err
	}

	processId := task.Pid()

	return container.CopyToContainer(strconv.Itoa(int(processId)), srcFile, dstPath, extractDirName, override)
}

func (c *Client) ExecContainer(containerId, command string) (output string, err error) {
	id, err, _ := c.GetPidById(containerId)
	if err != nil {
		return "", err
	}
	return container.ExecContainer(id, command)
}

//ExecuteAndRemove: create and start a container for executing a command, and remove the container
func (c *Client) ExecuteAndRemove(config *containertype.Config, hostConfig *containertype.HostConfig,
	networkConfig *network.NetworkingConfig, containerName string, removed bool, timeout time.Duration,
	command string, containerInfo container.ContainerInfo) (containerId string, output string, err error, code int32) {

	snapshotter := DefaultSnapshotter

	// 1. get container network namespace path
	var specInfo specs.Spec
	json.Unmarshal(containerInfo.Spec.Value, &specInfo)
	specNS := specInfo.Linux.Namespaces
	var networkNsPath string
	for _, nsInfo := range specNS {
		if nsInfo.Type == NetworkNsType {
			networkNsPath = nsInfo.Path
		}
	}
	if networkNsPath == "" {
		return "", "", fmt.Errorf(spec.CreateContainerFailed.Sprintf("target container network namespace path is nil")), spec.CreateContainerFailed.Code
	}

	// 2. pull image befor create container
	if _, err := c.cclient.Pull(c.Ctx, config.Image, containerd.WithPullUnpack, containerd.WithPullSnapshotter(snapshotter)); err != nil {
		return "", "", fmt.Errorf(spec.ImagePullFailed.Sprintf(config.Image, err.Error())), spec.ImagePullFailed.Code
	}

	images, err := c.cclient.GetImage(c.Ctx, config.Image)
	if err != nil {
		return "", "", fmt.Errorf(spec.ImagePullFailed.Sprintf(config.Image, fmt.Sprintf("Get image failed, %s", err.Error()))), spec.ImagePullFailed.Code
	}

	unpacked, err := images.IsUnpacked(c.Ctx, snapshotter)
	if err != nil {
		return "", "", fmt.Errorf(spec.ImagePullFailed.Sprintf(config.Image, fmt.Sprintf("Get isUnpacked failed: %v", err))), spec.ImagePullFailed.Code
	}

	if !unpacked {
		if err := images.Unpack(c.Ctx, snapshotter); err != nil {
			return "", "", fmt.Errorf(spec.ImagePullFailed.Sprintf(config.Image, fmt.Sprintf("Unpack failed: %v", err))), spec.ImagePullFailed.Code
		}
	}

	// 3. generate container id
	containerId = util.GenerateContainerId()

	// 4. build opts
	var (
		opts     []oci.SpecOpts
		cOpts    []containerd.NewContainerOpts
		specOpts containerd.NewContainerOpts
	)

	opts = append(opts, oci.WithDefaultSpec(), oci.WithDefaultUnixDevices)
	opts = append(opts, withMount())
	opts = append(opts, oci.WithImageConfig(images))
	cOpts = append(cOpts, containerd.WithImage(images),
		containerd.WithSnapshotter(snapshotter),
		containerd.WithContainerLabels(config.Labels),
		containerd.WithImageName(config.Image))
	cOpts = append(cOpts, containerd.WithNewSnapshot(containerId, images, snapshots.WithLabels(make(map[string]string))))
	cOpts = append(cOpts, containerd.WithImageStopSignal(images, "SIGTERM"))

	opts = append(opts, oci.WithLinuxNamespace(specs.LinuxNamespace{Type: NetworkNsType, Path: networkNsPath}))
	opts = append(opts, oci.WithAddedCapabilities([]string{"CAP_NET_ADMIN"})) // ADD NET_ADMIN capabilities

	runtimeOpts, err := getRuntimeOptions()
	if err != nil {
		return "", "", fmt.Errorf(spec.CreateContainerFailed.Sprintf(fmt.Sprintf("Get runtime options failed: %v", err))), spec.CreateContainerFailed.Code
	}
	cOpts = append(cOpts, containerd.WithRuntime(DefaultRuntime, runtimeOpts))
	opts = append(opts, oci.WithAnnotations(config.Labels))

	var s specs.Spec
	specOpts = containerd.WithSpec(&s, opts...)
	cOpts = append(cOpts, specOpts)

	// 5. create new container
	var cntr containerd.Container
	if cntr, err = c.cclient.NewContainer(c.Ctx, containerId, cOpts...); err != nil {
		return "", "", fmt.Errorf(spec.CreateContainerFailed.Sprintf(err)), spec.CreateContainerFailed.Code
	}

	defer func() {
		deferCtx, deferCancel := ctrdutil.DeferContext()
		defer deferCancel()

		if err := cntr.Delete(deferCtx, containerd.WithSnapshotCleanup); err != nil {
			logrus.Warnf("Failed to delete containerd container %v, err: %v", containerId, err)
		}
	}()

	// 6. start a container that has been created
	task, err := c.NewTask(config.Image, cntr)
	if err != nil {
		return "", "", fmt.Errorf(spec.CreateContainerFailed.Sprintf(fmt.Sprintf("New task, %s", err.Error()))), spec.CreateContainerFailed.Code
	}
	defer func() {
		if _, err = task.Delete(c.Ctx); err != nil {
			logrus.Warnf("Failed to delete containerd task %v, err: %v", containerId, err)
		}
	}()

	tStatus, err := task.Wait(c.Ctx)
	if err != nil {
		return "", "", fmt.Errorf(spec.CreateContainerFailed.Sprintf(fmt.Sprintf("Task wait, %s", err.Error()))), spec.CreateContainerFailed.Code
	}

	if err = task.Start(c.Ctx); err != nil {
		return "", "", fmt.Errorf(spec.CreateContainerFailed.Sprintf(fmt.Sprintf("Task start, %s", err.Error()))), spec.CreateContainerFailed.Code
	}

	// 7. exec command in new container
	output, err = c.ExecContainer(containerId, command)
	if err != nil {
		return containerId, output, fmt.Errorf(spec.ContainerExecFailed.Sprintf(command, err)), spec.ContainerExecFailed.Code
	}

	if err := task.Kill(c.Ctx, syscall.SIGKILL); err != nil {
		return containerId, output, fmt.Errorf(spec.ContainerExecFailed.Sprintf(command, err)), spec.ContainerExecFailed.Code
	}

	<-tStatus

	return cntr.ID(), output, nil, spec.OK.Code
}

func (c *Client) NewTask(imageRef string, cntr containerd.Container) (containerd.Task, error) {
	var tOpts []containerd.NewTaskOpts

	ioCreator := cio.NullIO
	task, err := cntr.NewTask(c.Ctx, ioCreator, tOpts...)
	if err != nil {
		return nil, err
	}
	return task, nil
}
func (c *Client) Spec(ci container.ContainerInfo) (*oci.Spec, error) {
	var s oci.Spec
	if err := json.Unmarshal(ci.Spec.Value, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func getRuntimeOptions() (interface{}, error) {
	runtimeOpts := &options.Options{}

	return runtimeOpts, nil
}
func withMount() oci.SpecOpts {
	return func(ctx context.Context, client oci.Client, container *containers.Container, s *specs.Spec) error {
		mounts := make([]specs.Mount, 0)
		return oci.WithMounts(mounts)(ctx, client, container, s)
	}
}
