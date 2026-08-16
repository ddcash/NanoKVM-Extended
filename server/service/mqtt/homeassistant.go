package mqtt

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Home Assistant MQTT discovery.
//
// Publishing a retained config document under the discovery prefix makes HA
// create the entity by itself; removing it (an empty retained payload) makes HA
// delete it. Everything below is built from that one mechanism.
//
// https://www.home-assistant.io/integrations/mqtt/#mqtt-discovery
const (
	defaultDiscoveryPrefix = "homeassistant"
	manufacturer           = "Sipeed"
	model                  = "NanoKVM"
)

type haDevice struct {
	Identifiers  []string `json:"identifiers"`
	Name         string   `json:"name"`
	Manufacturer string   `json:"manufacturer"`
	Model        string   `json:"model"`
	SwVersion    string   `json:"sw_version,omitempty"`
}

type haEntity struct {
	Name              string   `json:"name"`
	UniqueId          string   `json:"unique_id"`
	Device            haDevice `json:"device"`
	AvailabilityTopic string   `json:"availability_topic"`
	StateTopic        string   `json:"state_topic,omitempty"`
	CommandTopic      string   `json:"command_topic,omitempty"`
	Icon              string   `json:"icon,omitempty"`
	DeviceClass       string   `json:"device_class,omitempty"`
	EntityCategory    string   `json:"entity_category,omitempty"`
	PayloadPress      string   `json:"payload_press,omitempty"`
	Topic             string   `json:"topic,omitempty"`
	ImageEncoding     string   `json:"image_encoding,omitempty"`
	ValueTemplate     string   `json:"value_template,omitempty"`
}

// nodeId identifies this device within the discovery topic tree. MQTT topics
// only accept a restricted character set, so anything else is replaced.
func nodeId(cfg *Config) string {
	id := strings.TrimSpace(cfg.HaNodeId)
	if id == "" {
		id = "nanokvm"
	}

	var b strings.Builder
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_', r == '-':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + 32)
		default:
			b.WriteRune('_')
		}
	}

	return b.String()
}

func discoveryPrefix(cfg *Config) string {
	if strings.TrimSpace(cfg.HaDiscoveryPrefix) == "" {
		return defaultDiscoveryPrefix
	}
	return strings.TrimSpace(cfg.HaDiscoveryPrefix)
}

// baseTopic is where this device's own state and command topics live, kept
// separate from the discovery prefix so the two can be configured apart.
func baseTopic(cfg *Config) string {
	return fmt.Sprintf("nanokvm/%s", nodeId(cfg))
}

func availabilityTopic(cfg *Config) string {
	return baseTopic(cfg) + "/availability"
}

func device(cfg *Config, version string) haDevice {
	node := nodeId(cfg)
	name := strings.TrimSpace(cfg.HaDeviceName)
	if name == "" {
		name = "NanoKVM"
	}

	return haDevice{
		Identifiers:  []string{node},
		Name:         name,
		Manufacturer: manufacturer,
		Model:        model,
		SwVersion:    version,
	}
}

// configTopic builds the retained discovery topic for one entity.
func configTopic(cfg *Config, component string, objectId string) string {
	return fmt.Sprintf("%s/%s/%s/%s/config", discoveryPrefix(cfg), component, nodeId(cfg), objectId)
}

func marshalEntity(entity *haEntity) ([]byte, error) {
	data, err := json.Marshal(entity)
	if err != nil {
		return nil, fmt.Errorf("marshal discovery payload: %w", err)
	}
	return data, nil
}
