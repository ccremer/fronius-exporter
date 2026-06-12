package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ccremer/fronius-exporter/cfg"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
)

type mqttPublisher struct {
	client          mqtt.Client
	baseTopic       string
	publishCh       chan mqttPendingMessage
	discoveryConfig cfg.MQTTConfig
	symoConfig      cfg.SymoConfig
}

type mqttPendingMessage struct {
	topic string
	body  []byte
}

type mqttEnvelope struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

type homeAssistantDevice struct {
	Identifiers  []string `json:"identifiers"`
	Name         string   `json:"name"`
	Manufacturer string   `json:"manufacturer,omitempty"`
	Model        string   `json:"model,omitempty"`
}

type homeAssistantSensorConfig struct {
	Name              string              `json:"name"`
	UniqueID          string              `json:"unique_id"`
	StateTopic        string              `json:"state_topic"`
	ValueTemplate     string              `json:"value_template"`
	UnitOfMeasurement string              `json:"unit_of_measurement,omitempty"`
	DeviceClass       string              `json:"device_class,omitempty"`
	StateClass        string              `json:"state_class,omitempty"`
	ObjectID          string              `json:"object_id,omitempty"`
	EnabledByDefault  *bool               `json:"enabled_by_default,omitempty"`
	AvailabilityTopic string              `json:"availability_topic,omitempty"`
	PayloadAvailable  string              `json:"payload_available,omitempty"`
	PayloadNotAvail   string              `json:"payload_not_available,omitempty"`
	Device            homeAssistantDevice `json:"device"`
}

type homeAssistantDiscoveryEntity struct {
	ObjectID          string
	Name              string
	StateTopic        string
	ValueTemplate     string
	UnitOfMeasurement string
	DeviceClass       string
	StateClass        string
	EnabledByDefault  *bool
}

func newMQTTPublisher(config cfg.MQTTConfig, symoConfig cfg.SymoConfig, pollConfig cfg.PollConfig) (*mqttPublisher, error) {
	if config.Broker == "" {
		if hasPartialMQTTConfig(config) {
			return nil, fmt.Errorf("mqtt partial configuration: mqtt.broker is required when mqtt.username or mqtt.password is set")
		}
		return nil, nil
	}

	var publisher *mqttPublisher
	opts := mqtt.NewClientOptions().AddBroker(config.Broker)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(config.ReconnectInterval)
	if config.ClientID != "" {
		opts.SetClientID(config.ClientID)
	} else {
		opts.SetClientID(fmt.Sprintf("fronius-exporter-%d", time.Now().UnixNano()))
	}
	if config.AvailabilityTopic != "" {
		opts.SetWill(config.AvailabilityTopic, config.AvailabilityPayloadDown, 0, true)
	}
	opts.SetOnConnectHandler(func(_ mqtt.Client) {
		log.WithField("broker", config.Broker).Info("Connected to MQTT broker")
		if config.AvailabilityTopic != "" && publisher != nil {
			publisher.publish(mqttPendingMessage{topic: config.AvailabilityTopic, body: []byte(config.AvailabilityPayloadUp)}, true)
		}
		if publisher != nil {
			publisher.publishDiscoveryConfigs()
		}
	})
	opts.SetConnectionLostHandler(func(_ mqtt.Client, err error) {
		log.WithError(err).WithField("broker", config.Broker).Warn("Lost connection to MQTT broker")
	})
	if config.Username != "" {
		opts.SetUsername(config.Username)
	}
	if config.Password != "" {
		opts.SetPassword(config.Password)
	}

	tlsConfig, err := newMQTTTLSConfig(config)
	if err != nil {
		return nil, err
	}
	if tlsConfig != nil {
		opts.SetTLSConfig(tlsConfig)
	}

	publisher = &mqttPublisher{
		client:          mqtt.NewClient(opts),
		baseTopic:       strings.Trim(config.BaseTopic, "/"),
		publishCh:       make(chan mqttPendingMessage, config.QueueSize),
		discoveryConfig: config,
		symoConfig:      symoConfig,
	}

	publisher.client.Connect()
	go publisher.run()
	return publisher, nil
}

func newMQTTTLSConfig(config cfg.MQTTConfig) (*tls.Config, error) {
	if config.TLSCAFile == "" && config.TLSCertFile == "" && config.TLSKeyFile == "" && config.TLSServerName == "" && !config.TLSInsecureSkipVerify {
		return nil, nil
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: config.TLSInsecureSkipVerify,
	}
	if config.TLSServerName != "" {
		tlsConfig.ServerName = config.TLSServerName
	}
	if config.TLSCAFile != "" {
		pem, err := os.ReadFile(config.TLSCAFile)
		if err != nil {
			return nil, err
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("could not parse MQTT CA file %q", config.TLSCAFile)
		}
		tlsConfig.RootCAs = pool
	}
	if config.TLSCertFile != "" || config.TLSKeyFile != "" {
		if config.TLSCertFile == "" || config.TLSKeyFile == "" {
			return nil, fmt.Errorf("mqtt TLS client auth requires both tls-cert-file and tls-key-file")
		}
		cert, err := tls.LoadX509KeyPair(config.TLSCertFile, config.TLSKeyFile)
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}
	return tlsConfig, nil
}

func hasPartialMQTTConfig(config cfg.MQTTConfig) bool {
	return config.Username != "" || config.Password != ""
}

func (p *mqttPublisher) PublishJSON(topicSuffix string, payload interface{}) {
	if p == nil {
		return
	}

	topicSuffix = strings.Trim(topicSuffix, "/")
	body, err := json.Marshal(mqttEnvelope{
		Type:      topicSuffix,
		Timestamp: time.Now().UTC(),
		Data:      payload,
	})
	if err != nil {
		log.WithError(err).WithField("topicSuffix", topicSuffix).Warn("Could not marshal MQTT payload")
		return
	}

	topic := p.topic(topicSuffix)

	select {
	case p.publishCh <- mqttPendingMessage{topic: topic, body: body}:
	default:
		log.WithField("topic", topic).Warn("Dropping MQTT message because publish queue is full")
	}
}

func (p *mqttPublisher) run() {
	for msg := range p.publishCh {
		p.publish(msg, false)
	}
}

func (p *mqttPublisher) publish(msg mqttPendingMessage, retained bool) {
	if !p.client.IsConnected() {
		log.WithField("topic", msg.topic).Debug("Skipping MQTT publish because client is not connected")
		return
	}

	token := p.client.Publish(msg.topic, 0, retained, msg.body)
	if ok := token.WaitTimeout(10 * time.Second); !ok {
		log.WithField("topic", msg.topic).Warn("Timed out publishing MQTT message")
		return
	}
	if err := token.Error(); err != nil {
		log.WithError(err).WithField("topic", msg.topic).Warn("Could not publish MQTT message")
	}
}

func (p *mqttPublisher) publishDiscoveryConfigs() {
	if p == nil || !p.discoveryConfig.DiscoveryEnabled || !p.client.IsConnected() {
		return
	}

	for _, entity := range p.discoveryEntities() {
		payload, err := json.Marshal(homeAssistantSensorConfig{
			Name:              entity.Name,
			UniqueID:          p.discoveryConfig.DiscoveryDeviceID + "_" + entity.ObjectID,
			ObjectID:          entity.ObjectID,
			StateTopic:        entity.StateTopic,
			ValueTemplate:     entity.ValueTemplate,
			UnitOfMeasurement: entity.UnitOfMeasurement,
			DeviceClass:       entity.DeviceClass,
			StateClass:        entity.StateClass,
			EnabledByDefault:  entity.EnabledByDefault,
			AvailabilityTopic: p.discoveryConfig.AvailabilityTopic,
			PayloadAvailable:  p.discoveryConfig.AvailabilityPayloadUp,
			PayloadNotAvail:   p.discoveryConfig.AvailabilityPayloadDown,
			Device: homeAssistantDevice{
				Identifiers:  []string{p.discoveryConfig.DiscoveryDeviceID},
				Name:         p.discoveryConfig.DiscoveryDeviceName,
				Manufacturer: "Fronius Exporter",
				Model:        "Fronius Symo",
			},
		})
		if err != nil {
			log.WithError(err).WithField("entity", entity.ObjectID).Warn("Could not marshal MQTT discovery payload")
			continue
		}

		topic := fmt.Sprintf("%s/sensor/%s/%s/config", p.discoveryConfig.DiscoveryPrefix, p.discoveryConfig.DiscoveryDeviceID, entity.ObjectID)
		p.publish(mqttPendingMessage{topic: topic, body: payload}, true)
	}
}

func (p *mqttPublisher) discoveryEntities() []homeAssistantDiscoveryEntity {
	enabled := func(v bool) *bool { return &v }
	entities := []homeAssistantDiscoveryEntity{}

	if p.symoConfig.PowerFlowEnabled {
		entities = append(entities,
			homeAssistantDiscoveryEntity{ObjectID: "site_power_load", Name: "Fronius Site Power Load", StateTopic: p.topic("power-flow"), UnitOfMeasurement: "W", DeviceClass: "power", StateClass: "measurement", EnabledByDefault: enabled(false), ValueTemplate: "{{ value_json.data.Site.P_Load }}"},
			homeAssistantDiscoveryEntity{ObjectID: "site_power_grid", Name: "Fronius Site Power Grid", StateTopic: p.topic("power-flow"), UnitOfMeasurement: "W", DeviceClass: "power", StateClass: "measurement", EnabledByDefault: enabled(false), ValueTemplate: "{{ value_json.data.Site.P_Grid }}"},
			homeAssistantDiscoveryEntity{ObjectID: "site_power_pv", Name: "Fronius Site Power PV", StateTopic: p.topic("power-flow"), UnitOfMeasurement: "W", DeviceClass: "power", StateClass: "measurement", EnabledByDefault: enabled(false), ValueTemplate: "{{ value_json.data.Site.P_PV }}"},
			homeAssistantDiscoveryEntity{ObjectID: "site_power_battery", Name: "Fronius Site Power Battery", StateTopic: p.topic("power-flow"), UnitOfMeasurement: "W", DeviceClass: "power", StateClass: "measurement", EnabledByDefault: enabled(false), ValueTemplate: "{{ value_json.data.Site.P_Akku }}"},
			homeAssistantDiscoveryEntity{ObjectID: "site_energy_day", Name: "Fronius Site Energy Day", StateTopic: p.topic("power-flow"), UnitOfMeasurement: "Wh", DeviceClass: "energy", StateClass: "total_increasing", ValueTemplate: "{{ value_json.data.Site.E_Day }}"},
			homeAssistantDiscoveryEntity{ObjectID: "site_energy_year", Name: "Fronius Site Energy Year", StateTopic: p.topic("power-flow"), UnitOfMeasurement: "Wh", DeviceClass: "energy", StateClass: "total_increasing", ValueTemplate: "{{ value_json.data.Site.E_Year }}"},
			homeAssistantDiscoveryEntity{ObjectID: "site_energy_total", Name: "Fronius Site Energy Total", StateTopic: p.topic("power-flow"), UnitOfMeasurement: "Wh", DeviceClass: "energy", StateClass: "total_increasing", ValueTemplate: "{{ value_json.data.Site.E_Total }}"},
			homeAssistantDiscoveryEntity{ObjectID: "site_autonomy", Name: "Fronius Site Autonomy", StateTopic: p.topic("power-flow"), UnitOfMeasurement: "%", StateClass: "measurement", EnabledByDefault: enabled(false), ValueTemplate: "{{ value_json.data.Site.rel_Autonomy }}"},
			homeAssistantDiscoveryEntity{ObjectID: "site_self_consumption", Name: "Fronius Site Self Consumption", StateTopic: p.topic("power-flow"), UnitOfMeasurement: "%", StateClass: "measurement", ValueTemplate: "{{ value_json.data.Site.rel_SelfConsumption }}"},
			homeAssistantDiscoveryEntity{ObjectID: "inverter_1_power", Name: "Fronius Inverter 1 Power", StateTopic: p.topic("power-flow"), UnitOfMeasurement: "W", DeviceClass: "power", StateClass: "measurement", ValueTemplate: "{{ value_json.data.Inverters['1'].P if value_json.data.Inverters['1'] is defined else 0 }}"},
			homeAssistantDiscoveryEntity{ObjectID: "inverter_1_soc", Name: "Fronius Inverter 1 Battery SoC", StateTopic: p.topic("power-flow"), UnitOfMeasurement: "%", StateClass: "measurement", EnabledByDefault: enabled(false), ValueTemplate: "{{ value_json.data.Inverters['1'].SOC if value_json.data.Inverters['1'] is defined else 0 }}"},
		)
	}
	if p.symoConfig.InverterRealtimeEnabled {
		entities = append(entities,
			homeAssistantDiscoveryEntity{ObjectID: "ac_power", Name: "Fronius AC Power", StateTopic: p.topic("inverter-realtime"), UnitOfMeasurement: "W", DeviceClass: "power", StateClass: "measurement", ValueTemplate: "{{ value_json.data.PAC.Value }}"},
			homeAssistantDiscoveryEntity{ObjectID: "ac_frequency", Name: "Fronius AC Frequency", StateTopic: p.topic("inverter-realtime"), UnitOfMeasurement: "Hz", StateClass: "measurement", ValueTemplate: "{{ value_json.data.FAC.Value }}"},
			homeAssistantDiscoveryEntity{ObjectID: "total_energy_generated", Name: "Fronius Total Energy Generated", StateTopic: p.topic("inverter-realtime"), UnitOfMeasurement: "Wh", DeviceClass: "energy", StateClass: "total_increasing", ValueTemplate: "{{ value_json.data.TOTAL_ENERGY.Value }}"},
			homeAssistantDiscoveryEntity{ObjectID: "dc_current_mppt1", Name: "Fronius DC Current MPPT1", StateTopic: p.topic("inverter-realtime"), UnitOfMeasurement: "A", StateClass: "measurement", EnabledByDefault: enabled(true), ValueTemplate: "{{ value_json.data.IDC.Value }}"},
			homeAssistantDiscoveryEntity{ObjectID: "dc_voltage_mppt1", Name: "Fronius DC Voltage MPPT1", StateTopic: p.topic("inverter-realtime"), UnitOfMeasurement: "V", StateClass: "measurement", EnabledByDefault: enabled(true), ValueTemplate: "{{ value_json.data.UDC.Value }}"},
			homeAssistantDiscoveryEntity{ObjectID: "dc_current_mppt2", Name: "Fronius DC Current MPPT2", StateTopic: p.topic("inverter-realtime"), UnitOfMeasurement: "A", StateClass: "measurement", EnabledByDefault: enabled(false), ValueTemplate: "{{ value_json.data.IDC_2.Value }}"},
			homeAssistantDiscoveryEntity{ObjectID: "dc_voltage_mppt2", Name: "Fronius DC Voltage MPPT2", StateTopic: p.topic("inverter-realtime"), UnitOfMeasurement: "V", StateClass: "measurement", EnabledByDefault: enabled(false), ValueTemplate: "{{ value_json.data.UDC_2.Value }}"},
			homeAssistantDiscoveryEntity{ObjectID: "dc_current_mppt3", Name: "Fronius DC Current MPPT3", StateTopic: p.topic("inverter-realtime"), UnitOfMeasurement: "A", StateClass: "measurement", EnabledByDefault: enabled(false), ValueTemplate: "{{ value_json.data.IDC_3.Value }}"},
			homeAssistantDiscoveryEntity{ObjectID: "dc_voltage_mppt3", Name: "Fronius DC Voltage MPPT3", StateTopic: p.topic("inverter-realtime"), UnitOfMeasurement: "V", StateClass: "measurement", EnabledByDefault: enabled(false), ValueTemplate: "{{ value_json.data.UDC_3.Value }}"},
			homeAssistantDiscoveryEntity{ObjectID: "dc_current_mppt4", Name: "Fronius DC Current MPPT4", StateTopic: p.topic("inverter-realtime"), UnitOfMeasurement: "A", StateClass: "measurement", EnabledByDefault: enabled(false), ValueTemplate: "{{ value_json.data.IDC_4.Value }}"},
			homeAssistantDiscoveryEntity{ObjectID: "dc_voltage_mppt4", Name: "Fronius DC Voltage MPPT4", StateTopic: p.topic("inverter-realtime"), UnitOfMeasurement: "V", StateClass: "measurement", EnabledByDefault: enabled(false), ValueTemplate: "{{ value_json.data.UDC_4.Value }}"},
		)
	}
	if p.symoConfig.MeterRealtimeEnabled {
		entities = append(entities,
			homeAssistantDiscoveryEntity{ObjectID: "meter_energy_produced", Name: "Fronius Meter Energy Produced", StateTopic: p.topic("meter-realtime"), UnitOfMeasurement: "Wh", DeviceClass: "energy", StateClass: "total_increasing", ValueTemplate: "{{ value_json.data.EnergyReal_WAC_Sum_Produced }}"},
			homeAssistantDiscoveryEntity{ObjectID: "meter_energy_consumed", Name: "Fronius Meter Energy Consumed", StateTopic: p.topic("meter-realtime"), UnitOfMeasurement: "Wh", DeviceClass: "energy", StateClass: "total_increasing", ValueTemplate: "{{ value_json.data.EnergyReal_WAC_Sum_Consumed }}"},
		)
	}
	if p.symoConfig.ArchiveEnabled {
		entities = append(entities,
			homeAssistantDiscoveryEntity{ObjectID: "archive_inverter_1_mppt1_current", Name: "Fronius Archive Inverter 1 MPPT1 Current", StateTopic: p.topic("archive"), UnitOfMeasurement: "A", StateClass: "measurement", EnabledByDefault: enabled(true), ValueTemplate: "{{ value_json.data['inverter/1'].Data.Current_DC_String_1.Values['0'] if value_json.data['inverter/1'] is defined else 0 }}"},
			homeAssistantDiscoveryEntity{ObjectID: "archive_inverter_1_mppt1_voltage", Name: "Fronius Archive Inverter 1 MPPT1 Voltage", StateTopic: p.topic("archive"), UnitOfMeasurement: "V", StateClass: "measurement", EnabledByDefault: enabled(true), ValueTemplate: "{{ value_json.data['inverter/1'].Data.Voltage_DC_String_1.Values['0'] if value_json.data['inverter/1'] is defined else 0 }}"},
			homeAssistantDiscoveryEntity{ObjectID: "archive_inverter_1_mppt2_current", Name: "Fronius Archive Inverter 1 MPPT2 Current", StateTopic: p.topic("archive"), UnitOfMeasurement: "A", StateClass: "measurement", EnabledByDefault: enabled(false), ValueTemplate: "{{ value_json.data['inverter/1'].Data.Current_DC_String_2.Values['0'] if value_json.data['inverter/1'] is defined else 0 }}"},
			homeAssistantDiscoveryEntity{ObjectID: "archive_inverter_1_mppt2_voltage", Name: "Fronius Archive Inverter 1 MPPT2 Voltage", StateTopic: p.topic("archive"), UnitOfMeasurement: "V", StateClass: "measurement", EnabledByDefault: enabled(false), ValueTemplate: "{{ value_json.data['inverter/1'].Data.Voltage_DC_String_2.Values['0'] if value_json.data['inverter/1'] is defined else 0 }}"},
		)
	}
	return entities
}

func (p *mqttPublisher) topic(suffix string) string {
	suffix = strings.Trim(suffix, "/")
	if p.baseTopic == "" {
		return suffix
	}
	if suffix == "" {
		return p.baseTopic
	}
	return p.baseTopic + "/" + suffix
}
