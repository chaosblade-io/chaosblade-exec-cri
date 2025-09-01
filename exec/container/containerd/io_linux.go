//go:build linux

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

	"github.com/containerd/containerd/cio"
)

func newDirectIO(ctx context.Context, execId string, terminal bool) (*directIO, error) {
	fifos, err := cio.NewFIFOSetInDir("/run/containerd/fifo", execId, terminal)
	if err != nil {
		return nil, err
	}

	// Linux implementation - NewDirectIO returns (DirectIO, error)
	dio, err := cio.NewDirectIO(ctx, fifos)
	if err != nil {
		return nil, err
	}

	return &directIO{DirectIO: *dio}, nil
}
