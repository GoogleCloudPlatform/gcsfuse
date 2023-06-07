package implicitdir

import (
	"log"
	"os"
	"path"
	"strconv"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/mounting/static_mounting"
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

	if setup.TestBucket() != "" && setup.MountedDirectory() != "" {
		log.Print("Both --testbucket and --mountedDirectory can't be specified at the same time.")
		os.Exit(1)
	}

	// Run tests for mountedDirectory only if --mountedDirectory flag is set.
	setup.RunTestsForMountedDirectoryFlag(m)

	// Run tests for testBucket
	setup.SetUpTestDirForTestBucketFlag()

	successCode := static_mounting.RunTests(flags, m)

	os.Exit(successCode)
}

func CreateDirectoryWithNFiles(numberOfFiles int, dirPath string, prefix string, t *testing.T) {
	err := os.Mkdir(dirPath, setup.FilePermission_0600)
	if err != nil {
		t.Errorf("Error in creating directory: %v", err)
	}

	for i := 1; i <= numberOfFiles; i++ {
		// Create file with name prefix + i
		// e.g. If prefix = temp  then temp1, temp2
		filePath := path.Join(dirPath, prefix+strconv.Itoa(i))
		_, err := os.Create(filePath)
		if err != nil {
			t.Errorf("Create file at %q: %v", dirPath, err)
		}
	}
}

func CreateImplicitDirectory() {
	// Clean the bucket for readonly testing.
	setup.RunScriptForTestData("../testdata/delete_objects.sh", setup.TestBucket())

	// Create implicit directory in bucket for testing.
	setup.RunScriptForTestData("../testdata/create_objects.sh", setup.TestBucket())
}

func CreateExplicitDirectory(t *testing.T) {
	dirPath := path.Join(setup.MntDir(), ExplicitDirectory)
	CreateDirectoryWithNFiles(NumberOfFilesInExplicitDirectory, dirPath, PrefixFileInExplicitDirectory, t)
}
