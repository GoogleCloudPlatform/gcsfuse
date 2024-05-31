// Copyright 2024 Google Inc. All Rights Reserved.
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

// GENERATED CODE - DO NOT EDIT MANUALLY.

package cfg

import (
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Config struct {
	AppName string `yaml:"app-name"`

	Debug DebugConfig `yaml:"debug"`

	FileSystem FileSystemConfig `yaml:"file-system"`
}

type DebugConfig struct {
	ExitOnInvariantViolation bool `yaml:"exit-on-invariant-violation"`

	LogMutex bool `yaml:"log-mutex"`
}

type FileSystemConfig struct {
	FileMode Octal `yaml:"file-mode"`

	Uid int `yaml:"uid"`
}

func BindFlags(flagSet *pflag.FlagSet) error {
	var err error

	flagSet.StringP("app-name", "", "", "The application name of this mount.")

	err = viper.BindPFlag("app-name", flagSet.Lookup("app-name"))
	if err != nil {
		return err
	}

	flagSet.BoolP("debug_fuse", "", true, "This flag is currently unused.")

	err = flagSet.MarkDeprecated("debug_fuse", "This flag is currently unused.")
	if err != nil {
		return err
	}

	flagSet.BoolP("debug_fuse_errors", "", true, "This flag is currently unused.")

	err = flagSet.MarkDeprecated("debug_fuse_errors", "This flag is currently unused.")
	if err != nil {
		return err
	}

	flagSet.BoolP("debug_invariants", "", false, "Exit when internal invariants are violated.")

	err = viper.BindPFlag("debug.exit-on-invariant-violation", flagSet.Lookup("debug_invariants"))
	if err != nil {
		return err
	}

	flagSet.BoolP("debug_mutex", "", false, "Print debug messages when a mutex is held too long.")

	err = viper.BindPFlag("debug.log-mutex", flagSet.Lookup("debug_mutex"))
	if err != nil {
		return err
	}

	flagSet.IntP("file-mode", "", 0, "Permissions bits for files, in octal.")

	err = viper.BindPFlag("file-system.file-mode", flagSet.Lookup("file-mode"))
	if err != nil {
		return err
	}

	flagSet.IntP("uid", "", -1, "UID owner of all inodes.")

	err = viper.BindPFlag("file-system.uid", flagSet.Lookup("uid"))
	if err != nil {
		return err
	}

	return nil
}
