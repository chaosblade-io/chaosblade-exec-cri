/*
 * Copyright 2025 The ChaosBlade Authors
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
	"github.com/containerd/containerd/cio"
)

type directIO struct {
	cio.DirectIO
}

// Delete removes the underlying directory containing fifos
func (f *directIO) Delete() error {
	return f.DirectIO.Close()
}

// ioCreate returns IO available for use with task creation
func (f *directIO) IOCreate(id string) (cio.IO, error) {
	return f, nil
}

// ioAttach returns IO available for use with task attachment
func (f *directIO) IOAttach(set *cio.FIFOSet) (cio.IO, error) {
	return f, nil
}
