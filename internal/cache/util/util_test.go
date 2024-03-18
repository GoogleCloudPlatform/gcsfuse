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

package util

import (
	"errors"
	"fmt"
	"os"
	"path"
	"reflect"
	"strings"
	"syscall"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/data"
	testutil "github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	. "github.com/jacobsa/ogletest"
)

func TestUtil(t *testing.T) { RunTests(t) }

type utilTest struct {
	fileSpec data.FileSpec
	flag     int
	uid      int
	gid      int
}

const FileDir = "/some/dir/"
const FileName = "foo.txt"

func init() { RegisterTestSuite(&utilTest{}) }

func (ut *utilTest) SetUp(*TestInfo) {
	operations.RemoveDir(FileDir)
	ut.flag = os.O_RDWR
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Errorf("error while finding home directory: %w", err))
	}
	ut.fileSpec = data.FileSpec{
		Path:     path.Join(homeDir, FileDir, FileName),
		FilePerm: DefaultFilePerm,
		DirPerm:  DefaultDirPerm,
	}
	ut.uid = os.Getuid()
	ut.gid = os.Getgid()
}

func (ut *utilTest) TearDown() {
	operations.RemoveDir(path.Dir(ut.fileSpec.Path))
}

func (ut *utilTest) assertFileAndDirCreationWithGivenDirPerm(file *os.File, err error, dirPerm os.FileMode) {
	ExpectEq(nil, err)

	dirStat, dirErr := os.Stat(path.Dir(file.Name()))
	ExpectEq(false, os.IsNotExist(dirErr))
	ExpectEq(path.Dir(ut.fileSpec.Path), path.Dir(file.Name()))
	ExpectEq(dirPerm, dirStat.Mode().Perm())
	ExpectEq(ut.uid, dirStat.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(ut.gid, dirStat.Sys().(*syscall.Stat_t).Gid)

	fileStat, fileErr := os.Stat(file.Name())
	ExpectEq(false, os.IsNotExist(fileErr))
	ExpectEq(ut.fileSpec.Path, file.Name())
	ExpectEq(ut.fileSpec.FilePerm, fileStat.Mode())
	ExpectEq(ut.uid, fileStat.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(ut.gid, fileStat.Sys().(*syscall.Stat_t).Gid)
}

func (ut *utilTest) Test_CreateFile_FileDirNotPresent() {

	file, err := CreateFile(ut.fileSpec, ut.flag)

	ut.assertFileAndDirCreationWithGivenDirPerm(file, err, 0700)
	ExpectEq(nil, file.Close())
}

func (ut *utilTest) Test_CreateFile_ShouldThrowErrorIfFileDirNotPresentAndProvidedPermissionsAreInsufficient() {
	ut.fileSpec.DirPerm = 644

	_, err := CreateFile(ut.fileSpec, ut.flag)

	ExpectNe(nil, err)
	ExpectEq("error in stating file "+ut.fileSpec.Path+": stat "+ut.fileSpec.Path+": permission denied", err.Error())
}

func (ut *utilTest) Test_CreateFile_FileDirPresent() {
	err := os.MkdirAll(path.Dir(ut.fileSpec.Path), 0755)

	ExpectEq(nil, err)

	file, err := CreateFile(ut.fileSpec, ut.flag)

	ut.assertFileAndDirCreationWithGivenDirPerm(file, err, 0755)
	ExpectEq(nil, file.Close())
}

func (ut *utilTest) Test_CreateFile_ReadOnlyFile() {
	ut.flag = os.O_RDONLY

	file, err := CreateFile(ut.fileSpec, ut.flag)

	ut.assertFileAndDirCreationWithGivenDirPerm(file, err, 0700)
	content := "foo"
	_, err = file.Write([]byte(content))
	ExpectNe(nil, err)
	ExpectTrue(strings.Contains(err.Error(), "bad file descriptor"))
	ExpectEq(nil, file.Close())
	fileContent, err := os.ReadFile(ut.fileSpec.Path)
	ExpectEq(nil, err)
	ExpectEq("", string(fileContent))
}

func (ut *utilTest) Test_CreateFile_ReadWriteFile() {
	ut.flag = os.O_RDWR

	file, err := CreateFile(ut.fileSpec, ut.flag)

	ut.assertFileAndDirCreationWithGivenDirPerm(file, err, 0700)
	content := "foo"
	n, err := file.Write([]byte(content))
	ExpectEq(nil, err)
	ExpectEq(3, n)
	ExpectEq(nil, file.Close())
	fileContent, err := os.ReadFile(ut.fileSpec.Path)
	ExpectEq(nil, err)
	ExpectEq(content, string(fileContent))
}

func (ut *utilTest) Test_CreateFile_FilePerm0755() {
	ut.fileSpec.FilePerm = os.FileMode(0755)

	file, err := CreateFile(ut.fileSpec, ut.flag)

	ut.assertFileAndDirCreationWithGivenDirPerm(file, err, 0700)
	ExpectEq(nil, file.Close())
}

func (ut *utilTest) Test_CreateFile_FilePerm0544() {
	ut.fileSpec.FilePerm = os.FileMode(0544)

	file, err := CreateFile(ut.fileSpec, ut.flag)

	ut.assertFileAndDirCreationWithGivenDirPerm(file, err, 0700)
	ExpectEq(nil, file.Close())
}

func (ut *utilTest) Test_CreateFile_FilePresent() {
	err := os.MkdirAll(path.Dir(ut.fileSpec.Path), 0755)
	ExpectEq(nil, err)
	file, err := os.OpenFile(ut.fileSpec.Path, os.O_CREATE|os.O_RDWR, DefaultFilePerm)
	ExpectEq(nil, err)
	ExpectEq(nil, file.Close())

	file, err = CreateFile(ut.fileSpec, ut.flag)

	ut.assertFileAndDirCreationWithGivenDirPerm(file, err, 0755)
	ExpectEq(nil, file.Close())
}

func (ut *utilTest) Test_CreateFile_FilePresentWithLessAccess() {
	err := os.MkdirAll(path.Dir(ut.fileSpec.Path), 0755)
	ExpectEq(nil, err)
	file, err := os.OpenFile(ut.fileSpec.Path, os.O_CREATE, os.FileMode(0544))
	ExpectEq(nil, err)
	ExpectEq(nil, file.Close())

	_, err = CreateFile(ut.fileSpec, ut.flag)

	ExpectNe(nil, err)
	ExpectTrue(strings.Contains(strings.ToLower(err.Error()), "permission denied"))
}

func (ut *utilTest) Test_CreateFile_RelativePath() {
	ut.fileSpec.Path = "./some/path/foo.txt"

	file, err := CreateFile(ut.fileSpec, ut.flag)

	ut.assertFileAndDirCreationWithGivenDirPerm(file, err, 0700)
	ExpectEq(nil, file.Close())
}

func (ut *utilTest) Test_getObjectPath() {
	inputs := [][]string{{"", ""}, {"a", "b"}, {"a/b/", "/c/d"}, {"", "a"}, {"a", ""}}
	expectedOutPuts := [5]string{"", "a/b", "a/b/c/d", "a", "a"}

	results := [5]string{}
	for i := 0; i < 5; i++ {
		results[i] = GetDownloadPath(inputs[i][0], inputs[i][1])
	}

	ExpectTrue(reflect.DeepEqual(expectedOutPuts, results))
}

func (ut *utilTest) Test_getDownloadPath() {
	inputs := []string{"/", "a/b", "a/b/c/d", "/a", "a/"}
	cacheDir := "/test/dir"
	expectedOutputs := [5]string{cacheDir, cacheDir + "/a/b",
		cacheDir + "/a/b/c/d", cacheDir + "/a", cacheDir + "/a"}

	results := [5]string{}
	for i := 0; i < 5; i++ {
		results[i] = GetDownloadPath(cacheDir, inputs[i])
	}

	ExpectTrue(reflect.DeepEqual(expectedOutputs, results))
}

func (ut *utilTest) Test_IsCacheHandleValid_True() {
	errMessages := []string{
		InvalidFileHandleErrMsg + "test",
		InvalidFileDownloadJobErrMsg + "test",
		InvalidFileInfoCacheErrMsg + "test",
		ErrInSeekingFileHandleMsg + "test",
		ErrInReadingFileHandleMsg + "test",
	}

	for _, errMsg := range errMessages {
		ExpectTrue(IsCacheHandleInvalid(errors.New(errMsg)))
	}
}

func (ut *utilTest) Test_IsCacheHandleValid_False() {
	errMessages := []string{
		FallbackToGCSErrMsg + "test",
		"random error message",
	}

	for _, errMsg := range errMessages {
		ExpectFalse(IsCacheHandleInvalid(errors.New(errMsg)))
	}
}

func Test_CreateCacheDirectoryIfNotPresentAt_ShouldNotReturnAnyErrorWhenDirectoryExists(t *testing.T) {
	base := path.Join("./", string(testutil.GenerateRandomBytes(4)))
	dirPath := path.Join(base, "/", "path/cachedir")
	dirCreationErr := os.MkdirAll(dirPath, 0700)
	defer os.RemoveAll(base)
	AssertEq(nil, dirCreationErr)

	err := CreateCacheDirectoryIfNotPresentAt(dirPath, 0000)

	AssertEq(nil, err)
	fileInfo, err := os.Stat(dirPath)
	AssertEq(nil, err)
	AssertEq(0700, fileInfo.Mode().Perm())
}

func Test_CreateCacheDirectoryIfNotPresentAt_ShouldNotReturnAnyErrorWhenDirectoryCanBeCreatedWithOwnerPermissions(t *testing.T) {
	base := path.Join("./", string(testutil.GenerateRandomBytes(4)))
	dirPath := path.Join(base, "/", "path/cachedir")
	defer os.RemoveAll(base)

	err := CreateCacheDirectoryIfNotPresentAt(dirPath, 0700)

	AssertEq(nil, err)
	fileInfo, err := os.Stat(dirPath)
	AssertEq(nil, err)
	AssertEq(0700, fileInfo.Mode().Perm())
}

func Test_CreateCacheDirectoryIfNotPresentAt_ShouldNotReturnAnyErrorWhenDirectoryCanBeCreatedWithOthersPermissions(t *testing.T) {
	base := path.Join("./", string(testutil.GenerateRandomBytes(4)))
	dirPath := path.Join(base, "/", "path/cachedir")
	defer os.RemoveAll(base)

	err := CreateCacheDirectoryIfNotPresentAt(dirPath, 0755)

	AssertEq(nil, err)
	fileInfo, err := os.Stat(dirPath)
	AssertEq(nil, err)
	AssertEq(0755, fileInfo.Mode().Perm())
}

func Test_CreateCacheDirectoryIfNotPresentAt_ShouldReturnErrorWhenDirectoryDoesNotHavePermissions(t *testing.T) {
	dirPath := path.Join("./", string(testutil.GenerateRandomBytes(4)))
	dirCreationErr := os.MkdirAll(dirPath, 0444)
	defer os.RemoveAll(dirPath)
	AssertEq(nil, dirCreationErr)

	err := CreateCacheDirectoryIfNotPresentAt(dirPath, 0755)

	AssertNe(nil, err)
	AssertTrue(strings.Contains(err.Error(), "error creating file at directory ("+dirPath+")"))
}
