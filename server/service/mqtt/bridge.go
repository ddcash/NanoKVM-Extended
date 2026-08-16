package mqtt

import (
	"fmt"
	"strings"
	"sync"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"

	"NanoKVM-Server/common"
	"NanoKVM-Server/service/switcher"
)

// The bridge keeps one connection open, unlike the fire-and-forget publish used
// for user-defined commands. Home Assistant needs a live session: a last-will
// message so entities go unavailable when this device drops off, a subscription
// so buttons can be pressed, and periodic state.
const (
	stateInterval    = 30 * time.Second
	bridgeConnectTO  = 10 * time.Second
	payloadOnline    = "online"
	payloadOffline   = "offline"
	payloadPress     = "PRESS"
	versionFile      = "/kvmapp/version"
	reconnectMaxWait = 2 * time.Minute
)

var (
	bridgeMu     sync.Mutex
	bridgeClient paho.Client
	bridgeStop   chan struct{}
)

// StartBridge brings the Home Assistant connection up, replacing any previous
// one. Safe to call repeatedly; the settings UI calls it after every save.
func StartBridge() {
	bridgeMu.Lock()
	defer bridgeMu.Unlock()

	stopBridgeLocked()

	cfg, err := loadConfig()
	if err != nil {
		log.Errorf("mqtt bridge: failed to load config: %s", err)
		return
	}
	if !cfg.Enabled || !cfg.HaEnabled || cfg.Broker == "" {
		return
	}

	scheme := "tcp"
	if cfg.Tls {
		scheme = "ssl"
	}

	opts := paho.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("%s://%s:%d", scheme, cfg.Broker, cfg.Port))
	opts.SetClientID(fmt.Sprintf("nanokvm-%s", nodeId(&cfg)))
	opts.SetConnectTimeout(bridgeConnectTO)
	// Reconnect rather than give up: the broker may well outlive or restart
	// independently of this device.
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(reconnectMaxWait)
	opts.SetCleanSession(true)
	if cfg.Username != "" {
		opts.SetUsername(cfg.Username)
		opts.SetPassword(cfg.Password)
	}

	// Retained, so Home Assistant sees the device as unavailable even if it
	// subscribes after this device has already dropped off.
	opts.SetWill(availabilityTopic(&cfg), payloadOffline, 1, true)

	conf := cfg
	opts.SetOnConnectHandler(func(client paho.Client) {
		log.Infof("mqtt bridge: connected to %s", conf.Broker)
		onBridgeConnected(client, &conf)
	})
	opts.SetConnectionLostHandler(func(_ paho.Client, err error) {
		log.Warnf("mqtt bridge: connection lost: %s", err)
	})

	client := paho.NewClient(opts)
	token := client.Connect()

	stop := make(chan struct{})
	bridgeClient = client
	bridgeStop = stop

	go func() {
		if !token.WaitTimeout(bridgeConnectTO) || token.Error() != nil {
			log.Errorf("mqtt bridge: initial connect failed: %v", token.Error())
			// AutoReconnect keeps retrying, so this is not fatal.
		}
		publishStateLoop(client, &conf, stop)
	}()
}

// StopBridge tears the connection down, marking the device offline first so
// Home Assistant does not have to wait for the will.
func StopBridge() {
	bridgeMu.Lock()
	defer bridgeMu.Unlock()
	stopBridgeLocked()
}

func stopBridgeLocked() {
	if bridgeStop != nil {
		close(bridgeStop)
		bridgeStop = nil
	}
	if bridgeClient != nil {
		if bridgeClient.IsConnected() {
			cfg, err := loadConfig()
			if err == nil {
				token := bridgeClient.Publish(availabilityTopic(&cfg), 1, true, payloadOffline)
				token.WaitTimeout(2 * time.Second)
			}
		}
		bridgeClient.Disconnect(250)
		bridgeClient = nil
	}
}

func onBridgeConnected(client paho.Client, cfg *Config) {
	publishDiscovery(client, cfg)

	// Buttons all arrive on one wildcard subscription rather than one per
	// target, so adding a target does not require re-subscribing.
	commandTopic := baseTopic(cfg) + "/switcher/+/press"
	if token := client.Subscribe(commandTopic, 1, handleSwitcherCommand); token.Wait() && token.Error() != nil {
		log.Errorf("mqtt bridge: subscribe failed: %s", token.Error())
	}

	client.Publish(availabilityTopic(cfg), 1, true, payloadOnline)
	publishState(client, cfg)
}

// handleSwitcherCommand replays the target named in the topic. The id is taken
// from the topic rather than the payload so Home Assistant's button entity,
// which sends a fixed payload, maps onto it directly.
func handleSwitcherCommand(_ paho.Client, msg paho.Message) {
	parts := strings.Split(msg.Topic(), "/")
	if len(parts) < 2 {
		return
	}
	id := parts[len(parts)-2]

	if err := switcher.Press(id); err != nil {
		log.Errorf("mqtt bridge: failed to press %q: %s", id, err)
		return
	}
	log.Debugf("mqtt bridge: pressed switcher target %s", id)
}

func publishStateLoop(client paho.Client, cfg *Config, stop <-chan struct{}) {
	ticker := time.NewTicker(stateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			if client.IsConnected() {
				publishState(client, cfg)
			}
		}
	}
}

func publishState(client paho.Client, cfg *Config) {
	base := baseTopic(cfg)

	hdmi := payloadOffline
	if common.GetKvmVision().HasHDMISignal() {
		hdmi = payloadOnline
	}
	client.Publish(base+"/hdmi", 0, true, hdmi)
}
