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

// A helper that allows using gcsfuse with mount(8).
//
// Can be invoked using a command-line of the form expected for mount helpers.
// Calls the gcsfuse binary, which it finds from one of a list of expected
// locations, and waits for it to complete. The device and mount point are
// passed on as positional arguments, and other known options are converted to
// appropriate flags.
//
// This binary returns with exit code zero only after gcsfuse has reported that
// it has successfully mounted the file system. Further output from gcsfuse is
// suppressed.
package main

// Example invocation on OS X:
//
//     mount -t porp -o foo=bar\ baz -o ro,blah bucket ~/tmp/mp
//
// becomes the following arguments:
//
//     Arg 0: "/sbin/mount_gcsfuse "
//     Arg 1: "-o"
//     Arg 2: "foo=bar baz"
//     Arg 3: "-o"
//     Arg 4: "ro"
//     Arg 5: "-o"
//     Arg 6: "blah"
//     Arg 7: "bucket"
//     Arg 8: "/path/to/mp"
//
// On Linux, the fstab entry
//
//     bucket /path/to/mp porp user,foo=bar\040baz
//
// becomes
//
//     Arg 0: "/sbin/mount.gcsfuse"
//     Arg 1: "bucket"
//     Arg 2: "/path/to/mp"
//     Arg 3: "-o"
//     Arg 4: "rw,noexec,nosuid,nodev,user,foo=bar baz"
//

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/mount"
	"github.com/spf13/pflag"
)

func flagTypes(flagSet *pflag.FlagSet) (boolFlags, nonBoolFlags []string) {
	flagSet.VisitAll(func(flag *pflag.Flag) {
		if flag.Value.Type() == "bool" {
			boolFlags = append(boolFlags, flag.Name)
		} else {
			nonBoolFlags = append(nonBoolFlags, flag.Name)
		}
	})
	return boolFlags, nonBoolFlags
}

func isEquiv(s1, s2 string) bool {
	return strings.ReplaceAll(s1, "_", "-") == strings.ReplaceAll(s2, "_", "-")
}

func findEquivFlag(s string, lst []string) string {
	idx := slices.IndexFunc(lst, func(e string) bool {
		return isEquiv(s, e)
	})
	if idx == -1 {
		return ""
	}
	return lst[idx]
}

// Turn mount-style options into gcsfuse arguments. Skip known detritus that
// the mount command gives us.
//
// The result of this function should be appended to exec.Command.Args.
func makeGcsfuseArgs(
	device string,
	mountPoint string,
	opts map[string]string) (args []string, err error) {
	var flagSet pflag.FlagSet
	if err := cfg.BuildFlagSet(&flagSet); err != nil {
		return nil, err
	}
	boolFlags, nonBoolFlags := flagTypes(&flagSet)
	// Handle "o" by not treating it as a flag for persistent mounting.
	nonBoolFlags = slices.DeleteFunc(nonBoolFlags, func(s string) bool {
		return s == "o"
	})
	nonBoolFlags = append(nonBoolFlags, cfg.ConfigFileFlagName)
	// 'rw' mount option is implicitly passed as CLI parameter by /etc/fstab
	// entries and overrides mount options in config file. Ignoring is safe, as
	// 'rw' is default, so that config file mount options take effect.
	noopOptions := []string{"rw", "user", "nouser", "auto", "noauto", "_netdev", "no_netdev"}

	// Deal with options.
	for name, value := range opts {
		if flg := findEquivFlag(name, noopOptions); flg != "" {
			// Don't pass through options that are relevant to mount(8) but not to
			// gcsfuse, and that fusermount chokes on with "Invalid argument" on Linux.
			continue
		}
		if flg := findEquivFlag(name, boolFlags); flg != "" {
			if value == "" {
				value = "true"
			}
			args = append(args, fmt.Sprintf("--%s=%s", flg, value))
		} else if flg := findEquivFlag(name, nonBoolFlags); flg != "" {
			args = append(args, fmt.Sprintf("--%s", flg), value)
		} else {
			// Pass through everything else
			formatted := name
			if value != "" {
				formatted = fmt.Sprintf("%s=%s", name, value)
			}

			args = append(args, "-o", formatted)
		}
	}

	// Set the bucket and mount point.
	return append(args, device, mountPoint), nil
}

// Parse the supplied command-line arguments from a mount(8) invocation on OS X
// or Linux.
func parseArgs(
	args []string) (
	device string,
	mountPoint string,
	opts map[string]string,
	err error) {
	opts = make(map[string]string)

	// Process each argument in turn.
	positionalCount := 0
	for i, s := range args {
		switch {
		// Skip the program name.
		case i == 0:
			continue

		// "-o" is illegal only when at the end. We handle its argument in the case
		// below.
		case s == "-o":
			if i == len(args)-1 {
				err = fmt.Errorf("unexpected -o at end of args")
				return
			}

		// systemd passes -n (alias --no-mtab) to the mount helper. This seems to
		// be a result of the new setup on many Linux systems with /etc/mtab as a
		// symlink pointing to /proc/self/mounts. /proc/self/mounts is read-only,
		// so any helper that would normally write to /etc/mtab should be
		// configured not to do so. Because systemd does not provide a way to
		// disable this behavior for mount helpers that do not write to /etc/mtab,
		// we ignore the flag.
		case s == "-n":
			continue

		// Is this an options string following a "-o"?
		case i > 0 && args[i-1] == "-o":
			mount.ParseOptions(opts, s)

		// Is this the device?
		case positionalCount == 0:
			device = s
			// The kernel might be converting the bucket name to a path if a directory with the
			// same name as the bucket exists in the folder where the mount command is executed.
			//
			// As of October 10th, 2024, bucket names don't support "/", so it is safe to
			// assume any received bucket name containing "/" is actually a path and
			// extract the base file name.
			// Ref: https://cloud.google.com/storage/docs/buckets#naming
			if strings.Contains(device, "/") {
				// Get the last part of the path (bucket name)
				device = filepath.Base(device)
			}
			positionalCount++

		// Is this the mount point?
		case positionalCount == 1:
			mountPoint = s
			positionalCount++

		default:
			err = fmt.Errorf("unexpected arg %d: %q", i, s)
			return
		}
	}

	if positionalCount != 2 {
		err = fmt.Errorf("expected two positional arguments; got %d", positionalCount)
		return
	}

	return
}

func run(args []string) (err error) {
	// If invoked with a single "--help" argument, print a usage message and exit
	// successfully.
	if len(args) == 2 && args[1] == "--help" {
		fmt.Fprintf(
			os.Stderr,
			"Usage: %s [-o options] bucket_name mount_point\n",
			args[0])

		return
	}

	// Find the path to gcsfuse.
	gcsfusePath, err := findGcsfuse()
	if err != nil {
		err = fmt.Errorf("findGcsfuse: %w", err)
		return
	}

	// Find the path to fusermount.
	fusermountPath, err := findFusermount()
	if err != nil {
		err = fmt.Errorf("findFusermount: %w", err)
		return
	}

	// Attempt to parse arguments.
	device, mountPoint, opts, err := parseArgs(args)
	if err != nil {
		err = fmt.Errorf("parseArgs: %w", err)
		return
	}

	// Choose gcsfuse args.
	gcsfuseArgs, err := makeGcsfuseArgs(device, mountPoint, opts)
	if err != nil {
		err = fmt.Errorf("makeGcsfuseArgs: %w", err)
		return
	}

	fmt.Fprintf(
		os.Stderr,
		"Calling gcsfuse with arguments: %s\n",
		strings.Join(gcsfuseArgs, " "))

	// Run gcsfuse.
	cmd := exec.Command(gcsfusePath, gcsfuseArgs...)
	cmd.Env = append(cmd.Env, fmt.Sprintf("PATH=%s", path.Dir(fusermountPath)))
	cmd.Env = append(cmd.Env, fmt.Sprintf("HOME=%s", os.Getenv("HOME")))

	// Pass through the https_proxy/http_proxy environment variable,
	// in case the host requires a proxy server to reach the GCS endpoint.
	// http_proxy has precedence over http_proxy, in case both are set
	if p, ok := os.LookupEnv("https_proxy"); ok {
		cmd.Env = append(cmd.Env, fmt.Sprintf("https_proxy=%s", p))
	} else if p, ok := os.LookupEnv("http_proxy"); ok {
		cmd.Env = append(cmd.Env, fmt.Sprintf("http_proxy=%s", p))
	}
	// Pass through the no_proxy enviroment variable. Whenever
	// using the http(s)_proxy environment variables. This should
	// also be included to know for which hosts the use of proxies
	// should be ignored.
	if p, ok := os.LookupEnv("no_proxy"); ok {
		cmd.Env = append(cmd.Env, fmt.Sprintf("no_proxy=%s", p))
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		err = fmt.Errorf("running gcsfuse: %w", err)
		return
	}

	return
}

func main() {
	err := run(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
