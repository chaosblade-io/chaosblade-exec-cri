package container

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/chaosblade-io/chaosblade-spec-go/spec"
	containertype "github.com/docker/docker/api/types/container"
	cri "k8s.io/cri-api/pkg/apis"
	v1 "k8s.io/cri-api/pkg/apis/runtime/v1"
)

type Client struct {
	endpoint string
	timeout  time.Duration
}

const verbose = false

func (c *Client) GetPidById(ctx context.Context, containerId string) (int32, error, int32) {
	runtimeService, err := GetRuntimeService(ctx, c.endpoint, c.timeout)
	if err != nil {
		return -1, fmt.Errorf(spec.ContainerExecFailed.Sprintf("GetRuntimeService", err.Error())), spec.ContainerExecFailed.Code
	}

	r, err := runtimeService.ContainerStatus(ctx, containerId, verbose)
	if err != nil {
		return 0, nil, 0
	}

	pid, err := strconv.ParseInt(r.Info["pid"], 10 ,32)
	if err != nil || pid <= 0 {
		return 0, nil, 0
	}
	return int32(pid), nil, spec.OK.Code
}


func (c *Client) GetContainerById(ctx context.Context, runtimeService cri.RuntimeService, containerId string) (*v1.ContainerStats, error, int32) {
	// todo ，直接返回v1定义的containerinfo，后续引用的地方都改一下，之前是因为不同的runtime需要统一标准，但这里不需要了，已经统一过了
	containerStats, err := runtimeService.ContainerStats(ctx, containerId)
	if err != nil {
		return containerStats, fmt.Errorf(spec.ContainerExecFailed.Sprintf("GetRuntimeService", err.Error())), spec.ContainerExecFailed.Code
	}
	return containerStats, nil, spec.OK.Code
}

func (c *Client) GetContainerByName(ctx context.Context, runtimeService cri.RuntimeService, containerName string) (*v1.ContainerStats, error, int32) {
	//runtimeService, err := GetRuntimeService(ctx, c.endpoint, c.timeout)
	//if err != nil {
	//	return nil, fmt.Errorf(spec.ContainerExecFailed.Sprintf("GetRuntimeService", err.Error())), spec.ContainerExecFailed.Code
	//}

	containers, err := runtimeService.ListContainerStats(ctx, &v1.ContainerStatsFilter{})
	if err != nil {
		return nil, fmt.Errorf("GetContainerByName by `%s`, failed, %v", containerName, err), spec.CriExecNotFound.Code
	}

	for _,container := range containers {
		if container.Attributes.Metadata.GetName() == containerName {
			return container, nil, spec.OK.Code
		}
	}
	return nil, fmt.Errorf("GetContainerByName by `%s` not found", containerName), spec.CriExecNotFound.Code
}

func (c *Client) GetContainerByLabelSelector(ctx context.Context, runtimeService cri.RuntimeService, labels map[string]string) (*v1.ContainerStats, error, int32) {

	filter := &v1.ContainerStatsFilter{
		LabelSelector: labels,
	}
	lists, err := runtimeService.ListContainerStats(ctx, filter)
	if err != nil || len(lists) == 0 {
		return nil, fmt.Errorf(spec.ContainerExecFailed.Sprintf("ListContainers", err.Error())), spec.ContainerExecFailed.Code
	}
	return lists[0], nil, spec.OK.Code
}

func (c *Client) RemoveContainer(ctx context.Context, runtimeService cri.RuntimeService, containerId string, force bool) error {
	return runtimeService.RemoveContainer(ctx, containerId)
}
// cri api can not support pause container
//func (c *Client) PauseCotainer(ctx context.Context, runtimeService cri.RuntimeService, containerId string) error {
//	runtimeService.RemoveContainer()
//	runtimeService.
//}
//
//
//func (c *Client) UnpauseCotainer(ctx context.Context, containerId string) error {
//
//}
func (c *Client) CopyToContainer(ctx context.Context, runtimeService cri.RuntimeService, containerId, srcFile, dstPath, extractDirName string, override bool) error {
	// todo 是否可用exec 输入
}
func (c *Client) ExecContainer(ctx context.Context, runtimeService cri.RuntimeService, containerId, command string) (output string, err error) {
	req := &v1.ExecRequest{
		ContainerId: containerId,
		Cmd:         []string{"sh", "-c", command},
		Tty:         false,
		Stdin:       false,
		Stdout:      true,
		Stderr:      true,
	}
	runtimeService.Exec(ctx, req)

}
func (c *Client) ExecuteAndRemove(ctx context.Context, config *containertype.Config, hostConfig *containertype.HostConfig,
	func (c *Client) networkConfig *network.NetworkingConfig, containerName string, removed bool, timeout time.Duration,
	func (c *Client) command string, containerInfo ContainerInfo) (containerId string, output string, err error, code int32) {

}