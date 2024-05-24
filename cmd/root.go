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

package cmd

import (
	"fmt"
	"os"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/common"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile       string
	bindErr       error
	configFileErr error
	unmarshalErr  error
	MountConfig   cfg.Config
)

var rootCmd = &cobra.Command{
	Use:   "gcsfuse [flags] bucket mount_point",
	Short: "Mount a specified GCS bucket or all accessible buckets locally",
	Long: `Cloud Storage FUSE is an open source FUSE adapter that lets you mount
          and access Cloud Storage buckets as local file systems. For a
          technical overview of Cloud Storage FUSE, see
          https://cloud.google.com/storage/docs/gcs-fuse.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if bindErr != nil {
			return bindErr
		}
		if configFileErr != nil {
			return configFileErr
		}
		if unmarshalErr != nil {
			return unmarshalErr
		}
		common.InitMount()
		// Resolve flags

		return nil
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config-file", "", "Path to the config-file")
	bindErr = cfg.BindFlags(rootCmd.PersistentFlags())
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
		viper.SetConfigType("yaml")
	}

	if err := viper.ReadInConfig(); err != nil {
		configFileErr = fmt.Errorf("error while reading config file: %w", err)
		return
	}
	unmarshalErr = viper.Unmarshal(&MountConfig)
}
