// Copyright 2023 Google LLC
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
	"bufio"
	"bytes"
	"compress/gzip"
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

	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
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
	// TmpDirectory specifies the directory where temporary files will be created.
	// In this case, we are using the system's default temporary directory.
	TmpDirectory             = "/tmp"
	WaitDurationAfterFlushZB = time.Minute
	WaitDurationAfterCloseZB = time.Second
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
		err = fmt.Errorf("error in the opening the file %v", err)
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
		err = fmt.Errorf("renamed file %s already present", newFileName)
		return
	}

	if err = os.Rename(fileName, newFileName); err != nil {
		err = fmt.Errorf("rename unsuccessful: %v", err)
		return
	}

	if _, err = os.Stat(fileName); err == nil {
		err = fmt.Errorf("original file %s still exists", fileName)
		return
	}
	if _, err = os.Stat(newFileName); err != nil {
		err = fmt.Errorf("renamed file %s not found", newFileName)
		return
	}
	return
}

func WriteFileInAppendMode(fileName string, content string) (err error) {
	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY|syscall.O_DIRECT, FilePermission_0600)
	if err != nil {
		err = fmt.Errorf("open file for append: %v", err)
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
		err = fmt.Errorf("open file for write at start: %v", err)
		return
	}

	// Closing file at the end.
	defer CloseFile(f)

	_, err = f.WriteAt([]byte(content), 0)

	return
}

func CloseFiles(t *testing.T, files []*os.File) {
	t.Helper()
	for _, file := range files {
		err := file.Close()
		assert.NoError(t, err)
	}
}

// Deprecated: please use CloseFileShouldNotThrowError instead.
func CloseFile(file *os.File) {
	if err := file.Close(); err != nil {
		log.Fatalf("error in closing: %v", err)
	}
	WaitForSizeUpdate(setup.IsZonalBucketRun(), WaitDurationAfterCloseZB)
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
		return nil, fmt.Errorf("error in opening file %q: %w", filePath, err)
	}
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

func WriteChunkOfRandomBytesToFiles(files []*os.File, chunkSize int, offset int64) error {
	// Generate random data of chunk size.
	chunk, err := GenerateRandomData(int64(chunkSize))
	if err != nil {
		return fmt.Errorf("error in generating random data: %v", err)
	}

	for _, file := range files {
		// Write data in the file.
		n, err := file.WriteAt(chunk, offset)
		if err != nil {
			return fmt.Errorf("error in writing randomly in file: %s, %v", file.Name(), err)
		}

		if n != chunkSize {
			return fmt.Errorf("incorrect number of bytes written in the file %s actual %d, expected %d", file.Name(), n, chunkSize)
		}

		if !setup.IsZonalBucketRun() {
			err = file.Sync()
			if err != nil {
				return fmt.Errorf("error in syncing file: %v", err)
			}
			WaitForSizeUpdate(setup.IsZonalBucketRun(), WaitDurationAfterFlushZB)
		}
	}

	return nil
}

func WriteFilesSequentially(t *testing.T, filePaths []string, fileSize int64, chunkSize int64) {
	t.Helper()
	files := OpenFiles(t, filePaths)
	defer CloseFiles(t, files)

	var offset int64 = 0
	for offset < fileSize {
		// Reduce chunk size to remaining file size in case chunk size is larger.
		chunkSize = min(chunkSize, fileSize-offset)
		err := WriteChunkOfRandomBytesToFiles(files, int(chunkSize), offset)
		assert.NoError(t, err)
		offset = offset + chunkSize
	}
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

func ReadFileBetweenOffset(t *testing.T, file *os.File, startOffset, endOffset, chunkSize int64) string {
	t.Helper()
	chunk := make([]byte, chunkSize)
	var readData []byte

	for startOffset < endOffset {
		readSize := min(chunkSize, endOffset-startOffset)

		n, err := file.ReadAt(chunk[:readSize], startOffset)
		if err == io.EOF {
			readData = append(readData, chunk[:n]...)
			break
		} else if err != nil {
			t.Errorf("Failed to read file chunk at offset %d: %v", startOffset, err)
			return ""
		}
		readData = append(readData, chunk[:n]...)
		startOffset += int64(n)
	}

	return string(readData)
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

func OpenFileAsReadonly(filepath string) (*os.File, error) {
	f, err := os.OpenFile(filepath, os.O_RDONLY|syscall.O_DIRECT, FilePermission_0400)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s as readonly: %v", filepath, err)
	}

	return f, nil
}

// Open file in mode opens the given file in the flag mode provided.
func OpenFileInMode(t *testing.T, filepath string, flag int) *os.File {
	t.Helper()
	fh, err := os.OpenFile(filepath, flag, FilePermission_0600)
	require.NoError(t, err)
	return fh
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

	f1, err := OpenFileAsReadonly(filepath1)
	if err != nil {
		return false, err
	}

	defer CloseFile(f1)

	f2, err := OpenFileAsReadonly(filepath2)
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

func CreateFile(filePath string, filePerms os.FileMode, t testing.TB) (f *os.File) {
	// Creating a file shouldn't create file on GCS.
	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC|syscall.O_DIRECT, filePerms)
	if err != nil {
		t.Fatalf("CreateFile(%s): %v", filePath, err)
	}
	return
}

func OpenFiles(t *testing.T, filePaths []string) []*os.File {
	t.Helper()
	var files []*os.File

	// Open all files.
	for _, filePath := range filePaths {
		file, err := os.OpenFile(filePath, os.O_RDWR|syscall.O_DIRECT|os.O_CREATE, FilePermission_0600)
		require.NoError(t, err)
		files = append(files, file)
	}
	return files
}

func OpenFile(filePath string, t *testing.T) (f *os.File) {
	f, err := os.OpenFile(filePath, os.O_RDWR, FilePermission_0777)
	if err != nil {
		t.Fatalf("OpenFile(%s): %v", filePath, err)
	}
	return
}

func OpenFileWithODirect(t *testing.T, filePath string) (f *os.File) {
	t.Helper()
	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC|syscall.O_DIRECT, FilePermission_0600)
	if err != nil {
		require.NoError(t, err)
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

func WriteAt(content string, offset int64, fh *os.File, t testing.TB) {
	_, err := fh.WriteAt([]byte(content), offset)
	if err != nil {
		t.Fatalf("%s.WriteAt(%s, %d): %v", fh.Name(), content, offset, err)
	}
}

func CloseFileShouldNotThrowError(t testing.TB, file *os.File) {
	err := file.Close()
	assert.NoError(t, err)
	WaitForSizeUpdate(setup.IsZonalBucketRun(), WaitDurationAfterCloseZB)
}

func CloseFileShouldThrowError(t *testing.T, file *os.File) {
	t.Helper()
	if err := file.Close(); err == nil {
		t.Fatalf("file.Close() for file %s should throw an error: %v", file.Name(), err)
	}
}

func SyncFile(fh *os.File, t *testing.T) {
	err := fh.Sync()

	// Verify fh.Sync operation succeeds.
	if err != nil {
		t.Fatalf("%s.Sync(): %v", fh.Name(), err)
	}
	WaitForSizeUpdate(setup.IsZonalBucketRun(), WaitDurationAfterFlushZB)
}

func SyncFiles(files []*os.File, t *testing.T) {
	t.Helper()
	for _, file := range files {
		SyncFile(file, t)
	}
}

func SyncFileShouldThrowError(t *testing.T, file *os.File) {
	t.Helper()
	if err := file.Sync(); err == nil {
		t.Fatalf("file.Close() for file %s should throw an error: %v", file.Name(), err)
	}
}

func CreateFileWithContent(filePath string, filePerms os.FileMode, content string, t testing.TB) {
	fh := CreateFile(filePath, filePerms, t)
	WriteAt(content, 0, fh, t)
	CloseFileShouldNotThrowError(t, fh)
}

// CreateFileOfSize creates a file of given size with random data.
func CreateFileOfSize(fileSize int64, filePath string, t testing.TB) {
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

func writeGzipToFile(f *os.File, filepath, content string, contentSize int) (string, error) {
	w := gzip.NewWriter(f)
	if w == nil {
		return "", fmt.Errorf("failed to create gzip writer for file %s", filepath)
	}
	defer func() {
		if err := w.Close(); err != nil {
			log.Printf("failed to close gzip writer for file %s: %v", filepath, err)
		}
	}()

	n, err := w.Write([]byte(content))
	if err != nil {
		return "", fmt.Errorf("failed to write content to %s using gzip-writer: %w", filepath, err)
	}
	if n != contentSize {
		return "", fmt.Errorf("failed to write to gzip file %s. Content-size: %d bytes, wrote = %d bytes", filepath, contentSize, n)
	}

	return filepath, nil
}

func writeTextToFile(f *os.File, filepath, content string, contentSize int) (string, error) {
	n, err := f.WriteString(content)
	if err != nil {
		return "", err
	}
	if n != contentSize {
		return "", fmt.Errorf("failed to write to text file %s. Content-size: %d bytes, wrote = %d bytes", filepath, contentSize, n)
	}

	return filepath, nil
}

// Creates a temporary file (name-collision-safe) in /tmp with given content.
// If gzipCompress is true, output file is a gzip-compressed file.
// Caller is responsible for deleting the created file when done using it.
// Failure cases:
// 1. os.CreateTemp() returned error or nil handle
// 2. gzip.NewWriter() returned nil handle
// 3. Failed to write the content to the created temp file
func CreateLocalTempFile(content string, gzipCompress bool) (string, error) {
	// create appropriate name template for temp file
	filenameTemplate := "testfile-*.txt"
	if gzipCompress {
		filenameTemplate += ".gz"
	}

	f, err := os.CreateTemp(TmpDirectory, filenameTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to create tempfile for template %s: %w", filenameTemplate, err)
	}
	if f == nil {
		return "", fmt.Errorf("nil file handle returned from os.CreateTemp")
	}
	defer CloseFile(f)
	if gzipCompress {
		return writeGzipToFile(f, f.Name(), content, len(content))
	}

	return writeTextToFile(f, f.Name(), content, len(content))
}

// ReadAndCompare reads content from the given file paths and compares them.
func ReadAndCompare(t *testing.T, filePathInMntDir string, filePathInLocalDisk string, offset int64, chunkSize int64) {
	t.Helper()
	mountContents, err := ReadChunkFromFile(filePathInMntDir, chunkSize, offset, os.O_RDONLY|syscall.O_DIRECT)
	if err != nil {
		t.Fatalf("error in read file from mounted directory :%d", err)
	}

	diskContents, err := ReadChunkFromFile(filePathInLocalDisk, chunkSize, offset, os.O_RDONLY)
	if err != nil {
		t.Fatalf("error in read file from local directory :%d", err)
	}

	if !bytes.Equal(mountContents, diskContents) {
		t.Fatalf("data mismatch between mounted directory and local disk")
	}
}

func CreateLocalFile(ctx context.Context, t *testing.T, mntDir string, bucket gcs.Bucket, fileName string) (filePath string, f *os.File) {
	t.Helper()
	// Creating a file shouldn't create file on GCS.
	filePath = path.Join(mntDir, fileName)

	f, err := os.Create(filePath)

	assert.Equal(t, nil, err)
	ValidateObjectNotFoundErr(ctx, t, bucket, fileName)
	return
}

func CloseLocalFile(t *testing.T, f **os.File) error {
	t.Helper()
	err := (*f).Close()
	*f = nil
	return err
}

func CheckLogFileForMessage(t *testing.T, expectedLog, logFile string) bool {
	file, err := os.Open(logFile)
	require.NoError(t, err, "Failed to open log file")
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), expectedLog) {
			return true
		}
	}
	return false
}

// ValidateSyncGivenThatFileIsClobbered method validates sync operation on file which has already been clobbered.
// 1. With streaming writes sync operation only uploads pending buffers and it doesn't return any error.
// 2. Without streaming writes file is synced with GCS and returns ESTALE error.
func ValidateSyncGivenThatFileIsClobbered(t *testing.T, file *os.File, streamingWrites bool) {
	t.Helper()
	err := file.Sync()
	if streamingWrites {
		assert.NoError(t, err)
	} else {
		ValidateESTALEError(t, err)
	}
}

// CreateFileAndCopyToMntDir creates a file of given size.
// The same file will be copied to the mounted directory as well.
func CreateFileAndCopyToMntDir(t *testing.T, fileSize int, dirName string) (string, string) {
	testDir := setup.SetupTestDirectory(dirName)
	fileInLocalDisk := "test_file" + setup.GenerateRandomString(5) + ".txt"
	filePathInLocalDisk := path.Join(os.TempDir(), fileInLocalDisk)
	filePathInMntDir := path.Join(testDir, fileInLocalDisk)
	CreateFileOnDiskAndCopyToMntDir(t, filePathInLocalDisk, filePathInMntDir, fileSize)
	return filePathInLocalDisk, filePathInMntDir
}

// CreateFileOnDiskAndCopyToMntDir creates a file of given size and copies to given path.
func CreateFileOnDiskAndCopyToMntDir(t *testing.T, filePathInLocalDisk string, filePathInMntDir string, fileSize int) {
	setup.RunScriptForTestData("../util/setup/testdata/write_content_of_fix_size_in_file.sh", filePathInLocalDisk, strconv.Itoa(fileSize))
	err := CopyFile(filePathInLocalDisk, filePathInMntDir)
	if err != nil {
		t.Errorf("Error in copying file:%v", err)
	}
}
