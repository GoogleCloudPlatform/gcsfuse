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

// Provide helper functions related to file.
package operations

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
)

func copyFile(srcFileName, dstFileName string, allowOverwrite bool) (err error) {
	if !allowOverwrite {
		if _, err = os.Stat(dstFileName); err == nil {
			err = fmt.Errorf("destination file %s already present", dstFileName)
			return
		}
	}

	source, err := os.OpenFile(srcFileName, syscall.O_DIRECT, FilePermission_0600)
	if err != nil {
		err = fmt.Errorf("file %s opening error: %v", srcFileName, err)
		return
	}

	// Closing file at the end.
	defer CloseFile(source)

	var destination *os.File
	if allowOverwrite {
		destination, err = os.OpenFile(dstFileName, os.O_WRONLY|os.O_CREATE|syscall.O_DIRECT|os.O_TRUNC, FilePermission_0600)
	} else {
		destination, err = os.OpenFile(dstFileName, os.O_WRONLY|os.O_CREATE|syscall.O_DIRECT, FilePermission_0600)
	}

	if err != nil {
		err = fmt.Errorf("copied file creation error: %v", err)
		return
	}

	// Closing file at the end.
	defer CloseFile(destination)

	// File copying with io.Copy() utility.
	_, err = io.Copy(destination, source)
	if err != nil {
		err = fmt.Errorf("error in file copying: %v", err)
		return
	}
	return
}

func CopyFile(srcFileName, newFileName string) (err error) {
	return copyFile(srcFileName, newFileName, false)
}

func CopyFileAllowOverwrite(srcFileName, newFileName string) (err error) {
	return copyFile(srcFileName, newFileName, true)
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
		log.Fatalf("error in closing: %v", err)
	}
}

func RemoveFile(filePath string) {
	err := os.Remove(filePath)
	if err != nil {
		log.Printf("os.Remove(%s): %v", filePath, err)
	}
}

func ReadFileSequentially(filePath string, chunkSize int64) (content []byte, err error) {
	chunk := make([]byte, chunkSize)
	var offset int64 = 0

	file, err := os.OpenFile(filePath, os.O_RDONLY|syscall.O_DIRECT, FilePermission_0600)
	if err != nil {
		log.Printf("Error in opening file: %v", err)
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

// Write data of chunkSize in file at given offset.
func WriteChunkOfRandomBytesToFile(file *os.File, chunkSize int, offset int64) error {
	// Generate random data of chunk size.
	chunk := make([]byte, chunkSize)
	_, err := rand.Read(chunk)
	if err != nil {
		return fmt.Errorf("error while generating random string: %v", err)
	}

	// Write data in the file.
	n, err := file.WriteAt(chunk, offset)
	if err != nil {
		return fmt.Errorf("Error in writing randomly in file: %v", err)
	}

	if n != chunkSize {
		return fmt.Errorf("Incorrect number of bytes written in the file actual %d, expected %d", n, chunkSize)
	}

	err = file.Sync()
	if err != nil {
		return fmt.Errorf("Error in syncing file: %v", err)
	}

	return nil
}

func WriteFileSequentially(filePath string, fileSize int64, chunkSize int64) (err error) {
	file, err := os.OpenFile(filePath, os.O_RDWR|syscall.O_DIRECT|os.O_CREATE, FilePermission_0600)
	if err != nil {
		log.Fatalf("Error in opening file: %v", err)
	}

	// Closing file at the end.
	defer CloseFile(file)

	var offset int64 = 0

	for offset < fileSize {
		// Get random chunkSize or remaining filesize data into chunk.
		if (fileSize - offset) < chunkSize {
			chunkSize = (fileSize - offset)
		}

		err := WriteChunkOfRandomBytesToFile(file, int(chunkSize), offset)
		if err != nil {
			log.Fatalf("Error in writing chunk: %v", err)
		}

		offset = offset + chunkSize
	}
	return
}

func ReadChunkFromFile(filePath string, chunkSize int64, offset int64) (chunk []byte, err error) {
	chunk = make([]byte, chunkSize)

	file, err := os.OpenFile(filePath, os.O_RDONLY, FilePermission_0600)
	if err != nil {
		log.Printf("Error in opening file: %v", err)
		return
	}

	f, err := os.Stat(filePath)
	if err != nil {
		log.Printf("Error in stating file: %v", err)
		return
	}

	// Closing the file at the end.
	defer CloseFile(file)

	var numberOfBytes int

	// Reading chunk size randomly from the file.
	numberOfBytes, err = file.ReadAt(chunk, offset)
	if err == io.EOF {
		err = nil
	}
	if err != nil {
		return
	}

	// The number of bytes read is not equal to 200MB.
	if int64(numberOfBytes) != chunkSize && int64(numberOfBytes) != f.Size()-offset {
		log.Printf("Incorrect number of bytes read from file.")
	}

	return
}

// Returns the stats of a file.
// Fails if the passed input is a directory.
func StatFile(file string) (*fs.FileInfo, error) {
	fstat, err := os.Stat(file)
	if err != nil {
		return nil, fmt.Errorf("failed to stat input file %s: %v", file, err)
	} else if fstat.IsDir() {
		return nil, fmt.Errorf("input file %s is a directory", file)
	}

	return &fstat, nil
}

// Finds if two local files have identical content (equivalnt to binary diff).
// Needs (a) both files to exist, (b)read permission on both the files, (c) both
// inputs to be proper files, i.e. directories not supported.
// Compares file names first. If different, compares sizes next.
// If sizes match, then compares the contents of both the files.
// Not a good idea for very large files as it loads both the files' contents in
// the memory completely.
// Returns 0 if no error and files match.
// Returns 1 if files don't match and captures reason for mismatch in err.
// Returns 2 if any error.
func DiffFiles(filepath1, filepath2 string) (int, error) {
	if filepath1 == "" || filepath2 == "" {
		return 2, fmt.Errorf("one or both files being diff'ed have empty path")
	} else if filepath1 == filepath2 {
		return 0, nil
	}

	fstat1, err := StatFile(filepath1)
	if err != nil {
		return 2, err
	}

	fstat2, err := StatFile(filepath2)
	if err != nil {
		return 2, err
	}

	file1size := (*fstat1).Size()
	file2size := (*fstat2).Size()
	if file1size != file2size {
		return 1, fmt.Errorf("files don't match in size: %s (%d bytes), %s (%d bytes)", filepath1, file1size, filepath2, file2size)
	}

	bytes1, err := ReadFile(filepath1)
	if err != nil || bytes1 == nil {
		return 2, fmt.Errorf("failed to read file %s", filepath1)
	} else if int64(len(bytes1)) != file1size {
		return 2, fmt.Errorf("failed to completely read file %s", filepath1)
	}

	bytes2, err := ReadFile(filepath2)
	if err != nil || bytes2 == nil {
		return 2, fmt.Errorf("failed to read file %s", filepath2)
	} else if int64(len(bytes2)) != file2size {
		return 2, fmt.Errorf("failed to completely read file %s", filepath2)
	}

	if !bytes.Equal(bytes1, bytes2) {
		return 1, fmt.Errorf("files don't match in content: %s, %s", filepath1, filepath2)
	}

	return 0, nil
}

// Returns size of a give GCS object with path (without 'gs://').
// Fails if the object doesn't exist or permission to read object's metadata is not
// available.
// Uses 'gsutil du -s gs://gcsObjPath'.
// Alternative 'gcloud storage du -s gs://gcsObjPath', but it doesn't work on kokoro VM.
func GetGcsObjectSize(gcsObjPath string) (int, error) {
	stdout, err := ExecuteGsutilCommandf("du -s gs://%s", gcsObjPath)
	if err != nil {
		return 0, err
	}

	// The above gcloud command returns output in the following format:
	// <size> <gcs-object-path>
	// So, we need to pick out only the first string before ' '.
	gcsObjectSize, err := strconv.Atoi(strings.TrimSpace(strings.Split(string(stdout), " ")[0]))
	if err != nil {
		return gcsObjectSize, err
	}

	return gcsObjectSize, nil
}

// Downloads given GCS object (with path without 'gs://') to localPath.
// Fails if the object doesn't exist or permission to read object is not
// available.
// Uses 'gsutil cp gs://gcsObjPath localPath'
// Alternative 'gcloud storage cp gs://gcsObjPath localPath' but it doesn't work on kokoro VM.
func DownloadGcsObject(gcsObjPath, localPath string) error {
	_, err := ExecuteGsutilCommandf("cp gs://%s %s", gcsObjPath, localPath)
	if err != nil {
		return err
	}

	return nil
}

// Uploads given local file to GCS object (with path without 'gs://').
// Fails if the file doesn't exist or permission to write to object/bucket is not
// available.
// Uses 'gsutil cp localPath gs://gcsObjPath'
// Alternative 'gcloud storage cp localPath gs://gcsObjPath' but it doesn't work on kokoro VM.
func UploadGcsObject(localPath, gcsObjPath string, uploadGzipEncoded bool) error {
	var err error
	if uploadGzipEncoded {
		// Using gsutil instead of `gcloud alpha` here as `gcloud alpha`
		// option `-Z` isn't supported on the kokoro VM.
		_, err = ExecuteGsutilCommandf("cp -Z %s gs://%s", localPath, gcsObjPath)
	} else {
		_, err = ExecuteGsutilCommandf("cp %s gs://%s", localPath, gcsObjPath)
	}

	return err
}

// Deletes a given GCS object (with path without 'gs://').
// Fails if the object doesn't exist or permission to delete object is not
// available.
// Uses 'gsutil rm gs://gcsObjPath'
// Alternative 'gcloud storage rm gs://gcsObjPath' but it doesn't work on kokoro VM.
func DeleteGcsObject(gcsObjPath string) error {
	_, err := ExecuteGsutilCommandf("rm gs://%s", gcsObjPath)
	return err
}

// Clears cache-control attributes on given GCS object (with path without 'gs://').
// Fails if the file doesn't exist or permission to modify object's metadata is not
// available.
// Uses 'gsutil setmeta -h "Cache-Control:" gs://<path>'
// Preferred approach is 'gcloud storage objects update gs://gs://gcsObjPath --cache-control=' ' ' but it doesn't work on kokoro VM.
func ClearCacheControlOnGcsObject(gcsObjPath string) error {
	// Using gsutil instead of `gcloud alpha` here as `gcloud alpha`
	// implementation for updating object metadata is missing on the kokoro VM.
	_, err := ExecuteGsutilCommandf("setmeta -h \"Cache-Control:\" gs://%s ", gcsObjPath)
	return err
}

func CreateFile(filePath string, filePerms os.FileMode, t *testing.T) (f *os.File) {
	// Creating a file shouldn't create file on GCS.
	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, filePerms)
	if err != nil {
		t.Fatalf("CreateFile(%s): %v", filePath, err)
	}
	return
}

func VerifyFileEntry(entry os.DirEntry, fileName string, size int64, t *testing.T) {
	if entry.IsDir() {
		t.Fatalf("Expected: file entry, Got: directory entry.")
	}
	if entry.Name() != fileName {
		t.Fatalf("File name, Expected: %s, Got: %s", fileName, entry.Name())
	}
	fileInfo, err := entry.Info()
	if err != nil {
		t.Fatalf("%s.Info() err: %v", fileName, err)
	}
	if fileInfo.Size() != size {
		t.Fatalf("Local file %s size, Expected: %d, Got: %d", fileName, size, fileInfo.Size())
	}
}

func WriteWithoutClose(fh *os.File, content string, t *testing.T) {
	_, err := fh.Write([]byte(content))
	if err != nil {
		t.Fatalf("Error while writing to local file. err: %v", err)
	}
}

func WriteAt(content string, offset int64, fh *os.File, t *testing.T) {
	_, err := fh.WriteAt([]byte(content), offset)
	if err != nil {
		t.Fatalf("%s.WriteAt(%s, %d): %v", fh.Name(), content, offset, err)
	}
}

func CloseFileShouldNotThrowError(file *os.File, t *testing.T) {
	if err := file.Close(); err != nil {
		t.Fatalf("file.Close() for file %s: %v", file.Name(), err)
	}
}
