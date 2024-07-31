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
	"hash/crc32"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

const (
	OneKiB = 1024
	OneMiB = OneKiB * OneKiB
	// ChunkSizeForContentComparison is currently set to 1 MiB.
	ChunkSizeForContentComparison int = OneMiB

	// TimeSlop The radius we use for "expect mtime is within"-style assertions as kernel
	// time can be slightly out of sync of time.Now().
	// Ref: https://github.com/golang/go/issues/33510
	TimeSlop = 25 * time.Millisecond
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
		destination, err = os.OpenFile(dstFileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, FilePermission_0600)
	} else {
		destination, err = os.OpenFile(dstFileName, os.O_WRONLY|os.O_CREATE, FilePermission_0600)
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

func ReadChunkFromFile(filePath string, chunkSize int64, offset int64, flag int) (chunk []byte, err error) {
	chunk = make([]byte, chunkSize)

	file, err := os.OpenFile(filePath, flag, FilePermission_0600)
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

func openFileAsReadonly(filepath string) (*os.File, error) {
	f, err := os.OpenFile(filepath, os.O_RDONLY|syscall.O_DIRECT, FilePermission_0400)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s as readonly: %v", filepath, err)
	}

	return f, nil
}

func readBytesFromFile(f *os.File, numBytesToRead int, b []byte) error {
	numBytesRead, err := f.Read(b)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %v", f.Name(), err)
	}
	if numBytesRead != numBytesToRead {
		return fmt.Errorf("failed to read file %s, expected read bytes = %d, actual read bytes = %d", f.Name(), numBytesToRead, numBytesRead)
	}

	return nil
}

// Finds if two local files have identical content (equivalnt to binary diff).
// Needs (a) both files to exist, (b)read permission on both the files, (c) both
// inputs to be proper files, i.e. directories not supported.
// Compares file names first. If different, compares sizes next.
// If sizes match, then compares the contents of both the files.
// Returns true if no error and files match.
// Returns false if files don't match (captures reason for mismatch in err) or if any other error.
func AreFilesIdentical(filepath1, filepath2 string) (bool, error) {
	if filepath1 == "" || filepath2 == "" {
		return false, fmt.Errorf("one or both files being diff'ed have empty path")
	} else if filepath1 == filepath2 {
		return true, nil
	}

	fstat1, err := StatFile(filepath1)
	if err != nil {
		return false, err
	}

	fstat2, err := StatFile(filepath2)
	if err != nil {
		return false, err
	}

	file1size := (*fstat1).Size()
	file2size := (*fstat2).Size()
	if file1size != file2size {
		return false, fmt.Errorf("files don't match in size: %s (%d bytes), %s (%d bytes)", filepath1, file1size, filepath2, file2size)
	}

	if file1size == 0 {
		return true, nil
	}

	f1, err := openFileAsReadonly(filepath1)
	if err != nil {
		return false, err
	}

	defer CloseFile(f1)

	f2, err := openFileAsReadonly(filepath2)
	if err != nil {
		return false, err
	}

	defer CloseFile(f2)

	sizeRemaining := int(file1size)
	b1 := make([]byte, ChunkSizeForContentComparison)
	b2 := make([]byte, ChunkSizeForContentComparison)
	numBytesBeingRead := ChunkSizeForContentComparison

	for sizeRemaining > 0 {
		if sizeRemaining < ChunkSizeForContentComparison {
			numBytesBeingRead = sizeRemaining
		}

		err := readBytesFromFile(f1, numBytesBeingRead, b1)
		if err != nil {
			return false, err
		}

		err = readBytesFromFile(f2, numBytesBeingRead, b2)
		if err != nil {
			return false, err
		}

		if !bytes.Equal(b1[:numBytesBeingRead], b2[:numBytesBeingRead]) {
			return false, fmt.Errorf("files don't match in content: %s, %s", filepath1, filepath2)
		}

		sizeRemaining -= numBytesBeingRead
	}

	return true, nil
}

// Returns size of a give GCS object with path (without 'gs://').
// Fails if the object doesn't exist or permission to read object's metadata is not
// available.
func GetGcsObjectSize(gcsObjPath string) (int, error) {
	stdout, err := ExecuteGcloudCommandf("storage du -s gs://%s", gcsObjPath)
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
func DownloadGcsObject(gcsObjPath, localPath string) error {
	_, err := ExecuteGcloudCommandf("storage cp gs://%s %s", gcsObjPath, localPath)
	if err != nil {
		return err
	}

	return nil
}

// Uploads given local file to GCS object (with path without 'gs://').
// Fails if the file doesn't exist or permission to write to object/bucket is not
// available.
func UploadGcsObject(localPath, gcsObjPath string, uploadGzipEncoded bool) error {
	var err error
	if uploadGzipEncoded {
		_, err = ExecuteGcloudCommandf("storage cp -Z %s gs://%s", localPath, gcsObjPath)
	} else {
		_, err = ExecuteGcloudCommandf("storage cp %s gs://%s", localPath, gcsObjPath)
	}

	return err
}

// Deletes a given GCS object (with path without 'gs://').
// Fails if the object doesn't exist or permission to delete object is not
// available.
func DeleteGcsObject(gcsObjPath string) error {
	_, err := ExecuteGcloudCommandf("rm gs://%s", gcsObjPath)
	return err
}

// Clears cache-control attributes on given GCS object (with path without 'gs://').
// Fails if the file doesn't exist or permission to modify object's metadata is not
// available.
func ClearCacheControlOnGcsObject(gcsObjPath string) error {
	_, err := ExecuteGcloudCommandf("storage objects update --cache-control='' gs://%s", gcsObjPath)
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

func CreateSymLink(filePath, symlink string, t *testing.T) {
	err := os.Symlink(filePath, symlink)

	// Verify os.Symlink operation succeeds.
	if err != nil {
		t.Fatalf("os.Symlink(%s, %s): %v", filePath, symlink, err)
	}
}

func VerifyStatFile(filePath string, fileSize int64, filePerms os.FileMode, t *testing.T) {
	fi, err := os.Stat(filePath)

	if err != nil {
		t.Fatalf("os.Stat err: %v", err)
	}

	if fi.Name() != path.Base(filePath) {
		t.Fatalf("File name mismatch in stat call. Expected: %s, Got: %s", path.Base(filePath), fi.Name())
	}

	if fi.Size() != fileSize {
		t.Fatalf("File size mismatch in stat call. Expected: %d, Got: %d", fileSize, fi.Size())
	}

	if fi.Mode() != filePerms {
		t.Fatalf("File permissions mismatch in stat call. Expected: %v, Got: %v", filePerms, fi.Mode())
	}
}

func VerifyReadFile(filePath, expectedContent string, t *testing.T) {
	gotContent, err := os.ReadFile(filePath)

	// Verify os.ReadFile operation succeeds.
	if err != nil {
		t.Fatalf("os.ReadFile(%s): %v", filePath, err)
	}
	if expectedContent != string(gotContent) {
		t.Fatalf("Content mismatch. Expected: %s, Got: %s", expectedContent, gotContent)
	}
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

func VerifyReadLink(expectedTarget, symlinkName string, t *testing.T) {
	gotTarget, err := os.Readlink(symlinkName)

	// Verify os.Readlink operation succeeds.
	if err != nil {
		t.Fatalf("os.Readlink(%s): %v", symlinkName, err)
	}
	if expectedTarget != gotTarget {
		t.Fatalf("Symlink target mismatch. Expected: %s, Got: %s", expectedTarget, gotTarget)
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

func SyncFile(fh *os.File, t *testing.T) {
	err := fh.Sync()

	// Verify fh.Sync operation succeeds.
	if err != nil {
		t.Fatalf("%s.Sync(): %v", fh.Name(), err)
	}
}

func CreateFileWithContent(filePath string, filePerms os.FileMode,
	content string, t *testing.T) {
	fh := CreateFile(filePath, filePerms, t)
	WriteAt(content, 0, fh, t)
	CloseFile(fh)
}

// CreateFileOfSize creates a file of given size with random data.
func CreateFileOfSize(fileSize int64, filePath string, t *testing.T) {
	randomData, err := GenerateRandomData(fileSize)
	if err != nil {
		t.Errorf("operations.GenerateRandomData: %v", err)
	}
	CreateFileWithContent(filePath, FilePermission_0600, string(randomData), t)
}

// CalculateCRC32 calculates and returns the CRC-32 checksum of the data from the provided Reader.
func CalculateCRC32(src io.Reader) (uint32, error) {
	crc32Table := crc32.MakeTable(crc32.Castagnoli) // Pre-calculate the table
	hasher := crc32.New(crc32Table)

	if _, err := io.Copy(hasher, src); err != nil {
		return 0, fmt.Errorf("error calculating CRC-32: %w", err) // Wrap error
	}

	return hasher.Sum32(), nil // Return checksum and nil error on success
}

// CalculateFileCRC32 calculates and returns the CRC-32 checksum of a file.
func CalculateFileCRC32(filePath string) (uint32, error) {
	// Open file with simplified flags and permissions
	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close() // Ensure file closure

	return CalculateCRC32(file)
}

// SizeOfFile returns the size of the given file by path.
// by invoking a stat call on it.
func SizeOfFile(filepath string) (size int64, err error) {
	fstat, err := StatFile(filepath)
	if err != nil {
		return 0, err
	}

	return (*fstat).Size(), nil
}
