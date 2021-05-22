package cfg

import (
	"net/http"
	"os"
	"testing"
	"time"

	flag "github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func TestConvertHeaders(t *testing.T) {
	type args struct {
		headers []string
		header  *http.Header
	}
	tests := map[string]struct {
		args   args
		verify func(header *http.Header)
	}{
		"WhenEmptyArray_ThenDoNothing": {
			args: args{
				headers: []string{},
				header:  &http.Header{},
			},
			verify: func(header *http.Header) {
				assert.Empty(t, header)
			},
		},
		"WhenInvalidEntry_ThenIgnore": {
			args: args{
				headers: []string{"invalid"},
				header:  &http.Header{},
			},
			verify: func(header *http.Header) {
				assert.Empty(t, header)
			},
		},
		"WhenValidEntry_ThenParse": {
			args: args{
				headers: []string{"Authentication= Bearer <token>"},
				header:  &http.Header{},
			},
			verify: func(header *http.Header) {
				assert.Equal(t, "Bearer <token>", header.Get("Authentication"))
			},
		},
		"GivenValidEntry_WhenSpacesAroundValues_ThenTrim": {
			args: args{
				headers: []string{"  Authentication =   Bearer <token>  "},
				header:  &http.Header{},
			},
			verify: func(header *http.Header) {
				assert.Equal(t, "Bearer <token>", header.Get("Authentication"))
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ConvertHeaders(tt.args.headers, tt.args.header)
			tt.verify(tt.args.header)
		})
	}
}

func TestParseConfig(t *testing.T) {
	tests := map[string]struct {
		args   []string
		envs   map[string]string
		want   *Configuration
		fs     flag.FlagSet
		verify func(c *Configuration)
	}{
		"GivenNoFlags_ThenReturnDefaultConfig": {
			args: []string{},
			verify: func(c *Configuration) {
				assert.Equal(t, "info", c.Log.Level)
			},
		},
		"GivenLogFlags_WhenVerboseEnabled_ThenSetLoggingLevelToDebug": {
			args: []string{"-v"},
			verify: func(c *Configuration) {
				assert.Equal(t, "debug", c.Log.Level)
				assert.Equal(t, true, c.Log.Verbose)
			},
		},
		"GivenLogFlags_WhenLogLevelSpecified_ThenOverrideLogLevel": {
			args: []string{"--log.level=warn"},
			verify: func(c *Configuration) {
				assert.Equal(t, "warn", c.Log.Level)
			},
		},
		"GivenLogFlags_WhenInvalidLogLevelSpecified_ThenSetLoggingLevelToInfo": {
			args: []string{"--log.level=invalid"},
			verify: func(c *Configuration) {
				assert.Equal(t, "info", c.Log.Level)
			},
		},
		"GivenLogLevel_WhenVerboseEnabled_ThenSetLoggingLevelToDebug": {
			args: []string{"--log.level=fatal", "-v"},
			verify: func(c *Configuration) {
				assert.Equal(t, "debug", c.Log.Level)
				assert.Equal(t, true, c.Log.Verbose)
			},
		},
		"GivenFlags_WhenBindAddrSpecified_ThenOverridePort": {
			args: []string{"--bindAddr", ":9090"},
			verify: func(c *Configuration) {
				assert.Equal(t, ":9090", c.BindAddr)
			},
		},
		"GivenHeaderFlags_WhenMultipleHeadersSpecified_ThenFillArray": {
			args: []string{"--symo.header", "key1=value1", "--symo.header", "KEY2= value2"},
			verify: func(c *Configuration) {
				assert.Contains(t, c.Symo.Headers, "key1=value1")
				assert.Contains(t, c.Symo.Headers, "KEY2= value2")
			},
		},
		"GivenHeaderEnvVar_WhenMultipleHeadersSpecified_ThenFillArray": {
			envs: map[string]string{
				"SYMO_HEADER": "key1=value1, KEY2= value2",
			},
			verify: func(c *Configuration) {
				assert.Contains(t, c.Symo.Headers, "key1=value1")
				assert.Contains(t, c.Symo.Headers, " KEY2= value2")
			},
		},
		"GivenUrlFlag_ThenOverrideDefault": {
			args: []string{"--symo.url", "myurl"},
			verify: func(c *Configuration) {
				assert.Equal(t, "myurl", c.Symo.URL)
			},
		},
		"GivenTimeoutFlag_WhenSpecified_ThenOverrideDefault": {
			args: []string{"--symo.timeout", "3"},
			verify: func(c *Configuration) {
				assert.Equal(t, 3*time.Second, c.Symo.Timeout)
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			setEnv(tt.envs)
			result := ParseConfig("version", "commit", "date", &tt.fs, tt.args)
			tt.verify(result)
			unsetEnv(tt.envs)
		})
	}
}

func setEnv(m map[string]string) {
	for key, value := range m {
		os.Setenv(key, value)
	}
}

func unsetEnv(m map[string]string) {
	for key, _ := range m {
		os.Unsetenv(key)
	}
}
