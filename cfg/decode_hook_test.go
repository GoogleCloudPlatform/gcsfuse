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

package cfg

import (
	"os"
	"path"
	"testing"
	"time"

	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func bindFlag(t *testing.T, v *viper.Viper, key string, f *flag.Flag) {
	t.Helper()
	err := v.BindPFlag(key, f)
	if err != nil {
		t.Fatalf("Error occured while binding key: %s to flag: %v", key, err)
	}
}

func TestParsingSuccess(t *testing.T) {
	t.Parallel()
	type TestConfig struct {
		OctalParam       Octal
		BoolParam        bool
		StringParam      string
		IntParam         int
		FloatParam       float64
		DurationParam    time.Duration
		StringSliceParam []string
		IntSliceParam    []int
		LogSeverityParam LogSeverity
		ProtocolParam    Protocol
		PathParam        ResolvedPath
	}
	declareFlags := func() *flag.FlagSet {
		fs := flag.NewFlagSet("test", flag.ExitOnError)
		fs.String("octalParam", "0", "")
		fs.String("stringParam", "", "")
		fs.Int("intParam", 0, "")
		fs.Float64("floatParam", 0.0, "")
		fs.Duration("durationParam", 0*time.Nanosecond, "")
		fs.StringSlice("stringSliceParam", []string{}, "")
		fs.IntSlice("intSliceParam", []int{}, "")
		fs.Bool("boolParam", false, "")
		fs.String("logSeverityParam", "INFO", "")
		fs.String("protocolParam", "http1", "")
		fs.String("pathParam", "", "")
		return fs
	}

	bindFlags := func(fs *flag.FlagSet) *viper.Viper {
		t.Helper()
		v := viper.New()
		bindFlag(t, v, "OctalParam", fs.Lookup("octalParam"))
		bindFlag(t, v, "StringParam", fs.Lookup("stringParam"))
		bindFlag(t, v, "IntParam", fs.Lookup("intParam"))
		bindFlag(t, v, "FloatParam", fs.Lookup("floatParam"))
		bindFlag(t, v, "DurationParam", fs.Lookup("durationParam"))
		bindFlag(t, v, "StringSliceParam", fs.Lookup("stringSliceParam"))
		bindFlag(t, v, "IntSliceParam", fs.Lookup("intSliceParam"))
		bindFlag(t, v, "BoolParam", fs.Lookup("boolParam"))
		bindFlag(t, v, "LogSeverityParam", fs.Lookup("logSeverityParam"))
		bindFlag(t, v, "ProtocolParam", fs.Lookup("protocolParam"))
		bindFlag(t, v, "PathParam", fs.Lookup("pathParam"))
		return v
	}
	tests := []struct {
		name    string
		args    []string
		param   string
		setupFn func()
		testFn  func(*testing.T, TestConfig)
	}{
		{
			name: "Octal",
			args: []string{"--octalParam=755"},
			testFn: func(t *testing.T, c TestConfig) {
				assert.Equal(t, Octal(0755), c.OctalParam)
			},
		},
		{
			name: "Bool1",
			args: []string{"--boolParam"},
			testFn: func(t *testing.T, c TestConfig) {
				assert.True(t, c.BoolParam)
			},
		},
		{
			name: "Bool2",
			args: []string{"--boolParam=true"},
			testFn: func(t *testing.T, c TestConfig) {
				assert.True(t, c.BoolParam)
			},
		},
		{
			name: "Bool3",
			args: []string{"--boolParam=false"},
			testFn: func(t *testing.T, c TestConfig) {
				assert.False(t, c.BoolParam)
			},
		},
		{
			name: "String",
			args: []string{"--stringParam=abc"},
			testFn: func(t *testing.T, c TestConfig) {
				assert.Equal(t, "abc", c.StringParam)
			},
		},
		{
			name: "Int",
			args: []string{"--intParam=23"},
			testFn: func(t *testing.T, c TestConfig) {
				assert.Equal(t, 23, c.IntParam)
			},
		},
		{
			name: "Float",
			args: []string{"--floatParam=2.5"},
			testFn: func(t *testing.T, c TestConfig) {
				assert.Equal(t, 2.5, c.FloatParam)
			},
		},
		{
			name: "Duration",
			args: []string{"--durationParam=30s"},
			testFn: func(t *testing.T, c TestConfig) {
				assert.Equal(t, 30*time.Second, c.DurationParam)
			},
		},
		{
			name: "Duration2",
			args: []string{"--durationParam=1h5m30s"},
			testFn: func(t *testing.T, c TestConfig) {
				assert.Equal(t, 1*time.Hour+5*time.Minute+30*time.Second, c.DurationParam)
			},
		},
		{
			name: "StringSlice1",
			args: []string{"--stringSliceParam=a,b"},
			testFn: func(t *testing.T, c TestConfig) {
				assert.ElementsMatch(t, []string{"a", "b"}, c.StringSliceParam)
			},
		},
		{
			name: "StringSlice2",
			args: []string{"--stringSliceParam=a", "--stringSliceParam=b"},
			testFn: func(t *testing.T, c TestConfig) {
				assert.ElementsMatch(t, []string{"a", "b"}, c.StringSliceParam)
			},
		},
		{
			name: "IntSlice1",
			args: []string{"--intSliceParam=2,-11"},
			testFn: func(t *testing.T, c TestConfig) {
				assert.ElementsMatch(t, []int{2, -11}, c.IntSliceParam)
			},
		},
		{
			name: "IntSlice2",
			args: []string{"--intSliceParam=3", "--intSliceParam=5"},
			testFn: func(t *testing.T, c TestConfig) {
				assert.ElementsMatch(t, []int{3, 5}, c.IntSliceParam)
			},
		},
		{
			name: "LogSeverity",
			args: []string{"--logSeverityParam=WARNING"},
			testFn: func(t *testing.T, c TestConfig) {
				assert.Equal(t, LogSeverity("WARNING"), c.LogSeverityParam)
			},
		},
		{
			name: "ResolvedPath1",
			args: []string{"--pathParam=~/test.txt"},
			testFn: func(t *testing.T, c TestConfig) {
				h, err := os.UserHomeDir()
				if assert.Nil(t, err) {
					assert.Equal(t, path.Join(h, "test.txt"), string(c.PathParam))
				}
			},
		},
		{
			name: "ResolvedPath - with gcsfuse-parent-process-dir env set",
			setupFn: func() {
				os.Setenv("gcsfuse-parent-process-dir", "/a")
				t.Cleanup(func() { os.Unsetenv("gcsfuse-parent-process-dir") })
			},
			args: []string{"--pathParam=./test.txt"},
			testFn: func(t *testing.T, c TestConfig) {
				assert.Equal(t, "/a/test.txt", string(c.PathParam))
			},
		},
		{
			name: "ResolvedPath - absolute path",
			args: []string{"--pathParam=/a/test.txt"},
			testFn: func(t *testing.T, c TestConfig) {
				assert.Equal(t, "/a/test.txt", string(c.PathParam))
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.setupFn != nil {
				tc.setupFn()
			}
			c := TestConfig{}
			fs := declareFlags()
			v := bindFlags(fs)
			args := []string{"test"}
			args = append(args, tc.args...)
			err := fs.Parse(args)
			if err != nil {
				t.Fatalf("Flag parsing failed: %v", err)
			}

			err = v.Unmarshal(&c, viper.DecodeHook(DecodeHook()))

			if assert.Nil(t, err) {
				tc.testFn(t, c)
			}
		})
	}
}

func TestParsingError(t *testing.T) {
	t.Parallel()
	type TestConfig struct {
		OctalParam       Octal
		LogSeverityParam LogSeverity
		ProtocolParam    Protocol
	}
	declareFlags := func() *flag.FlagSet {
		fs := flag.NewFlagSet("test", flag.ExitOnError)
		fs.String("octalParam", "0", "")
		fs.String("logSeverityParam", "INFO", "")
		fs.String("protocolParam", "http1", "")
		return fs
	}
	bindFlags := func(fs *flag.FlagSet) *viper.Viper {
		t.Helper()
		v := viper.New()
		bindFlag(t, v, "OctalParam", fs.Lookup("octalParam"))
		bindFlag(t, v, "LogSeverityParam", fs.Lookup("logSeverityParam"))
		bindFlag(t, v, "ProtocolParam", fs.Lookup("protocolParam"))
		return v
	}
	tests := []struct {
		name   string
		args   []string
		param  string
		errMsg string
	}{
		{
			name: "Octal",
			args: []string{"--octalParam=923"},
		},
		{
			name:   "LogSeverity",
			args:   []string{"--logSeverityParam=abc"},
			errMsg: "invalid log severity level: abc. Must be one of [TRACE, DEBUG, INFO, WARNING, ERROR, OFF]",
		},
		{
			name:   "Protocol",
			args:   []string{"--protocolParam=pqr"},
			errMsg: "invalid protocol value: pqr. It can only accept values in the list: [http1 http2 grpc]",
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fs := declareFlags()
			v := bindFlags(fs)
			c := TestConfig{}
			args := []string{"test"}
			args = append(args, tc.args...)
			err := fs.Parse(args)
			if err != nil {
				t.Fatalf("Flag parsing failed: %v", err)
			}

			err = v.Unmarshal(&c, viper.DecodeHook(DecodeHook()))

			if assert.NotNil(t, err) && tc.errMsg != "" {
				assert.ErrorContains(t, err, tc.errMsg)
			}
		})
	}
}
