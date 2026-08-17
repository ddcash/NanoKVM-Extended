import { http } from '@/lib/http.ts';

export type ActionGPIO = {
  pin: number;
  // 'high', 'low' or 'pulse'
  mode: string;
  durationMs: number;
  // Relay boards usually activate when the pin is pulled low.
  activeLow: boolean;
  // Set when the firmware already drives this pin. Advisory only.
  reservedFor?: string;
};

export type ActionInfo = {
  id: string;
  name: string;
  // 'gpio' or 'command'
  type: string;
  gpio?: ActionGPIO;
  command?: string;
  showInMenu: boolean;
};

export type ActionsConfig = {
  actions: ActionInfo[];
  shortPress: string;
  longPress: string;
  keepDefaults: boolean;
  minGpio: number;
  maxGpio: number;
};

export function getActions() {
  return http.get('/api/actions');
}

export function setActions(config: {
  actions: ActionInfo[];
  shortPress: string;
  longPress: string;
  keepDefaults: boolean;
}) {
  return http.post('/api/actions', config);
}

export function runAction(id: string) {
  return http.post('/api/actions/run', { id });
}
