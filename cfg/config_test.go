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
		"GivenMQTTFlags_WhenSpecified_ThenOverrideDefaults": {
			args: []string{"--mqtt.broker", "tcp://broker:1883", "--mqtt.username", "user", "--mqtt.password", "secret", "--mqtt.base-topic", "/plant/status/", "--mqtt.client-id", "client-1", "--mqtt.queue-size", "32", "--mqtt.reconnect-interval", "9", "--mqtt.availability-topic", "/plant/status/availability/", "--mqtt.availability-payload-up", "up", "--mqtt.availability-payload-down", "down", "--mqtt.discovery-enabled", "--mqtt.discovery-prefix", "/ha/", "--mqtt.discovery-device-name", "Solar", "--mqtt.discovery-device-id", "solar-1", "--mqtt.tls-ca-file", "/tmp/ca.pem", "--mqtt.tls-cert-file", "/tmp/cert.pem", "--mqtt.tls-key-file", "/tmp/key.pem", "--mqtt.tls-server-name", "broker.local", "--mqtt.tls-insecure-skip-verify"},
			verify: func(c *Configuration) {
				assert.Equal(t, "tcp://broker:1883", c.MQTT.Broker)
				assert.Equal(t, "user", c.MQTT.Username)
				assert.Equal(t, "secret", c.MQTT.Password)
				assert.Equal(t, "plant/status", c.MQTT.BaseTopic)
				assert.Equal(t, "client-1", c.MQTT.ClientID)
				assert.Equal(t, 32, c.MQTT.QueueSize)
				assert.Equal(t, 9*time.Second, c.MQTT.ReconnectInterval)
				assert.Equal(t, "plant/status/availability", c.MQTT.AvailabilityTopic)
				assert.Equal(t, "up", c.MQTT.AvailabilityPayloadUp)
				assert.Equal(t, "down", c.MQTT.AvailabilityPayloadDown)
				assert.Equal(t, true, c.MQTT.DiscoveryEnabled)
				assert.Equal(t, "ha", c.MQTT.DiscoveryPrefix)
				assert.Equal(t, "Solar", c.MQTT.DiscoveryDeviceName)
				assert.Equal(t, "solar-1", c.MQTT.DiscoveryDeviceID)
				assert.Equal(t, "/tmp/ca.pem", c.MQTT.TLSCAFile)
				assert.Equal(t, "/tmp/cert.pem", c.MQTT.TLSCertFile)
				assert.Equal(t, "/tmp/key.pem", c.MQTT.TLSKeyFile)
				assert.Equal(t, "broker.local", c.MQTT.TLSServerName)
				assert.Equal(t, true, c.MQTT.TLSInsecureSkipVerify)
			},
		},
		"GivenMQTTEnv_WhenSpecified_ThenParse": {
			envs: map[string]string{
				"MQTT__BROKER":                    "ssl://broker:8883",
				"MQTT__USERNAME":                  "user",
				"MQTT__PASSWORD":                  "secret",
				"MQTT__BASE_TOPIC":                "fronius/site",
				"MQTT__CLIENT_ID":                 "client-1",
				"MQTT__QUEUE_SIZE":                "32",
				"MQTT__RECONNECT_INTERVAL":        "9",
				"MQTT__AVAILABILITY_TOPIC":        "fronius/site/status",
				"MQTT__AVAILABILITY_PAYLOAD_UP":   "up",
				"MQTT__AVAILABILITY_PAYLOAD_DOWN": "down",
				"MQTT__DISCOVERY_ENABLED":         "true",
				"MQTT__DISCOVERY_PREFIX":          "homeassistant",
				"MQTT__DISCOVERY_DEVICE_NAME":     "Solar",
				"MQTT__DISCOVERY_DEVICE_ID":       "solar-1",
				"MQTT__TLS_CA_FILE":               "/tmp/ca.pem",
				"MQTT__TLS_CERT_FILE":             "/tmp/cert.pem",
				"MQTT__TLS_KEY_FILE":              "/tmp/key.pem",
				"MQTT__TLS_SERVER_NAME":           "broker.local",
				"MQTT__TLS_INSECURE_SKIP_VERIFY":  "true",
			},
			verify: func(c *Configuration) {
				assert.Equal(t, "ssl://broker:8883", c.MQTT.Broker)
				assert.Equal(t, "user", c.MQTT.Username)
				assert.Equal(t, "secret", c.MQTT.Password)
				assert.Equal(t, "fronius/site", c.MQTT.BaseTopic)
				assert.Equal(t, "client-1", c.MQTT.ClientID)
				assert.Equal(t, 32, c.MQTT.QueueSize)
				assert.Equal(t, 9*time.Second, c.MQTT.ReconnectInterval)
				assert.Equal(t, "fronius/site/status", c.MQTT.AvailabilityTopic)
				assert.Equal(t, "up", c.MQTT.AvailabilityPayloadUp)
				assert.Equal(t, "down", c.MQTT.AvailabilityPayloadDown)
				assert.Equal(t, true, c.MQTT.DiscoveryEnabled)
				assert.Equal(t, "homeassistant", c.MQTT.DiscoveryPrefix)
				assert.Equal(t, "Solar", c.MQTT.DiscoveryDeviceName)
				assert.Equal(t, "solar-1", c.MQTT.DiscoveryDeviceID)
				assert.Equal(t, "/tmp/ca.pem", c.MQTT.TLSCAFile)
				assert.Equal(t, "/tmp/cert.pem", c.MQTT.TLSCertFile)
				assert.Equal(t, "/tmp/key.pem", c.MQTT.TLSKeyFile)
				assert.Equal(t, "broker.local", c.MQTT.TLSServerName)
				assert.Equal(t, true, c.MQTT.TLSInsecureSkipVerify)
			},
		},
		"GivenMQTTBrokerUnset_ThenKeepDefaultBaseTopic": {
			verify: func(c *Configuration) {
				assert.Equal(t, "fronius-exporter", c.MQTT.BaseTopic)
				assert.Equal(t, "fronius-exporter/availability", c.MQTT.AvailabilityTopic)
			},
		},
		"GivenPollFlags_WhenSpecified_ThenOverrideDefaults": {
			args: []string{"--poll.interval", "15", "--poll.fresh-timeout", "45"},
			verify: func(c *Configuration) {
				assert.Equal(t, 15*time.Second, c.Poll.Interval)
				assert.Equal(t, 45*time.Second, c.Poll.FreshTimeout)
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
