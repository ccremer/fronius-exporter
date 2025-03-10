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
			args: []string{"--bind-addr", ":9090"},
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
				"SYMO__HEADER": "key1=value1, KEY2= value2",
			},
			verify: func(c *Configuration) {
				assert.Contains(t, c.Symo.Headers, "key1=value1")
				assert.Contains(t, c.Symo.Headers, "KEY2= value2")
			},
		},
		"GivenHeaderEnvVarAndFlag_WhenMultipleHeadersSpecified_ThenTakePrecedenceFromCLI": {
			envs: map[string]string{
				"SYMO__HEADER": "key1=value1, KEY2= value2",
			},
			args: []string{"--symo.header", "key3=value3"},
			verify: func(c *Configuration) {
				assert.Equal(t, c.Symo.Headers, []string{"key3=value3"})
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
	for key := range m {
		os.Unsetenv(key)
	}
}

func Test_parseHeaderString(t *testing.T) {
	tests := map[string]struct {
		given    string
		expected []string
	}{
		"GivenSingleHeader_WhenParsing_LeaveUnchanged": {
			given:    "key=value",
			expected: []string{"key=value"},
		},
		"GivenTwoHeaders_WhenParsing_SplitInTwo": {
			given:    "key1=value1,key2=value2",
			expected: []string{"key1=value1", "key2=value2"},
		},
		"GivenThreeHeaders_WhenParsing_SplitInThree": {
			given:    "key1=value1,key2=value2,key3=value3",
			expected: []string{"key1=value1", "key2=value2", "key3=value3"},
		},
		"GivenMalformedHeaders_WhenParsing_RegardAsPartOfPreviousHeader": {
			given:    "key1=value1,key2value2",
			expected: []string{"key1=value1", "key2value2"},
		},
		"GivenHeadersWithSpace_WhenParsing_TrimSpaceAfterComma": {
			given:    "key1=value1 , key2=value2",
			expected: []string{"key1=value1", "key2=value2"},
		},
		"GivenHeadersWithTrailingComma_WhenParsing_IgnoreEmptyString": {
			given:    "key1=value1 ,",
			expected: []string{"key1=value1"},
		},
		"GivenHeadersWithSpaces_WhenParsing_Include": {
			given:    "key1=value with space,",
			expected: []string{"key1=value with space"},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var result []string
			result = splitHeaderStrings(tt.given, result)
			assert.Equal(t, tt.expected, result)
		})
	}
}
