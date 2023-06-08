package implicitdir

import (
	"log"
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

const ExplicitDirectory = "explicitDirectory"
const ImplicitDirectory = "implicitDirectory"
const ImplicitSubDirectory = "implicitSubDirectory"
const NumberOfFilesInExplicitDirectory = 2
const NumberOfFilesInImplicitDirectory = 2
const NumberOfFilesInImplicitSubDirectory = 1
const PrefixFileInExplicitDirectory = "fileInExplicitDir"
const FirstFileInExplicitDirectory = "fileInExplicitDir1"
const SecondFileInExplicitDirectory = "fileInExplicitDir2"
const FileInImplicitDirectory = "fileInImplicitDir1"
const FileInImplicitSubDirectory = "fileInImplicitDir2"

func RunTestsForImplicitDir(flags [][]string, m *testing.M) {
	setup.ParseSetUpFlags()

	setup.ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet()

	if setup.TestBucket() == "" && setup.MountedDirectory() != "" {
		log.Print("Please pass the name of bucket mounted at mountedDirectory to --testBucket flag.")
		os.Exit(1)
	}

	// Run tests for mountedDirectory only if --mountedDirectory flag is set.
	setup.RunTestsForMountedDirectoryFlag(m)

	// Run tests for testBucket
	setup.SetUpTestDirForTestBucketFlag()

	successCode := static_mounting.RunTests(flags, m)

	os.Exit(successCode)
}

func CreateImplicitDirectory() {
	// Implicit Directory Structure
	// implicitDirectory                                                  -- Dir
	// implicitDirectory/fileInImplicitDir1                               -- File
	// implicitDirectory/implicitSubDirectory                             -- Dir
	// implicitDirectory/implicitSubDirectory/fileInImplicitDir2          -- File

	// Clean the bucket.
	setup.RunScriptForTestData("../testdata/delete_objects.sh", setup.TestBucket())

	// Create implicit directory in bucket for testing.
	setup.RunScriptForTestData("../testdata/create_objects.sh", setup.TestBucket())
}

func CreateExplicitDirectory(t *testing.T) {
	// Explicit Directory structure
	// explicitDirectory                            -- Dir
	// explicitDirectory/fileInExplicitDir1         -- File
	// explicitDirectory/fileInExplicitDir2         -- File
	
	dirPath := path.Join(setup.MntDir(), ExplicitDirectory)
	operations.CreateDirectoryWithNFiles(NumberOfFilesInExplicitDirectory, dirPath, PrefixFileInExplicitDirectory, t)
}
