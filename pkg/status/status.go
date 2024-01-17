// Copyright 2023 Greptime Team
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

package status

import (
	"fmt"
	"time"

	"github.com/briandowns/spinner"
)

// The spinner frames is from kind project.
var spinnerFrames = []string{
	"⠈⠁",
	"⠈⠑",
	"⠈⠱",
	"⠈⡱",
	"⢀⡱",
	"⢄⡱",
	"⢄⡱",
	"⢆⡱",
	"⢎⡱",
	"⢎⡰",
	"⢎⡠",
	"⢎⡀",
	"⢎⠁",
	"⠎⠁",
	"⠊⠁",
}

const (
	defaultDelay = 100 * time.Millisecond
)

type Spinner struct {
	spinner *spinner.Spinner
}

func NewSpinner() (*Spinner, error) {
	s := spinner.New(spinnerFrames, defaultDelay)
	if err := s.Color("fgHiWhite", "bold"); err != nil {
		return nil, err
	}
	return &Spinner{
		spinner: s,
	}, nil
}

func (s *Spinner) Start(status string) {
	s.spinner.Start()
	s.spinner.Suffix = fmt.Sprintf(" %s", status)
}

func (s *Spinner) Stop(success bool, status string) {
	if success {
		s.spinner.FinalMSG = fmt.Sprintf(" \x1b[32m✓\x1b[0m %s\n", status)
	} else {
		s.spinner.FinalMSG = fmt.Sprintf(" \x1b[31m✗\x1b[0m %s 😵‍💫\n", status)
	}
	s.spinner.Stop()
}
