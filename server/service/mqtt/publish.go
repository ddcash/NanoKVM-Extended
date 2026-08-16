package mqtt

import (
	"crypto/tls"
	"fmt"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"NanoKVM-Server/proto"
)

const (
	connectTimeout = 5 * time.Second
	publishTimeout = 5 * time.Second
	disconnectMs   = 250
)

// Publish sends one configured command to the broker. Switching a KVM input is
// a rare, user-initiated action, so the client connects and disconnects per
// call rather than holding an idle connection open on a 256MB device.
func (s *Service) Publish(c *gin.Context) {
	var req proto.PublishMqttReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	cfg, err := loadConfig()
	if err != nil {
		log.Errorf("failed to load mqtt config: %s", err)
		rsp.ErrRsp(c, -2, "failed to load mqtt config")
		return
	}
	if !cfg.Enabled {
		rsp.ErrRsp(c, -3, "mqtt is not enabled")
		return
	}

	var command *proto.MqttCommand
	for i := range cfg.Commands {
		if cfg.Commands[i].Name == req.Name {
			command = &cfg.Commands[i]
			break
		}
	}
	if command == nil {
		rsp.ErrRsp(c, -4, "unknown command")
		return
	}

	topic := command.Topic
	if topic == "" {
		topic = cfg.Topic
	}

	if err := publish(&cfg, topic, command.Payload); err != nil {
		log.Errorf("mqtt publish failed: %s", err)
		rsp.ErrRsp(c, -5, "failed to publish")
		return
	}

	log.Debugf("mqtt published command %s to %s", command.Name, topic)
	rsp.OkRsp(c)
}

func publish(cfg *Config, topic string, payload string) error {
	scheme := "tcp"
	if cfg.Tls {
		scheme = "ssl"
	}

	opts := paho.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("%s://%s:%d", scheme, cfg.Broker, cfg.Port))
	opts.SetClientID(fmt.Sprintf("nanokvm-%d", time.Now().UnixNano()))
	opts.SetConnectTimeout(connectTimeout)
	opts.SetConnectRetry(false)
	opts.SetAutoReconnect(false)
	if cfg.Username != "" {
		opts.SetUsername(cfg.Username)
		opts.SetPassword(cfg.Password)
	}

	client := paho.NewClient(opts)

	token := client.Connect()
	if !token.WaitTimeout(connectTimeout) {
		return fmt.Errorf("connect to %s timed out", cfg.Broker)
	}
	if err := token.Error(); err != nil {
		return fmt.Errorf("connect to %s: %w", cfg.Broker, err)
	}
	defer client.Disconnect(disconnectMs)

	pubToken := client.Publish(topic, 0, false, payload)
	if !pubToken.WaitTimeout(publishTimeout) {
		return fmt.Errorf("publish to %s timed out", topic)
	}
	if err := pubToken.Error(); err != nil {
		return fmt.Errorf("publish to %s: %w", topic, err)
	}

	return nil
}
