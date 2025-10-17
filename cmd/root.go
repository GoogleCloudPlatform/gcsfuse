// Copyright 2024 Google LLC
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
	"log"
	"os"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/common"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/util"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// logUserSpecifiedAndOptimizedConfig logs the configuration values provided by the user,
// distinguishing between flags set on the command line and values from the
// config file and the flag sets that were optimized based on machine type.
func logUserSpecifiedAndOptimizedConfig(v *viper.Viper, cmd *cobra.Command, optimizedFlags map[string]any) {
	cliFlags := make(map[string]interface{})
	cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		if f.Changed {
			cliFlags[f.Name] = f.Value.String()
		}
	})
	if len(cliFlags) > 0 {
		logger.Info("GCSFuse CLI", "flags", cliFlags)
	}

	if v.ConfigFileUsed() != "" {
		configFileViper := viper.New()
		configFileViper.SetConfigFile(v.ConfigFileUsed())
		configFileViper.SetConfigType("yaml")
		if err := configFileViper.ReadInConfig(); err == nil {
			logger.Info("GCSFuse Config", "flags", configFileViper.AllSettings())
		}
	}
	if len(optimizedFlags) > 0 {
		logger.Info("GCSFuse machine type based optimized flags", "flags", optimizedFlags)
	}
}

type mountFn func(c *cfg.Config, bucketName, mountPoint string) error

// pflagAsIsValueSet is an adapter that makes a pflag.FlagSet satisfy the
// cfg.isValueSet interface, allowing us to check for user-set flags reliably.
type pflagAsIsValueSet struct {
	fs *pflag.FlagSet
}

// IsSet correctly checks if a flag was set by the user on the command line.
func (p *pflagAsIsValueSet) IsSet(name string) bool {
	// The pflag.Changed method is the reliable way to check this.
	return p.fs.Changed(name)
}

// GetString is required to satisfy the interface used by getMachineType.
func (p *pflagAsIsValueSet) GetString(name string) string {
	val, err := p.fs.GetString(name)
	if err != nil {
		return ""
	}
	return val
}

// GetBool is required to satisfy the interface.
func (p *pflagAsIsValueSet) GetBool(name string) bool {
	val, err := p.fs.GetBool(name)
	if err != nil {
		return false
	}
	return val
}

// newRootCmd accepts the mountFn that it executes with the parsed configuration
func newRootCmd(m mountFn) (*cobra.Command, error) {
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
		Version:      common.GetVersion(),
		Args:         cobra.RangeArgs(2, 3),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if cfgErr != nil {
				return fmt.Errorf("error while parsing config: %w", cfgErr)
			}
			bucket, mountPoint, err := populateArgs(args[1:])
			if err != nil {
				return fmt.Errorf("error occurred while extracting the bucket and mountPoint: %w", err)
			}
			return m(&configObj, bucket, mountPoint)
		},
	}
	initConfig := func() {
		if cfgFile != "" {
			cfgFile, err := util.GetResolvedPath(cfgFile)
			if err != nil {
				cfgErr = fmt.Errorf("error while resolving config-file path[%s]: %w", cfgFile, err)
				return
			}
			v.SetConfigFile(cfgFile)
			v.SetConfigType("yaml")
			if err := v.ReadInConfig(); err != nil {
				cfgErr = fmt.Errorf("error while reading the config: %w", err)
				return
			}
		}

		if cfgErr = v.Unmarshal(&configObj, viper.DecodeHook(cfg.DecodeHook()), func(decoderConfig *mapstructure.DecoderConfig) {
			// By default, viper supports mapstructure tags for unmarshalling. Override that to support yaml tag.
			decoderConfig.TagName = "yaml"
			// Reject the config file if any of the fields in the YAML don't map to the struct.
			decoderConfig.ErrorUnused = true
		},
		); cfgErr != nil {
			return
		}
		if cfgErr = cfg.ValidateConfig(v, &configObj); cfgErr != nil {
			return
		}
		logger.SetLogFormat(configObj.Logging.Format)

		if configObj.Foreground {
			cfgErr = logger.InitLogFile(configObj.Logging)
			if cfgErr != nil {
				return
			}
		}

		isSet := &pflagAsIsValueSet{fs: rootCmd.PersistentFlags()}
		optimizedFlags := configObj.ApplyOptimizations(isSet)
		logUserSpecifiedAndOptimizedConfig(v, rootCmd, optimizedFlags)

		if cfgErr = cfg.Rationalize(v, &configObj, optimizedFlags); cfgErr != nil {
			return
		}
	}
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, cfg.ConfigFileFlagName, "", "The path to the config file where all gcsfuse related config needs to be specified. "+
		"Refer to 'https://cloud.google.com/storage/docs/gcsfuse-cli#config-file' for possible configurations.")

	// Add all the other flags.
	if err := cfg.BuildFlagSet(rootCmd.PersistentFlags()); err != nil {
		return nil, fmt.Errorf("error while declaring flags: %w", err)
	}
	if err := cfg.BindFlags(v, rootCmd.PersistentFlags()); err != nil {
		return nil, fmt.Errorf("error while binding flags: %w", err)
	}
	return rootCmd, nil
}

// convertToPosixArgs converts a slice of commandline args and transforms them
// into POSIX compliant args. All it does is that it converts flags specified
// using a single-hyphen to double-hyphens. We are excluding "-v" because it's
// reserved for showing version in Cobra.
func convertToPosixArgs(args []string, c *cobra.Command) []string {
	pArgs := make([]string, 0, len(args))
	flagSet := make(map[string]bool)
	c.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		flagSet[f.Name] = true
	})
	// Treat help and version like flags
	flagSet["version"] = true
	flagSet["help"] = true
	for _, a := range args {
		switch {
		case a == "--v", a == "-v":
			pArgs = append(pArgs, "-v")
		case a == "--h", a == "-h":
			pArgs = append(pArgs, "-h")
		case strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "--"):
			// Remove the string post the "=" sign.
			// This converts -a=b to -a.
			flg, _, _ := strings.Cut(a, "=")
			// Remove one hyphen from the beginning.
			// This converts -a -> a.
			flg, _ = strings.CutPrefix(flg, "-")

			if flagSet[flg] {
				// "a" is a full-form flag which has been specified with a single hyphen.
				// So add another hyphen so that pflag processes it correctly.
				pArgs = append(pArgs, "-"+a)
			} else {
				// "a" is a flag so, keep it as is.
				pArgs = append(pArgs, a)
			}
		default:
			pArgs = append(pArgs, a)
		}
	}
	return pArgs
}

var ExecuteMountCmd = func() {
	rootCmd, err := newRootCmd(Mount)
	if err != nil {
		log.Fatalf("Error occurred while creating the root command on gcsfuse/%s: %v", common.GetVersion(), err)
	}
	rootCmd.SetArgs(convertToPosixArgs(os.Args, rootCmd))
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Error occurred during command execution on gcsfuse/%s: %v", common.GetVersion(), err)
	}
}
