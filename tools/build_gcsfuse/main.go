// Copyright 2015 Google LLC
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

// A tool that builds gcsfuse binaries, assembles helper scripts, etc., and
// writes them out to a destination directory with the correct hierarchy.
//
// Usage:
//
//	build_gcsfuse src_dir dst_dir version
//
// where src_dir is the root of the gcsfuse git repository (or a tarball
// thereof).
//
// For Linux, writes the following to dst_dir:
//
//	bin/gcsfuse
//	sbin/mount.fuse.gcsfuse
//	sbin/mount.gcsfuse
//
// For OS X:
//
//	bin/gcsfuse
//	sbin/mount_gcsfuse
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"

	"github.com/spf13/pflag"
)

// Build release binaries according to the supplied settings, setting up the
// the file system structure we desire (see package-level comments).
//
// version is the gcsfuse version being built (e.g. "0.11.1"), or a short git
// commit name if this is not for an official release.
func buildBinaries(dstDir, srcDir, version, arch string, buildArgs []string) (err error) {
	osys := runtime.GOOS

	// Create the target structure.
	{
		dirs := []string{
			"bin",
			"sbin",
		}

		for _, d := range dirs {
			err = os.Mkdir(path.Join(dstDir, d), 0755)
			if err != nil {
				err = fmt.Errorf("mkdir: %w", err)
				return
			}
		}
	}

	pathEnv, exists := os.LookupEnv("PATH")
	if !exists {
		err = fmt.Errorf("$PATH not found in OS")
		return
	}

	// Create a directory to become GOPATH for our build below.
	gopath, err := os.MkdirTemp("", "build_gcsfuse_gopath")
	if err != nil {
		err = fmt.Errorf("TempDir: %w", err)
		return
	}
	defer os.RemoveAll(gopath)

	// Create a directory to become GOCACHE for our build below.
	var gocache string
	gocache, err = os.MkdirTemp("", "build_gcsfuse_gocache")
	if err != nil {
		err = fmt.Errorf("TempDir: %w", err)
		return
	}
	defer os.RemoveAll(gocache)

	// Make it appear as if the source directory is at the appropriate position
	// in $GOPATH.
	gcsfuseDir := path.Join(gopath, "src/github.com/googlecloudplatform/gcsfuse")
	err = os.MkdirAll(path.Dir(gcsfuseDir), 0700)
	if err != nil {
		err = fmt.Errorf("MkdirAll: %w", err)
		return
	}

	err = os.Symlink(srcDir, gcsfuseDir)
	if err != nil {
		err = fmt.Errorf("symlink: %w", err)
		return
	}

	// mount(8) expects a different name format on Linux.
	mountHelperName := "mount_gcsfuse"
	if osys == "linux" {
		mountHelperName = "mount.gcsfuse"
	}

	// Build the binaries.
	binaries := []struct {
		goTarget   string
		outputPath string
	}{
		{
			"github.com/googlecloudplatform/gcsfuse/v3",
			"bin/gcsfuse",
		},
		{
			"github.com/googlecloudplatform/gcsfuse/v3/tools/mount_gcsfuse",
			path.Join("sbin", mountHelperName),
		},
	}

	for _, bin := range binaries {
		log.Printf("Building %s to %s", bin.goTarget, bin.outputPath)

		// Set up arguments.
		cmd := exec.Command(
			"go",
			"build",
			"-C",
			srcDir,
			"-o",
			path.Join(dstDir, bin.outputPath))

		if path.Base(bin.outputPath) == "gcsfuse" {
			cmd.Args = append(
				cmd.Args,
				"-ldflags",
				fmt.Sprintf("-X github.com/googlecloudplatform/gcsfuse/v3/common.gcsfuseVersion=%s", version),
			)
			cmd.Args = append(cmd.Args, buildArgs...)
		}

		cmd.Args = append(cmd.Args, bin.goTarget)

		// Set up environment.
		cmd.Env = append(
			os.Environ(),
			"GO15VENDOREXPERIMENT=1",
			"GO111MODULE=auto",
			fmt.Sprintf("PATH=%s", pathEnv),
			fmt.Sprintf("GOROOT=%s", runtime.GOROOT()),
			fmt.Sprintf("GOPATH=%s", gopath),
			fmt.Sprintf("GOCACHE=%s", gocache),
			"CGO_ENABLED=0",
			fmt.Sprintf("GOARCH=%s", arch),
		)

		// Build.
		var output []byte
		output, err = cmd.CombinedOutput()
		if err != nil {
			err = fmt.Errorf("%v: %w\nOutput:\n%s", cmd, err, output)
			if strings.Contains(string(output), "flag provided but not defined: -C") {
				err = fmt.Errorf("%v: %w\nOutput:\n%s\nPlease upgrade to go version 1.20 or higher", cmd, err, output)
			}
			return
		}
	}

	// On Linux, also support `mount -t fuse.gcsfuse`. If there's no explicit
	// helper for this type, /sbin/mount.fuse will call the gcsfuse executable
	// directly, but it doesn't support the right argument format. So we install
	// an explicit helper.
	if osys == "linux" {
		err = os.Symlink("mount.gcsfuse", path.Join(dstDir, "sbin/mount.fuse.gcsfuse"))
		if err != nil {
			err = fmt.Errorf("symlink: %w", err)
			return
		}
	}

	return
}

func run() (err error) {
	var arch = pflag.String("arch", runtime.GOARCH, "Target architecture (e.g., amd64, arm64). Defaults to host architecture.")
	pflag.Parse()

	// Extract arguments.
	args := pflag.Args()
	if len(args) < 3 {
		err = fmt.Errorf("usage: %s [flags] src_dir dst_dir version [build args]", os.Args[0])
		return
	}

	srcDir := args[0]
	dstDir := args[1]
	version := args[2]
	buildArgs := args[3:]

	// Build.
	err = buildBinaries(dstDir, srcDir, version, *arch, buildArgs)
	if err != nil {
		err = fmt.Errorf("buildBinaries: %w", err)
		return
	}

	return
}

func main() {
	log.SetFlags(log.Lmicroseconds)
	// pflag.Parse() is called in run()

	err := run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
