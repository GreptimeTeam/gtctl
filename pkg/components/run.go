// Copyright 2024 Greptime Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package components

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"path"
	"strconv"
	"sync"
	"syscall"

	"github.com/GreptimeTeam/gtctl/pkg/logger"
)

// RunOptions contains all the options for one component to run on bare-metal.
type RunOptions struct {
	Binary string
	Name   string

	pidDir string
	logDir string
	args   []string
}

func runBinary(ctx context.Context, stop context.CancelFunc,
	option *RunOptions, wg *sync.WaitGroup, logger logger.Logger) error {
	cmd := exec.CommandContext(ctx, option.Binary, option.args...)

	// output to binary.
	logFile := path.Join(option.logDir, "log")
	outputFile, err := os.Create(logFile)
	if err != nil {
		return err
	}

	outputFileWriter := bufio.NewWriter(outputFile)
	cmd.Stdout = outputFileWriter
	cmd.Stderr = outputFileWriter

	if err = cmd.Start(); err != nil {
		return err
	}

	pid := strconv.Itoa(cmd.Process.Pid)
	logger.V(3).Infof("run '%s' binary '%s' with args: '%v', log: '%s', pid: '%s'",
		option.Name, option.Binary, option.args, option.logDir, pid)

	pidFile := path.Join(option.pidDir, "pid")
	f, err := os.Create(pidFile)
	if err != nil {
		return err
	}

	_, err = f.Write([]byte(pid))
	if err != nil {
		return err
	}

	go func() {
		defer wg.Done()
		wg.Add(1)
		if err := cmd.Wait(); err != nil {
			// Caught signal kill and interrupt error then ignore.
			if exit, ok := err.(*exec.ExitError); ok {
				if status, ok := exit.Sys().(syscall.WaitStatus); ok && status.Signaled() {
					if status.Signal() == syscall.SIGKILL || status.Signal() == syscall.SIGINT {
						return
					}
				}
			}
			logger.Errorf("component '%s' binary '%s' (pid '%s') exited with error: %v", option.Name, option.Binary, pid, err)
			logger.Errorf("args: '%v'", option.args)
			_ = outputFileWriter.Flush()

			// If one component has failed, stop the whole context.
			stop()
		}
	}()

	return nil
}
