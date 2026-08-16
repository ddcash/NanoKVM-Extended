import { http } from '@/lib/http.ts';

export type MqttCommand = {
  name: string;
  topic: string;
  payload: string;
};

export type MqttConfig = {
  enabled: boolean;
  broker: string;
  port: number;
  tls: boolean;
  username: string;
  topic: string;
  commands: MqttCommand[];
  hasPassword: boolean;
};

// The stored password is never sent to the client; omitting it here keeps it
// unchanged on the device, while an empty string clears it.
export type MqttConfigUpdate = Omit<MqttConfig, 'hasPassword'> & {
  password?: string;
};

export function getMqttConfig() {
  return http.get('/api/mqtt/config');
}

export function setMqttConfig(config: MqttConfigUpdate) {
  return http.post('/api/mqtt/config', config);
}

export function publishMqttCommand(name: string) {
  return http.post('/api/mqtt/publish', { name });
}
