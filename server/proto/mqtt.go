package proto

type MqttCommand struct {
	Name    string `json:"name"`
	Topic   string `json:"topic"`
	Payload string `json:"payload"`
}

type GetMqttConfigRsp struct {
	Enabled  bool          `json:"enabled"`
	Broker   string        `json:"broker"`
	Port     int           `json:"port"`
	Tls      bool          `json:"tls"`
	Username string        `json:"username"`
	Topic    string        `json:"topic"`
	Commands []MqttCommand `json:"commands"`
	// The password itself is never returned.
	HasPassword bool `json:"hasPassword"`
}

type SetMqttConfigReq struct {
	Enabled  *bool  `json:"enabled"`
	Broker   string `json:"broker"`
	Port     int    `json:"port"`
	Tls      *bool  `json:"tls"`
	Username string `json:"username"`
	Topic    string `json:"topic"`
	// Omitted leaves the stored password unchanged; an empty string clears it.
	Password *string       `json:"password"`
	Commands []MqttCommand `json:"commands"`
}

type PublishMqttReq struct {
	Name string `json:"name" form:"name" validate:"required"`
}
