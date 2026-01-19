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
	"maps"
	"os"
	"slices"
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

type mountInfo struct {
	// cliFlags are the flags passed through the command line to GCSFuse Program.
	// This field is used only for logging purpose.
	cliFlags map[string]string
	// configFileFlags are the flags passed through the config file to GCSFuse Program.
	// This field is used only for logging purpose.
	configFileFlags map[string]any
	// config is the final config object after merging cli and config file flags applying
	// all optimisation based on machineType, Profile etc. This is the final config used for mounting GCSFuse.
	config *cfg.Config
	// optimizedFlags contains the flags that were optimized
	// based on either machine-type or profile.
	optimizedFlags map[string]any
	// isUserSet is used to check if a flag was explicitly set by the user.
	// This is needed for bucket-type-based optimizations.
	isUserSet cfg.IsValueSet
}

type mountFn func(mountInfo *mountInfo, bucketName, mountPoint string) error

// getCliFlags returns the cli flags set by the user in map[string]string format.
func getCliFlags(flagSet *pflag.FlagSet) map[string]string {
	cliFlags := make(map[string]string)
	flagSet.VisitAll(func(f *pflag.Flag) {
		if f.Changed {
			cliFlags[f.Name] = f.Value.String()
		}
	})
	// Do not display --foreground flag to the user in logs if user
	// hasn't passed this flag and was added by GCSFuse during demonized run.
	if _, ok := os.LookupEnv(logger.GCSFuseInBackgroundMode); ok {
		delete(cliFlags, "foreground")
	}
	return cliFlags
}

// getConfigFileFlags returns the flags set by the user in the config file.
func getConfigFileFlags(v *viper.Viper) map[string]any {
	if v.ConfigFileUsed() == "" {
		return nil
	}

	// v.AllSettings() includes defaults, which we don't want.
	// We only want what's explicitly in the config file.
	// We can achieve this by creating a new Viper instance and reading the
	// same config file into it without setting any defaults.
	configOnlyViper := viper.New()
	configOnlyViper.SetConfigFile(v.ConfigFileUsed())
	configOnlyViper.SetConfigType("yaml")
	// We can ignore the error here, as the original viper instance would have already failed.
	_ = configOnlyViper.ReadInConfig()
	return configOnlyViper.AllSettings()
}

// newRootCmd accepts the mountFn that it executes with the parsed configuration
func newRootCmd(m mountFn) (*cobra.Command, error) {
	var (
		mountInfo   mountInfo
		cfgFile     string
		viperConfig = viper.New()
	)
	mountInfo.config = &cfg.Config{}
	rootCmd := &cobra.Command{
		Use:   "gcsfuse [flags] bucket mount_point",
		Short: "Mount a specified GCS bucket or all accessible buckets locally",
		Long: `Cloud Storage FUSE is an open source FUSE adapter that lets you mount 
and access Cloud Storage buckets as local file systems. For a technical overview
of Cloud Storage FUSE, see https://cloud.google.com/storage/docs/gcs-fuse.`,
		Version:      common.GetVersion(),
		Args:         cobra.RangeArgs(2, 3),
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cfgFile != "" {
				resolvedCfgFile, err := util.GetResolvedPath(cfgFile)
				if err != nil {
					return fmt.Errorf("error while resolving config-file path[%s]: %w", cfgFile, err)
				}
				viperConfig.SetConfigFile(resolvedCfgFile)
				viperConfig.SetConfigType("yaml")
				if err := viperConfig.ReadInConfig(); err != nil {
					return fmt.Errorf("error while reading the config: %w", err)
				}
			}

			if err := viperConfig.Unmarshal(mountInfo.config, viper.DecodeHook(cfg.DecodeHook()), func(decoderConfig *mapstructure.DecoderConfig) {
				// By default, viper supports mapstructure tags for unmarshalling. Override that to support yaml tag.
				decoderConfig.TagName = "yaml"
				// Reject the config file if any of the fields in the YAML don't map to the struct.
				decoderConfig.ErrorUnused = true
			},
			); err != nil {
				return fmt.Errorf("error while unmarshalling config: %w", err)
			}
			if err := cfg.ValidateConfig(viperConfig, mountInfo.config); err != nil {
				return fmt.Errorf("invalid config: %w", err)
			}

			mountInfo.isUserSet = viperConfig
			optimizedFlags := mountInfo.config.ApplyOptimizations(viperConfig)
			optimizedFlagNames := slices.Collect(maps.Keys(optimizedFlags))
			for k := range optimizedFlags {
				optimizedFlagNames = append(optimizedFlagNames, k)
			}
			if err := cfg.Rationalize(viperConfig, mountInfo.config, optimizedFlagNames); err != nil {
				return fmt.Errorf("error rationalizing config: %w", err)
			}
			mountInfo.cliFlags = getCliFlags(cmd.PersistentFlags())
			mountInfo.configFileFlags = getConfigFileFlags(viperConfig)
			optimizedFlagsAsHierarchicalMap, err := cfg.CreateHierarchicalOptimizedFlags(optimizedFlags)
			if err != nil {
				logger.Errorf("GCSFuse Config: error creating hierarchical map for optimized flags: %v", err)
				// Log the raw map as a fallback
				optimizedFlagsAsHierarchicalMap = make(map[string]any, len(optimizedFlags))
				for flag, value := range optimizedFlags {
					optimizedFlagsAsHierarchicalMap[flag] = value
				}
			}
			mountInfo.optimizedFlags = optimizedFlagsAsHierarchicalMap
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			bucket, mountPoint, err := populateArgs(args[1:])
			if err != nil {
				return fmt.Errorf("error occurred while extracting the bucket and mountPoint: %w", err)
			}
			return m(&mountInfo, bucket, mountPoint)
		},
	}
	rootCmd.PersistentFlags().StringVar(&cfgFile, cfg.ConfigFileFlagName, "", "The path to the config file where all gcsfuse related config needs to be specified. "+
		"Refer to 'https://cloud.google.com/storage/docs/gcsfuse-cli#config-file' for possible configurations.")

	// Add all the other flags.
	if err := cfg.BuildFlagSet(rootCmd.PersistentFlags()); err != nil {
		return nil, fmt.Errorf("error while declaring flags: %w", err)
	}
	if err := cfg.BindFlags(viperConfig, rootCmd.PersistentFlags()); err != nil {
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
