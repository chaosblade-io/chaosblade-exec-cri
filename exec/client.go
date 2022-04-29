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
	"github.com/chaosblade-io/chaosblade-spec-go/log"
	"io/ioutil"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

var cli *Client

type Client struct {
	client *client.Client
}

// waitAndGetOutput returns the result
func (c *Client) waitAndGetOutput(ctx context.Context, containerId string) (string, error) {
	containerWait()
	resp, err := c.client.ContainerLogs(context.Background(), containerId, types.ContainerLogsOptions{
		ShowStderr: true,
		ShowStdout: true,
	})
	if err != nil {
		log.Warnf(ctx, "Get container: %s log err: %s", containerId, err)
		return "", err
	}
	defer resp.Close()
	bytes, err := ioutil.ReadAll(resp)
	return string(bytes), err
}

func containerWait() error {
	timer := time.NewTimer(500 * time.Millisecond)
	select {
	case <-timer.C:
	}
	return nil
}

//GetImageInspectById
func (c *Client) getImageInspectById(imageId string) (types.ImageInspect, error) {
	inspect, _, err := c.client.ImageInspectWithRaw(context.Background(), imageId)
	return inspect, err
}

//DeleteImageByImageId
func (c *Client) deleteImageByImageId(imageId string) error {
	_, err := c.client.ImageRemove(context.Background(), imageId, types.ImageRemoveOptions{
		Force:         false,
		PruneChildren: true,
	})
	return err
}
