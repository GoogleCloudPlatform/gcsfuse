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
	// cfgFileViper is intended to support the use-cases where the config-file has
	// higher precedence over cli. In such cases, one can use cfgFileViper.IsSet(<key>)
	// to determine whether the config file has the value set. If so use it,
	// otherwise, use the object unmarshalled from v.
	v, cfgFileViper *viper.Viper

	// configObj is the config object that is unmarshalled from both CLI and the Config file.
	// cfgFileConfigObj is the config that is unmarshalled from the Config file alone.
	configObj, cfgFileConfigObj cfg.Config
	cfgFile                     string
	err                         error
)

var rootCmd = &cobra.Command{
	Use:   "gcsfuse [flags] bucket mount_point",
	Short: "Mount a specified GCS bucket or all accessible buckets locally",
	Long: `Cloud Storage FUSE is an open source FUSE adapter that lets you mount 
and access Cloud Storage buckets as local file systems. For a technical overview
of Cloud Storage FUSE, see https://cloud.google.com/storage/docs/gcs-fuse.`,
	Version: getVersion(),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err != nil {
			return err
		}
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
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config-file", "", "absolute path to the config file")
	if v, err = cfg.BindFlags(rootCmd.PersistentFlags()); err != nil {
		err = fmt.Errorf("error while binding flags to viper: %w", err)
		return
	}
	cfgFlagset := flag.NewFlagSet("cfg-flagset", flag.ExitOnError)
	if cfgFileViper, err = cfg.BindFlags(cfgFlagset); err != nil {
		err = fmt.Errorf("error while binding flags to config-viper: %w", err)
		return
	}
}

func ReadConfig(v *viper.Viper) (err error) {
	if cfgFile == "" {
		return nil
	}
	v.SetConfigFile(cfgFile)
	v.SetConfigType("yaml")
	if err = v.ReadInConfig(); err != nil {
		return fmt.Errorf("error while reading the config file: %w", err)
	}
	return nil
}

func initConfig() {
	if cfgFile == "" {
		return
	}
	if err = ReadConfig(cfgFileViper); err != nil {
		err = fmt.Errorf("error while reading config for the cfg-viper: %w", err)
	}
	if err = ReadConfig(v); err != nil {
		err = fmt.Errorf("error while reading config for viper: %w", err)
	}
	decOpts := []viper.DecoderConfigOption{viper.DecodeHook(cfg.DecodeHook()), func(decoderConfig *mapstructure.DecoderConfig) {
		// By default, viper supports mapstructure tags for unmarshaling. Override that to support yaml tag.
		decoderConfig.TagName = "yaml"
	}}
	if err = v.Unmarshal(&configObj, decOpts...); err != nil {
		err = fmt.Errorf("error while unmarshaling params: %w", err)
		return
	}

	err = cfgFileViper.Unmarshal(&cfgFileConfigObj, decOpts...)
	if err != nil {
		err = fmt.Errorf("error while unmarshaling the config-file params: %w", err)
		return
	}
}
