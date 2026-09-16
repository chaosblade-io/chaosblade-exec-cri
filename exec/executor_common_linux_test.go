//go:build linux

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

// Coverage:
//  1. The real CRI startup entry fails when mem lacks memory or cpu lacks
//     cpu/cpuacct, and exec.Cmd.Process must stay nil (no pressure process).
//  2. With all required controllers present and only optional hugetlb missing,
//     the command starts and is written to the target cgroup.procs.
//  3. When cgroup attachment fails, the started process is killed and reaped.
//
// Approach: plain temp dirs stand in for cgroups while the production startup
// function is exercised. Negative and positive cases use /bin/true (no real
// pressure, no real cgroup change); the attach-failure case starts sleep and
// occupies cgroup.procs with a directory to make the write fail.
package exec

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/containerd/cgroups"
)

func commandHierarchy(t *testing.T, names ...string) (string, cgroups.Hierarchy) {
	t.Helper()
	root := t.TempDir()
	subsystems := []cgroups.Subsystem{cgroups.NewNamed(root, "hugetlb")}
	for _, name := range names {
		subsystems = append(subsystems, cgroups.NewNamed(root, cgroups.Name(name)))
		if err := os.MkdirAll(filepath.Join(root, name, "target"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return root, func() ([]cgroups.Subsystem, error) { return subsystems, nil }
}

func TestStartV1CommandRejectsMissingControllers(t *testing.T) {
	for _, tc := range []struct {
		target  string
		names   []string
		missing string
	}{
		{"mem", []string{"pids"}, "memory"},
		{"cpu", []string{"pids", "cpuacct"}, "cpu"},
		{"cpu", []string{"pids", "cpu"}, "cpuacct"},
		{"cpu", []string{"pids"}, "cpu, cpuacct"},
	} {
		t.Run(tc.target+"/"+tc.missing, func(t *testing.T) {
			_, hierarchy := commandHierarchy(t, tc.names...)
			command := exec.Command("/bin/true")
			err := startV1Command(command, hierarchy, cgroups.StaticPath("/target"), tc.target)
			if command.Process != nil {
				_ = command.Process.Kill()
				_ = command.Wait()
				t.Fatal("process started before required-controller validation")
			}
			if err == nil || !strings.Contains(err.Error(), "requires active controller(s): "+tc.missing) {
				t.Fatalf("expected missing %s error, got %v", tc.missing, err)
			}
		})
	}
}

func TestStartV1CommandAllowsMissingOptionalController(t *testing.T) {
	for _, tc := range []struct {
		target string
		names  []string
	}{
		{"mem", []string{"memory"}},
		{"cpu", []string{"cpu", "cpuacct"}},
	} {
		t.Run(tc.target, func(t *testing.T) {
			root, hierarchy := commandHierarchy(t, tc.names...)
			// The real kernel would provide cgroup.procs; pre-creating the file
			// also removes host umask dependence so the test runs unprivileged.
			for _, name := range tc.names {
				if err := os.WriteFile(filepath.Join(root, name, "target", "cgroup.procs"), nil, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			command := exec.Command("/bin/true")
			if err := startV1Command(command, hierarchy, cgroups.StaticPath("/target"), tc.target); err != nil {
				t.Fatal(err)
			}
			defer command.Wait()
			for _, name := range tc.names {
				data, err := os.ReadFile(filepath.Join(root, name, "target", "cgroup.procs"))
				if err != nil || string(data) != strconv.Itoa(command.Process.Pid) {
					t.Fatalf("%s: target cgroup did not receive process: %q, %v", name, data, err)
				}
			}
		})
	}
}

func TestStartV1CommandReapsProcessOnAddFailure(t *testing.T) {
	root, hierarchy := commandHierarchy(t, "memory")
	if err := os.Mkdir(filepath.Join(root, "memory", "target", "cgroup.procs"), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("/bin/sleep", "30")
	err := startV1Command(command, hierarchy, cgroups.StaticPath("/target"), "mem")
	t.Cleanup(func() {
		if command.Process != nil && command.ProcessState == nil {
			_ = command.Process.Kill()
			_ = command.Wait()
		}
	})
	if err == nil || !strings.Contains(err.Error(), "add process to cgroups V1 failed") {
		t.Fatalf("expected attach failure, got %v", err)
	}
	if command.ProcessState == nil || command.ProcessState.Success() {
		t.Fatal("failed attach left process running or unreaped")
	}
}
