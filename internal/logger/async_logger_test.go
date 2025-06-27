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
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	// "strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/natefinch/lumberjack.v2"
)

// setupTest creates a temporary directory and returns its path and a cleanup function.
func setupTest(t *testing.T) (string, func()) {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "async-logger-test-*")
	require.NoError(t, err)

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return tempDir, cleanup
}

// captureStderr captures everything written to os.Stderr during the execution of a function.
func captureStderr(f func()) string {
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	defer func() {
		os.Stderr = oldStderr
	}()

	f()
	w.Close()

	var stderrBuf bytes.Buffer
	io.Copy(&stderrBuf, r)
	r.Close()
	return stderrBuf.String()
}

func TestAsyncLogger_WriteAndClose(t *testing.T) {
	// Arrange
	tempDir, cleanup := setupTest(t)
	defer cleanup()
	logPath := filepath.Join(tempDir, "test.log")
	lj := &lumberjack.Logger{Filename: logPath}
	asyncLogger := NewAsyncLogger(lj, 10)

	// Act
	fmt.Fprintln(asyncLogger, "message 1")
	fmt.Fprintln(asyncLogger, "message 2")
	fmt.Fprintln(asyncLogger, "message 3")
	err := asyncLogger.Close()

	// Assert
	require.NoError(t, err)
	content, err := os.ReadFile(logPath)
	require.NoError(t, err)
	expected := "message 1\nmessage 2\nmessage 3\n"
	assert.Equal(t, expected, string(content))
}

// func TestAsyncLogger_DropMessageWhenBufferFull(t *testing.T) {
// 	// Arrange
// 	tempDir, cleanup := setupTest(t)
// 	defer cleanup()
// 	logPath := filepath.Join(tempDir, "test.log")
// 	lj := &lumberjack.Logger{Filename: logPath}
// 	bufferSize := 2
// 	asyncLogger := NewAsyncLogger(lj, bufferSize)

// 	// Act
// 	// Capture stderr to check for the "dropping message" warning.
// 	// We write more messages than the buffer can hold in a tight loop
// 	// to increase the chance of triggering the drop logic.
// 	var capturedOutput string
// 	act := func() {
// 		numMessages := 20
// 		for i := 0; i < numMessages; i++ {
// 			fmt.Fprintf(asyncLogger, "message %d\n", i)
// 		}
// 		err := asyncLogger.Close()
// 		require.NoError(t, err)
// 	}
// 	capturedOutput = captureStderr(act)

// 	// Assert
// 	assert.Contains(t, capturedOutput, "asynclogger: log buffer is full, dropping message.")
// 	// Because of the race between the write loop and the writer goroutine, we can't
// 	// writer goroutine, we can't know exactly how many messages made it.
// 	// We assert that it's more than the buffer size but less than the total
// 	// number of messages attempted.
// 	content, err := os.ReadFile(logPath)
// 	require.NoError(t, err)
// 	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
// 	assert.Greater(t, len(lines), bufferSize, "at least bufferSize messages should be written")
// 	assert.Less(t, len(lines), 20, "not all messages should have been written")
// }
