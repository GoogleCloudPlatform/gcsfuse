// Copyright 2023 Google Inc. All Rights Reserved.
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

// Provide a helper functions.
package operations

import (
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"syscall"
)

func CopyFile(srcFileName string, newFileName string) (err error) {
	if _, err = os.Stat(newFileName); err == nil {
		err = fmt.Errorf("Copied file %s already present", newFileName)
		return
	}

	source, err := os.OpenFile(srcFileName, syscall.O_DIRECT, FilePermission_0600)
	if err != nil {
		err = fmt.Errorf("File %s opening error: %v", srcFileName, err)
		return
	}

	// Closing file at the end.
	defer CloseFile(source)

	destination, err := os.OpenFile(newFileName, os.O_WRONLY|os.O_CREATE|syscall.O_DIRECT, FilePermission_0600)
	if err != nil {
		err = fmt.Errorf("Copied file creation error: %v", err)
		return
	}
	// Closing file at the end.
	defer CloseFile(destination)

	// File copying with io.Copy() utility.
	_, err = io.Copy(destination, source)
	if err != nil {
		err = fmt.Errorf("Error in file copying: %v", err)
		return
	}
	return
}

func ReadFile(filePath string) (content []byte, err error) {
	file, err := os.OpenFile(filePath, os.O_RDONLY|syscall.O_DIRECT, FilePermission_0600)
	if err != nil {
		err = fmt.Errorf("Error in the opening the file %v", err)
		return
	}

	// Closing file at the end.
	defer CloseFile(file)

	content, err = os.ReadFile(file.Name())
	if err != nil {
		err = fmt.Errorf("ReadAll: %v", err)
		return
	}
	return
}

func RenameFile(fileName string, newFileName string) (err error) {
	if _, err = os.Stat(newFileName); err == nil {
		err = fmt.Errorf("Renamed file %s already present", newFileName)
		return
	}

	if err = os.Rename(fileName, newFileName); err != nil {
		err = fmt.Errorf("Rename unsuccessful: %v", err)
		return
	}

	if _, err = os.Stat(fileName); err == nil {
		err = fmt.Errorf("Original file %s still exists", fileName)
		return
	}
	if _, err = os.Stat(newFileName); err != nil {
		err = fmt.Errorf("Renamed file %s not found", newFileName)
		return
	}
	return
}

func WriteFileInAppendMode(fileName string, content string) (err error) {
	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY|syscall.O_DIRECT, FilePermission_0600)
	if err != nil {
		err = fmt.Errorf("Open file for append: %v", err)
		return
	}

	// Closing file at the end.
	defer CloseFile(f)

	_, err = f.WriteString(content)

	return
}

func WriteFile(fileName string, content string) (err error) {
	f, err := os.OpenFile(fileName, os.O_RDWR|syscall.O_DIRECT, FilePermission_0600)
	if err != nil {
		err = fmt.Errorf("Open file for write at start: %v", err)
		return
	}

	// Closing file at the end.
	defer CloseFile(f)

	_, err = f.WriteAt([]byte(content), 0)

	return
}

func MoveFile(srcFilePath string, destDirPath string) (err error) {
	cmd := exec.Command("mv", srcFilePath, destDirPath)

	err = cmd.Run()
	if err != nil {
		err = fmt.Errorf("Moving file operation is failed: %v", err)
	}
	return err
}

func CloseFile(file *os.File) {
	if err := file.Close(); err != nil {
		log.Printf("error in closing: %v", err)
	}
}

func RemoveFile(filePath string) {
	err := os.Remove(filePath)
	if err != nil {
		log.Printf("Error in removing file:%v", err)
	}
}

func ReadFileSequentially(filePath string, chunkSize int64) (content []byte, err error) {
	chunk := make([]byte, chunkSize)
	var offset int64 = 0

	file, err := os.OpenFile(filePath, os.O_RDONLY|syscall.O_DIRECT, FilePermission_0600)
	if err != nil {
		log.Printf("Error in opening file:%v", err)
	}

	// Closing the file at the end.
	defer CloseFile(file)

	for err != io.EOF {
		var numberOfBytes int

		// Reading 200 MB chunk sequentially from the file.
		numberOfBytes, err = file.ReadAt(chunk, offset)
		// If the file reaches the end, write the remaining content in the buffer and return.
		if err == io.EOF {

			for i := offset; i < offset+int64(numberOfBytes); i++ {
				// Adding remaining bytes.
				content = append(content, chunk[i-offset])
			}
			err = nil
			return
		}
		if err != nil {
			return
		}
		// Write bytes in the buffer to compare with original content.
		content = append(content, chunk...)

		// The number of bytes read is not equal to 200MB.
		if int64(numberOfBytes) != chunkSize {
			log.Printf("Incorrect number of bytes read from file.")
		}

		// The offset will shift to read the next chunk.
		offset = offset + chunkSize
	}
	return
}

func WriteFileSequentially(filePath string, fileSize int64, chunkSize int64) (err error) {
	file, err := os.OpenFile(filePath, os.O_RDWR|syscall.O_DIRECT|os.O_CREATE, FilePermission_0600)
	if err != nil {
		log.Printf("Error in opening file:%v", err)
	}

	// Closing file at the end.
	defer CloseFile(file)

	var offset int64 = 0

	for offset < fileSize {
		// Get random chunkSize or remaining filesize data into chunk.
		if (fileSize - offset) < chunkSize {
			chunkSize = (fileSize - offset)
		}
		chunk := make([]byte, chunkSize)
		_, err = rand.Read(chunk)
		if err != nil {
			log.Fatalf("error while generating random string: %s", err)
		}

		var numberOfBytes int

		// Writes random chunkSize or remaining filesize data into file.
		numberOfBytes, err = file.Write(chunk)
		err = file.Sync()
		if err != nil {
			log.Printf("Error in syncing file:%v", err)
		}

		if err != nil {
			return
		}
		if int64(numberOfBytes) != chunkSize {
			log.Fatalf("Incorrect number of bytes written in the file.")
		}

		offset = offset + chunkSize
	}
	return
}
