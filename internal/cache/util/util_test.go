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
	"fmt"
	"os"
	"path"
	"strings"
	"syscall"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
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
		panic(fmt.Errorf("error while finding home directory: %v", err))
	}
	ut.fileSpec = data.FileSpec{
		Path: path.Join(homeDir, FileDir, FileName),
		Perm: DefaultFileMode,
	}
	ut.uid = os.Getuid()
	ut.gid = os.Getgid()
}

func (ut *utilTest) TearDown() {
	operations.RemoveDir(path.Dir(ut.fileSpec.Path))
}

func (ut *utilTest) assertFileAndDirCreation(file *os.File, err error) {
	ExpectEq(nil, err)

	dirStat, dirErr := os.Stat(path.Dir(file.Name()))
	ExpectEq(false, os.IsNotExist(dirErr))
	ExpectEq(path.Dir(ut.fileSpec.Path), path.Dir(file.Name()))
	ExpectEq(FileDirPerm|os.ModeDir, dirStat.Mode())
	ExpectEq(ut.uid, dirStat.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(ut.gid, dirStat.Sys().(*syscall.Stat_t).Gid)

	fileStat, fileErr := os.Stat(file.Name())
	ExpectEq(false, os.IsNotExist(fileErr))
	ExpectEq(ut.fileSpec.Path, file.Name())
	ExpectEq(ut.fileSpec.Perm, fileStat.Mode())
	ExpectEq(ut.uid, fileStat.Sys().(*syscall.Stat_t).Uid)
	ExpectEq(ut.gid, fileStat.Sys().(*syscall.Stat_t).Gid)
}

func (ut *utilTest) Test_CreateFile_FileDirNotPresent() {

	file, err := CreateFile(ut.fileSpec, ut.flag)

	ut.assertFileAndDirCreation(file, err)
	ExpectEq(nil, file.Close())
}

func (ut *utilTest) Test_CreateFile_FileDirPresent() {
	err := os.MkdirAll(path.Dir(ut.fileSpec.Path), 0755)
	ExpectEq(nil, err)

	file, err := CreateFile(ut.fileSpec, ut.flag)

	ut.assertFileAndDirCreation(file, err)
	ExpectEq(nil, file.Close())
}

func (ut *utilTest) Test_CreateFile_ReadOnlyFile() {
	ut.flag = os.O_RDONLY

	file, err := CreateFile(ut.fileSpec, ut.flag)

	ut.assertFileAndDirCreation(file, err)
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

	ut.assertFileAndDirCreation(file, err)
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
	ut.fileSpec.Perm = os.FileMode(0755)

	file, err := CreateFile(ut.fileSpec, ut.flag)

	ut.assertFileAndDirCreation(file, err)
	ExpectEq(nil, file.Close())
}

func (ut *utilTest) Test_CreateFile_FilePerm0544() {
	ut.fileSpec.Perm = os.FileMode(0544)

	file, err := CreateFile(ut.fileSpec, ut.flag)

	ut.assertFileAndDirCreation(file, err)
	ExpectEq(nil, file.Close())
}

func (ut *utilTest) Test_CreateFile_FilePresent() {
	err := os.MkdirAll(path.Dir(ut.fileSpec.Path), 0755)
	ExpectEq(nil, err)
	file, err := os.Create(ut.fileSpec.Path)
	ExpectEq(nil, err)
	ExpectEq(nil, file.Close())

	file, err = CreateFile(ut.fileSpec, ut.flag)

	ut.assertFileAndDirCreation(file, err)
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

	ut.assertFileAndDirCreation(file, err)
	ExpectEq(nil, file.Close())
}
