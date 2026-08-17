import { http } from '@/lib/http.ts';

export type ActionStepGPIO = {
  pin: number;
  // 'high', 'low', 'pulse' or 'toggle'
  mode: string;
  durationMs: number;
  // Relay boards usually activate when the pin is pulled low.
  activeLow: boolean;
  // Set when the firmware already drives this pin. Advisory only.
  reservedFor?: string;
};

export type ActionStepPWM = {
  chip: number;
  channel: number;
  periodNs: number;
  dutyPercent: number;
  enable: boolean;
};

export type ActionStepCommand = {
  command: string;
  background: boolean;
  timeoutSec: number;
};

export type ActionStep = {
  // 'gpio', 'pwm', 'command' or 'delay'
  type: string;
  gpio?: ActionStepGPIO;
  pwm?: ActionStepPWM;
  command?: ActionStepCommand;
  delayMs?: number;
};

export type ActionInfo = {
  id: string;
  name: string;
  steps: ActionStep[];
  showInMenu: boolean;
};

export type ActionsConfig = {
  actions: ActionInfo[];
  shortPress: string;
  doublePress: string;
  longPress: string;
  veryLongPress: string;
  keepDefaults: boolean;
  minGpio: number;
  maxGpio: number;
  longPressMs: number;
  veryLongPressMs: number;
};

export type GPIOState = {
  pin: number;
  value: number;
  // False when the pin has never been driven.
  known: boolean;
};

export function getActions() {
  return http.get('/api/actions');
}

export function setActions(config: {
  actions: ActionInfo[];
  shortPress: string;
  doublePress: string;
  longPress: string;
  veryLongPress: string;
  keepDefaults: boolean;
}) {
  return http.post('/api/actions', config);
}

export function runAction(id: string) {
  return http.post('/api/actions/run', { id });
}

export function getGpioState() {
  return http.get('/api/actions/gpio');
}
