package fs_test

import (
	"errors"
	"os"
	"path"

	"github.com/googlecloudplatform/gcsfuse/internal/config"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsutil"
	. "github.com/jacobsa/ogletest"
)

const FileName = "foo"
const FileContents = "teststring"

type LocalFileTest struct {
	fsTest
}

func init() {
	RegisterTestSuite(&LocalFileTest{})
}

func (t *LocalFileTest) SetUpTestSuite() {
	t.serverCfg.ImplicitDirectories = true
	t.serverCfg.MountConfig = &config.MountConfig{
		WriteConfig: config.WriteConfig{
			CreateEmptyFile: false,
		}}
	t.fsTest.SetUpTestSuite()
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *LocalFileTest) NewFileShouldNotGetSyncedToGCSTillClose() {
	t.newFileShouldGetSyncedToGCSAtClose(FileName)
}

func (t *LocalFileTest) NewFileUnderExplicitDirectoryShouldNotGetSyncedToGCSTillClose() {
	err := os.Mkdir(path.Join(mntDir, "explicit"), dirPerms)
	AssertEq(nil, err)

	t.newFileShouldGetSyncedToGCSAtClose("explicit/foo")
}

func (t *LocalFileTest) NewFileUnderImplicitDirectoryShouldNotGetSyncedToGCSTillClose() {
	AssertEq(
		nil,
		t.createObjects(
			map[string]string{
				// File
				"implicitFoo/bar": "",
			}))

	t.newFileShouldGetSyncedToGCSAtClose("implicitFoo/foo")
}

func (t *LocalFileTest) newFileShouldGetSyncedToGCSAtClose(fileName string) {
	// Creating a file shouldn't create file on GCS.
	f, err := os.Create(path.Join(mntDir, fileName))
	AssertEq(nil, err)
	t.validateObjectNotFoundErr(fileName)

	// Writing contents to local file shouldn't create file on GCS.
	_, err = f.Write([]byte(FileContents))
	AssertEq(nil, err)
	t.validateObjectNotFoundErr(fileName)

	// Close the file and validate if the file is created on GCS.
	t.closeFileAndValidateObjectContents(f, fileName, FileContents)
}

func (t *LocalFileTest) StatOnLocalFile() {
	// Creating a file shouldn't create file on GCS.
	fileName := path.Join(mntDir, FileName)
	f, err := os.Create(fileName)
	AssertEq(nil, err)
	t.validateObjectNotFoundErr(FileName)

	// Stat the local file.
	fi, err := os.Stat(fileName)
	AssertEq(nil, err)
	ExpectEq(path.Base(fileName), fi.Name())
	ExpectEq(0, fi.Size())
	ExpectEq(filePerms, fi.Mode())

	// Writing contents to local file shouldn't create file on GCS.
	_, err = f.Write([]byte(FileContents))
	AssertEq(nil, err)
	t.validateObjectNotFoundErr(FileName)

	// Stat the local file again to check if new contents are written.
	fi, err = os.Stat(fileName)
	AssertEq(nil, err)
	ExpectEq(path.Base(fileName), fi.Name())
	ExpectEq(10, fi.Size())
	ExpectEq(filePerms, fi.Mode())

	// Close the file and validate if the file is created on GCS.
	t.closeFileAndValidateObjectContents(f, FileName, FileContents)
}

func (t *LocalFileTest) TruncateLocalFile() {
	// Creating a file shouldn't create file on GCS.
	fileName := path.Join(mntDir, FileName)
	f, err := os.Create(fileName)
	AssertEq(nil, err)
	t.validateObjectNotFoundErr(FileName)

	// Writing contents to local file .
	_, err = f.Write([]byte(FileContents))
	AssertEq(nil, err)

	// Stat the file to validate if new contents are written.
	fi, err := os.Stat(fileName)
	AssertEq(nil, err)
	ExpectEq(path.Base(fileName), fi.Name())
	ExpectEq(10, fi.Size())
	ExpectEq(filePerms, fi.Mode())

	// Truncate the file to update the file size.
	err = os.Truncate(fileName, 5)
	AssertEq(nil, err)
	t.validateObjectNotFoundErr(FileName)

	// Stat the file to validate if file is truncated correctly.
	fi, err = os.Stat(fileName)
	AssertEq(nil, err)
	ExpectEq(path.Base(fileName), fi.Name())
	ExpectEq(5, fi.Size())
	ExpectEq(filePerms, fi.Mode())

	// Close the file and validate if the file is created on GCS.
	t.closeFileAndValidateObjectContents(f, FileName, "tests")
}

func (t *LocalFileTest) MultipleWritesToLocalFile() {
	// Creating a file shouldn't create file on GCS.
	f, err := os.Create(path.Join(mntDir, FileName))
	AssertEq(nil, err)
	t.validateObjectNotFoundErr(FileName)

	// Write some contents to file sequentially.
	_, err = f.Write([]byte("string1"))
	AssertEq(nil, err)
	_, err = f.Write([]byte("string2"))
	AssertEq(nil, err)
	_, err = f.Write([]byte("string3"))
	AssertEq(nil, err)
	// File shouldn't get created on GCS.
	t.validateObjectNotFoundErr(FileName)

	// Close the file and validate if the file is created on GCS.
	t.closeFileAndValidateObjectContents(f, FileName, "string1string2string3")
}

func (t *LocalFileTest) RandomWritesToLocalFile() {
	// Creating a file shouldn't create file on GCS.
	f, err := os.Create(path.Join(mntDir, FileName))
	AssertEq(nil, err)
	t.validateObjectNotFoundErr(FileName)

	// Write some contents to file randomly.
	_, err = f.WriteAt([]byte("string1"), 0)
	AssertEq(nil, err)
	_, err = f.WriteAt([]byte("string2"), 2)
	AssertEq(nil, err)
	_, err = f.WriteAt([]byte("string3"), 3)
	AssertEq(nil, err)
	t.validateObjectNotFoundErr(FileName)

	// Close the file and validate if the file is created on GCS.
	t.closeFileAndValidateObjectContents(f, FileName, "stsstring3")
}

func (t *LocalFileTest) validateObjectNotFoundErr(fileName string) {
	var notFoundErr *gcs.NotFoundError
	_, err := gcsutil.ReadObject(ctx, bucket, fileName)

	ExpectTrue(errors.As(err, &notFoundErr))
}

func (t *LocalFileTest) closeFileAndValidateObjectContents(f *os.File, fileName string, contents string) {
	err := f.Close()
	AssertEq(nil, err)

	contentBytes, err := gcsutil.ReadObject(ctx, bucket, fileName)
	AssertEq(nil, err)
	ExpectEq(contents, string(contentBytes))
}
