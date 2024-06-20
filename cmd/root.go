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

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile   string
	configObj cfg.Config
)
var rootCmd = &cobra.Command{
	Use:   "gcsfuse [flags] bucket mount_point",
	Short: "Mount a specified GCS bucket or all accessible buckets locally",
	Long: `Cloud Storage FUSE is an open source FUSE adapter that lets you mount 
and access Cloud Storage buckets as local file systems. For a technical overview
of Cloud Storage FUSE, see https://cloud.google.com/storage/docs/gcs-fuse.`,
	Version: getVersion(),
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: the following error will be removed once the command is implemented.
		return fmt.Errorf("unsupported operation")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config-file", "", "Absolute path to the config file.")

	// Add all the other flags.
	if err := cfg.BindFlags(rootCmd.PersistentFlags()); err != nil {
		logger.Fatal("error while declaring/binding flags: %v", err)
	}
}

func initConfig() {
	if cfgFile != "" {
		cfgFile, err := util.GetResolvedPath(cfgFile)
		if err != nil {
			logger.Fatal("error while resolving config-file path[%s]: %v", cfgFile, err)
		}
		viper.SetConfigFile(cfgFile)
		viper.SetConfigType("yaml")
		if err := viper.ReadInConfig(); err != nil {
			logger.Fatal("error while reading the config: %v", err)
		}
	}

	err := viper.Unmarshal(&configObj, viper.DecodeHook(cfg.DecodeHook()), func(decoderConfig *mapstructure.DecoderConfig) {
		// By default, viper supports mapstructure tags for unmarshalling. Override that to support yaml tag.
		decoderConfig.TagName = "yaml"
	},
	)
	if err != nil {
		logger.Fatal("error while unmarshalling the config: %v", err)
	}
}
