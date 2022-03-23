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
package container

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/chaosblade-io/chaosblade-spec-go/spec"
	"github.com/chaosblade-io/chaosblade-spec-go/util"
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"path"
	"strings"
)

func CopyToContainer(pid uint32, srcFile, dstPath, extractDirName string, override bool) error {

	args := fmt.Sprintf("-t %d -p -m -- /bin/sh -c", pid)
	argsArray := strings.Split(args, " ")
	nsbin := path.Join(util.GetProgramPath(), "bin", spec.NSExecBin)

	command := fmt.Sprintf("cat > %s", path.Join(dstPath, path.Base(srcFile)))
	logrus.Infoln("run copy cmd: ", nsbin, args, command)

	cmd := exec.Command(nsbin, append(argsArray, command)...)

	var outMsg bytes.Buffer
	var errMsg bytes.Buffer
	cmd.Stdout = &outMsg
	cmd.Stderr = &errMsg

	open, err := os.Open(srcFile)
	defer open.Close()
	if err != nil {
		return err
	}
	cmd.Stdin = open
	if err := cmd.Start(); err != nil {
		return err
	}
	if err := cmd.Wait(); err != nil {
		return err
	}
	logrus.Debugf("Command Result, output: %s, errMsg: %s", outMsg.String(), errMsg.String())

	if errMsg.Len() != 0 {
		return errors.New(errMsg.String())
	}

	// tar -zxf
	command = fmt.Sprintf("-t %d -p -m -- tar -zxf %s -C %s", pid, path.Join(dstPath, path.Base(srcFile)), dstPath)
	logrus.Infoln("run tar cmd: ", nsbin, command)
	cmd = exec.Command(nsbin, strings.Split(command, " ")...)
	//
	var outMsg2 bytes.Buffer
	var errMsg2 bytes.Buffer
	cmd.Stdout = &outMsg2
	cmd.Stderr = &errMsg2
	err = cmd.Run()
	logrus.Debugf("Tar Command Result, output: %s, errMsg: %s,  err: %v", outMsg.String(), errMsg.String(), err)

	if err != nil {
		return err
	}

	if errMsg2.Len() != 0 {
		return errors.New(errMsg.String())
	}

	return nil
}

func ExecContainer(pid int32, command string) (output string, err error) {

	args := fmt.Sprintf("-t %d -p -m -- /bin/sh -c", pid)
	argsArray := strings.Split(args, " ")
	nsbin := path.Join(util.GetProgramPath(), "bin", spec.NSExecBin)

	logrus.Infoln("cxec container cmd: ", nsbin, args, command)

	cmd := exec.Command(nsbin, append(argsArray, command)...)

	var outMsg bytes.Buffer
	var errMsg bytes.Buffer
	cmd.Stdout = &outMsg
	cmd.Stderr = &errMsg
	err = cmd.Run()

	logrus.Debugf("Command Result, output: %s, errMsg: %s, err: %v", outMsg.String(), errMsg.String(), err)

	if err != nil {
		return "", err
	}
	if errMsg.Len() > 0 {
		return errMsg.String(), nil
	}

	return outMsg.String(), nil
}
