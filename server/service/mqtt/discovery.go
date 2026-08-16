package mqtt

import (
	"fmt"
	"os"
	"strings"

	paho "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"

	"NanoKVM-Server/service/switcher"
)

// publishDiscovery creates this device's entities in Home Assistant.
//
// Every config document is retained, so Home Assistant restores the entities
// after a restart without this device having to be online.
func publishDiscovery(client paho.Client, cfg *Config) {
	version := readVersion()
	dev := device(cfg, version)
	base := baseTopic(cfg)
	avail := availabilityTopic(cfg)
	node := nodeId(cfg)

	// Whether a machine is actually connected and powered on the far side.
	hdmi := &haEntity{
		Name:              "HDMI signal",
		UniqueId:          node + "_hdmi",
		Device:            dev,
		AvailabilityTopic: avail,
		StateTopic:        base + "/hdmi",
		DeviceClass:       "connectivity",
		EntityCategory:    "diagnostic",
	}
	publishEntity(client, cfg, "binary_sensor", "hdmi", hdmi)

	publishSwitcherButtons(client, cfg, dev, avail, base, node)
}

// publishSwitcherButtons mirrors every configured KVM target as a Home
// Assistant button, so an automation can select a machine the same way the
// menu does.
func publishSwitcherButtons(
	client paho.Client, cfg *Config, dev haDevice, avail string, base string, node string,
) {
	targets, err := switcher.Targets()
	if err != nil {
		log.Errorf("mqtt discovery: failed to read switcher targets: %s", err)
		return
	}

	for _, target := range targets {
		id := sanitizeObjectId(target.Id)
		if id == "" {
			continue
		}

		button := &haEntity{
			Name:              target.Name,
			UniqueId:          fmt.Sprintf("%s_switch_%s", node, id),
			Device:            dev,
			AvailabilityTopic: avail,
			// The target id lives in the topic, so the button's fixed payload
			// does not have to carry it.
			CommandTopic: fmt.Sprintf("%s/switcher/%s/press", base, target.Id),
			PayloadPress: payloadPress,
			Icon:         "mdi:monitor-shimmer",
		}

		publishEntity(client, cfg, "button", "switch_"+id, button)
	}
}

func publishEntity(client paho.Client, cfg *Config, component string, objectId string, entity *haEntity) {
	payload, err := marshalEntity(entity)
	if err != nil {
		log.Errorf("mqtt discovery: %s", err)
		return
	}

	topic := configTopic(cfg, component, objectId)
	if token := client.Publish(topic, 1, true, payload); token.Wait() && token.Error() != nil {
		log.Errorf("mqtt discovery: failed to publish %s: %s", topic, token.Error())
	}
}

// sanitizeObjectId keeps discovery topics to the characters Home Assistant
// accepts in an object id.
func sanitizeObjectId(id string) string {
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

func readVersion() string {
	data, err := os.ReadFile(versionFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
