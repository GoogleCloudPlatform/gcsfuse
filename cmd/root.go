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

// NewRootCmd accepts the mountFn that it executes with the parsed configuration
func NewRootCmd(mountFn func(config cfg.Config) error) (*cobra.Command, error) {
	var (
		configObj cfg.Config
		cfgFile   string
		cfgErr    error
		v         = viper.New()
	)
	rootCmd := &cobra.Command{
		Use:   "gcsfuse [flags] bucket mount_point",
		Short: "Mount a specified GCS bucket or all accessible buckets locally",
		Long: `Cloud Storage FUSE is an open source FUSE adapter that lets you mount 
and access Cloud Storage buckets as local file systems. For a technical overview
of Cloud Storage FUSE, see https://cloud.google.com/storage/docs/gcs-fuse.`,
		Version: getVersion(),
		Args:    cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			if cfgErr != nil {
				return fmt.Errorf("error while parsing config: %w", cfgErr)
			}
			return mountFn(configObj)
		},
	}
	initConfig := func() {
		if cfgFile != "" {
			cfgFile, err := util.GetResolvedPath(cfgFile)
			if err != nil {
				logger.Fatal("error while resolving config-file path[%s]: %v", cfgFile, err)
			}
			v.SetConfigFile(cfgFile)
			v.SetConfigType("yaml")
			if err := v.ReadInConfig(); err != nil {
				logger.Fatal("error while reading the config: %v", err)
			}
		}

		cfgErr = v.Unmarshal(&configObj, viper.DecodeHook(cfg.DecodeHook()), func(decoderConfig *mapstructure.DecoderConfig) {
			// By default, viper supports mapstructure tags for unmarshalling. Override that to support yaml tag.
			decoderConfig.TagName = "yaml"
			// Reject the config file if any of the fields in the YAML don't map to the struct.
			decoderConfig.ErrorUnused = true
		},
		)
	}
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config-file", "", "The path to the config file where all gcsfuse related config needs to be specified. "+
		"Refer to 'https://cloud.google.com/storage/docs/gcsfuse-cli#config-file' for possible configurations.")

	// Add all the other flags.
	if err := cfg.BindFlags(v, rootCmd.PersistentFlags()); err != nil {
		return nil, fmt.Errorf("error while declaring/binding flags: %w", err)
	}
	return rootCmd, nil
}
