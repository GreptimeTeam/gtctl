// Copyright 2023 Greptime Team
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

package logger

import (
	"bytes"
	"fmt"
	"io"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/fatih/color"
	"sigs.k8s.io/kind/pkg/log"
)

// Logger is the new interface and base on log.Logger that from kind project.
type Logger interface {
	log.Logger
}

// logger is implementation of Logger interface.
// The implementation of logger is based on the 'kind/pkg/internal/cli/logger.go' file.
type logger struct {
	writer     io.Writer
	writerMu   sync.Mutex
	verbosity  log.Level
	bufferPool *bufferPool
	colored    bool
}

var _ Logger = &logger{}

type Option func(*logger)

func Bold(s string) string {
	return color.New(color.FgHiWhite, color.Bold).SprintfFunc()(s)
}

// New returns a new logger with the given verbosity.
func New(writer io.Writer, verbosity log.Level, opts ...Option) Logger {
	l := &logger{
		writer:     writer,
		verbosity:  verbosity,
		bufferPool: newBufferPool(),
	}

	for _, opt := range opts {
		opt(l)
	}

	return l
}

func WithColored() Option {
	return func(l *logger) {
		l.colored = true
	}
}

// Warn is part of the log.logger interface.
func (l *logger) Warn(message string) {
	if l.colored {
		// Output in yellow.
		message = fmt.Sprintf("\x1b[33m%s\x1b[0m", message)
	}
	l.print(message)
}

// Warnf is part of the log.logger interface.
func (l *logger) Warnf(format string, args ...interface{}) {
	if l.colored {
		// Output in yellow.
		format = fmt.Sprintf("\x1b[33m%s\x1b[0m", format)
	}
	l.printf(format, args...)
}

// Error is part of the log.logger interface.
func (l *logger) Error(message string) {
	if l.colored {
		// Output in red.
		message = fmt.Sprintf("\x1b[31m%s\x1b[0m", message)
	}
	l.print(message)
}

// Errorf is part of the log.logger interface.
func (l *logger) Errorf(format string, args ...interface{}) {
	if l.colored {
		// Output in red.
		format = fmt.Sprintf("\x1b[31m%s\x1b[0m", format)
	}
	l.printf(format, args...)
}

// V is part of the log.logger interface.
func (l *logger) V(level log.Level) log.InfoLogger {
	return infoLogger{
		logger:  l,
		level:   level,
		enabled: level <= l.getVerbosity(),
	}
}

// SetVerbosity sets the loggers verbosity.
func (l *logger) SetVerbosity(verbosity log.Level) {
	atomic.StoreInt32((*int32)(&l.verbosity), int32(verbosity))
}

// infoLogger implements log.InfoLogger for logger.
type infoLogger struct {
	logger  *logger
	level   log.Level
	enabled bool
}

// Enabled is part of the log.InfoLogger interface.
func (i infoLogger) Enabled() bool {
	return i.enabled
}

// Info is part of the log.InfoLogger interface.
func (i infoLogger) Info(message string) {
	if !i.enabled {
		return
	}
	// for > 0, we are writing debug messages, include extra info
	if i.level > 0 {
		i.logger.debug(message)
	} else {
		i.logger.print(message)
	}
}

// Infof is part of the log.InfoLogger interface.
func (i infoLogger) Infof(format string, args ...interface{}) {
	if !i.enabled {
		return
	}
	// for > 0, we are writing debug messages, include extra info.
	if i.level > 0 {
		i.logger.debugf(format, args...)
	} else {
		i.logger.printf(format, args...)
	}
}

// synchronized write to the inner writer
func (l *logger) write(p []byte) (n int, err error) {
	l.writerMu.Lock()
	defer l.writerMu.Unlock()
	return l.writer.Write(p)
}

// writeBuffer writes buf with write, ensuring there is a trailing newline.
func (l *logger) writeBuffer(buf *bytes.Buffer) {
	// ensure trailing newline
	if buf.Len() == 0 || buf.Bytes()[buf.Len()-1] != '\n' {
		buf.WriteByte('\n')
	}
	// TODO: should we handle this somehow??
	// Who logs for the logger? 🤔
	_, _ = l.write(buf.Bytes())
}

// print writes a simple string to the log writer.
func (l *logger) print(message string) {
	buf := bytes.NewBufferString(message)
	l.writeBuffer(buf)
}

// printf is roughly fmt.Fprintf against the log writer.
func (l *logger) printf(format string, args ...interface{}) {
	buf := l.bufferPool.Get()
	fmt.Fprintf(buf, format, args...)
	l.writeBuffer(buf)
	l.bufferPool.Put(buf)
}

// debug is like print but with a debug log header.
func (l *logger) debug(message string) {
	buf := l.bufferPool.Get()
	l.addDebugHeader(buf)
	if l.colored {
		// Output in blue.
		message = fmt.Sprintf("\x1b[34m%s\x1b[0m", message)
	}
	buf.WriteString(message)
	l.writeBuffer(buf)
	l.bufferPool.Put(buf)
}

// debugf is like printf but with a debug log header.
func (l *logger) debugf(format string, args ...interface{}) {
	buf := l.bufferPool.Get()
	l.addDebugHeader(buf)
	if l.colored {
		// Output in blue.
		format = fmt.Sprintf("\x1b[34m%s\x1b[0m", format)
	}
	fmt.Fprintf(buf, format, args...)
	l.writeBuffer(buf)
	l.bufferPool.Put(buf)
}

// addDebugHeader inserts the debug line header to buf.
func (l *logger) addDebugHeader(buf *bytes.Buffer) {
	_, file, line, ok := runtime.Caller(3)
	// lifted from klog
	if !ok {
		file = "???"
		line = 1
	} else {
		if slash := strings.LastIndex(file, "/"); slash >= 0 {
			path := file
			file = path[slash+1:]
			if dirsep := strings.LastIndex(path[:slash], "/"); dirsep >= 0 {
				file = path[dirsep+1:]
			}
		}
	}
	buf.Grow(len(file) + 11) // we know at least this many bytes are needed
	if l.colored {
		// Output in blue.
		buf.WriteString("\x1b[34m")
	}
	buf.WriteString("DEBUG: ")
	buf.WriteString(file)
	buf.WriteByte(':')
	fmt.Fprintf(buf, "%d", line)
	buf.WriteByte(']')
	buf.WriteByte(' ')
	if l.colored {
		// Reset color.
		buf.WriteString("\x1b[0m")
	}
}

func (l *logger) getVerbosity() log.Level {
	return log.Level(atomic.LoadInt32((*int32)(&l.verbosity)))
}

// bufferPool is a type safe sync.Pool of *byte.Buffer, guaranteed to be Reset.
type bufferPool struct {
	sync.Pool
}

// newBufferPool returns a new bufferPool
func newBufferPool() *bufferPool {
	return &bufferPool{
		sync.Pool{
			New: func() interface{} {
				// The Pool's New function should generally only return pointer
				// types, since a pointer can be put into the return interface
				// value without an allocation.
				return new(bytes.Buffer)
			},
		},
	}
}

// Get obtains a buffer from the pool.
func (b *bufferPool) Get() *bytes.Buffer {
	return b.Pool.Get().(*bytes.Buffer)
}

// Put returns a buffer to the pool, resetting it first.
func (b *bufferPool) Put(x *bytes.Buffer) {
	// only store small buffers to avoid pointless allocation
	// avoid keeping arbitrarily large buffers
	if x.Len() > 256 {
		return
	}
	x.Reset()
	b.Pool.Put(x)
}
