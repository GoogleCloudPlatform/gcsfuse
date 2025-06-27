// Copyright 2025 Google LLC
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

package logger

import (
	"fmt"
	"io"
	"os"
	"sync"

	"gopkg.in/natefinch/lumberjack.v2"
)

// AsyncLogger wraps a lumberjack.Logger to provide asynchronous, non-blocking writes.
// It implements the io.WriteCloser interface.
type AsyncLogger struct {
	lumberjackLogger io.WriteCloser
	logChan          chan []byte
	wg               sync.WaitGroup
}

// NewAsyncLogger creates a new AsyncLogger.
// lumberjackLogger is the underlying logger for file rotation.
// bufferSize is the number of log messages that can be buffered. If the buffer
// is full, new log messages will be dropped to prevent the application from blocking.
func NewAsyncLogger(lumberjackLogger *lumberjack.Logger, bufferSize int) *AsyncLogger {
	if bufferSize <= 0 {
		// Provide a sensible default if an invalid size is given.
		bufferSize = 1024
	}

	al := &AsyncLogger{
		lumberjackLogger: lumberjackLogger,
		logChan:          make(chan []byte, bufferSize),
	}

	al.startWriter()
	return al
}

// startWriter starts the background goroutine that writes logs to the lumberjack logger.
func (al *AsyncLogger) startWriter() {
	al.wg.Add(1)
	go func() {
		defer al.wg.Done()
		for logMsg := range al.logChan {
			_, err := al.lumberjackLogger.Write(logMsg)
			if err != nil {
				// If the write to lumberjack fails, we write the error to stderr.
				// This helps in debugging disk issues or permissions.
				fmt.Fprintf(os.Stderr, "asynclogger: failed to write log: %v\n", err)
			}
		}
	}()
}

// Write implements the io.Writer interface.
// It sends the log message to a channel to be written asynchronously.
// This is a non-blocking call. If the channel buffer is full, the log message is dropped.
func (al *AsyncLogger) Write(p []byte) (n int, err error) {
	// The log package may reuse the buffer `p`, so we must make a copy.
	msgCopy := make([]byte, len(p))
	copy(msgCopy, p)

	select {
	case al.logChan <- msgCopy:
		// The message was successfully sent to the channel.
	default:
		// The channel is full. We drop the message to avoid blocking.
		fmt.Fprintln(os.Stderr, "asynclogger: log buffer is full, dropping message.")
	}

	// Always return success to the caller. The actual file write is handled in the background.
	return len(p), nil
}

// Close flushes all buffered logs to the underlying lumberjack logger and closes it.
// It's essential to call Close before the application exits to prevent log loss.
func (al *AsyncLogger) Close() error {
	close(al.logChan)
	al.wg.Wait()
	return al.lumberjackLogger.Close()
}
