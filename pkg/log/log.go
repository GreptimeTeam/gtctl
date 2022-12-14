// Copyright 2022 Greptime Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package log

import (
	"fmt"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
)

type Logger interface {
	Info(message string)
	Infof(format string, args ...interface{})

	Warn(message string)
	Warnf(format string, args ...interface{})

	Error(message string)
	Errorf(format string, args ...interface{})

	Debug(message string)
	Debugf(format string, args ...interface{})
}

var _ Logger = &logger{}

type logger struct {
	enableDebug bool
}

type LoggerOption func(*logger)

func NewLogger(opts ...LoggerOption) Logger {
	l := &logger{}

	for _, opt := range opts {
		opt(l)
	}

	return l
}

func WithDebug() LoggerOption {
	return func(l *logger) {
		l.enableDebug = true
	}
}

func Bold(s string) string {
	return color.New(color.FgHiWhite, color.Bold).SprintfFunc()(s)
}

func StartSpinning(suffix string, processFunc func() error) error {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = fmt.Sprintf("  %s  ", suffix)

	if err := s.Color("fgHiWhite", "bold"); err != nil {
		return err
	}

	s.Start() // Start the spinner
	if err := processFunc(); err != nil {
		s.Stop()
		return err
	}
	s.Stop()
	return nil
}

func (l *logger) Info(message string) {
	l.infof(message)
}

func (l *logger) Infof(format string, args ...interface{}) {
	l.infof(format, args...)
}

func (l *logger) Warn(message string) {
	l.warnf(message)
}

func (l *logger) Warnf(format string, args ...interface{}) {
	l.warnf(format, args...)
}

func (l *logger) Error(message string) {
	l.errorf(message)
}

func (l *logger) Errorf(format string, args ...interface{}) {
	l.errorf(format, args...)
}

func (l *logger) Debug(message string) {
	if l.enableDebug {
		l.debugf(message)
	}
}

func (l *logger) Debugf(format string, args ...interface{}) {
	if l.enableDebug {
		l.debugf(format, args...)
	}
}

func (l *logger) infof(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

func (l *logger) errorf(format string, args ...interface{}) {
	boldWhite := color.New(color.FgHiWhite, color.Bold).PrintfFunc()
	boldWhite(format, args...)
}

func (l *logger) warnf(format string, args ...interface{}) {
	white := color.New(color.FgWhite).PrintfFunc()
	white(format, args...)
}

func (l *logger) debugf(format string, args ...interface{}) {
	white := color.New(color.FgWhite).PrintfFunc()
	white(format, args...)
}
