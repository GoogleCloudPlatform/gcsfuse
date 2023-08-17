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

type LocalFileTest struct {
	fsTest
}

func init() {
	RegisterTestSuite(&LocalFileTest{})
}

func (t *LocalFileTest) SetUpTestSuite() {
	t.serverCfg.MountConfig = &config.MountConfig{
		WriteConfig: config.WriteConfig{
			// Making the default value as true to keep it inline with current behaviour.
			CreateEmptyFile: false,
		}}
	t.fsTest.SetUpTestSuite()
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *LocalFileTest) NewFileShouldNotGetSyncedToGCSTillClose() {
	f, err := os.Create(path.Join(mntDir, "foo"))
	AssertEq(nil, err)

	_, err = gcsutil.ReadObject(ctx, bucket, "foo")
	var notFoundErr *gcs.NotFoundError
	ExpectTrue(errors.As(err, &notFoundErr))

	_, err = f.Write([]byte("teststring"))
	AssertEq(nil, err)

	_, err = gcsutil.ReadObject(ctx, bucket, "foo")
	ExpectTrue(errors.As(err, &notFoundErr))

	err = f.Close()
	AssertEq(nil, err)

	_, err = gcsutil.ReadObject(ctx, bucket, "foo")
	ExpectTrue(errors.As(err, &notFoundErr))
}
