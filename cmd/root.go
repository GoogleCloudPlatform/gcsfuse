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
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	cliViper, cfgViper *viper.Viper
	cfgFileObj, cliObj cfg.Config
	cfgFile            string
	err                error
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
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config-file", "", "config file (default is $HOME/.cobra.yaml)")
	if cliViper, err = cfg.BindFlags(rootCmd.PersistentFlags()); err != nil {
		err = fmt.Errorf("error while binding flags for cli-viper: %w", err)
		return
	}
	cfgFlagset := flag.NewFlagSet("cfg-flagset", flag.ExitOnError)
	if cfgViper, err = cfg.BindFlags(cfgFlagset); err != nil {
		err = fmt.Errorf("error while binding flags for config-viper: %w", err)
		return
	}
}

func initConfig() {
	if err = cliViper.Unmarshal(&cliObj, viper.DecodeHook(cfg.DecodeHook())); err != nil {
		err = fmt.Errorf("error while unmarshaling the cli flags: %w", err)
		return
	}
	if cfgFile == "" {
		return
	}
	// Use config file from the flag.
	cfgViper.SetConfigFile(cfgFile)
	cfgViper.SetConfigType("yaml")
	if err = cfgViper.ReadInConfig(); err != nil {
		err = fmt.Errorf("error while reading the config file: %w", err)
		return
	}
	err = cfgViper.Unmarshal(&cfgFileObj, viper.DecodeHook(cfg.DecodeHook()), func(decoderConfig *mapstructure.DecoderConfig) {
		decoderConfig.TagName = "yaml"
	})
	if err != nil {
		err = fmt.Errorf("error while unmarshaling the config-file params: %w", err)
		return
	}
}
