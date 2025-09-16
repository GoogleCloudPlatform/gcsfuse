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
	"github.com/google/uuid"
	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/common"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/util"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// logGcsfuseConfigs logs the configuration values provided by the user in CLI flags, config file
// and optimizations performed on various flags for high performance machine types.
func logGcsfuseConfigs(v *viper.Viper, cmd *cobra.Command, optimizedFlags map[string]interface{}, config cfg.Config) {
	configWrapper := make(map[string]interface{})
	cliFlags := make(map[string]interface{})
	cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		if f.Changed {
			cliFlags[f.Name] = f.Value.String()
		}
	})
	if os.Getenv(logger.GCSFuseInBackgroundMode) == "true" {
		delete(cliFlags, "foreground")
	}
	configWrapper["cli"] = cliFlags
	if v.ConfigFileUsed() != "" {
		configFileViper := viper.New()
		configFileViper.SetConfigFile(v.ConfigFileUsed())
		configFileViper.SetConfigType("yaml")
		if err := configFileViper.ReadInConfig(); err == nil {
			configWrapper["config"] = configFileViper.AllSettings()
		} else {
			log.Printf("Unable to read config file. Error: %v, Config file flags logging skipped", err)
		}
	}
	if len(optimizedFlags) > 0 {
		configWrapper["optimizations"] = optimizedFlags
	}
	configWrapper["gcsfuse"] = config
	logger.Info("GCSFuse config", "config", configWrapper)
}

type mountFn func(uuid string, c *cfg.Config, bucketName, mountPoint string) error

// newRootCmd accepts the mountFn that it executes with the parsed configuration
func newRootCmd(m mountFn) (*cobra.Command, error) {
	var (
		configObj cfg.Config
		cfgFile   string
		cfgErr    error
		v         = viper.New()
	)
	// Generate mount logger Id for logger attribute.
	uuid, err := uuid.NewRandom()
	mountLoggerId := logger.DefaultMountLoggerId
	if err == nil && uuid.String() != "" {
		mountLoggerId = uuid.String()[:8]
	} else {
		log.Printf("Could not generate random UUID for logger, err %v. Falling back to default %v", err, logger.DefaultMountLoggerId)
	}
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
			return m(mountLoggerId, &configObj, bucket, mountPoint)
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
			cfgErr = logger.InitLogFile(configObj.Logging, mountLoggerId)
			if cfgErr != nil {
				return
			}
		}

		optimizedFlags := cfg.Optimize(&configObj, v)

		if cfgErr = cfg.Rationalize(v, &configObj, optimizedFlags); cfgErr != nil {
			return
		}

		// If there is no log-file, then log GCSFuse configs to stdout.
		// Do not log these in stdout in case of daemonized run
		// if these are already being logged into a log-file, otherwise
		// there will be duplicate logs for these in both places (stdout and log-file).
		if configObj.Foreground || configObj.Logging.FilePath == "" {
			logGcsfuseConfigs(v, rootCmd, optimizedFlags, configObj)
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
