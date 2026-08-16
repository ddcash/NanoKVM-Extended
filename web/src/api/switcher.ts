import { http } from '@/lib/http.ts';

export type SwitcherKey = {
  code: string;
  label: string;
};

// Keys within a step are held together, so a step covers both a single tap and
// a chord such as Ctrl+Alt+1.
export type SwitcherStep = {
  keys: SwitcherKey[];
};

export type SwitcherTarget = {
  id: string;
  name: string;
  steps: SwitcherStep[];
};

export type SwitcherConfig = {
  targets: SwitcherTarget[];
  stepDelayMs: number;
};

export function getSwitcher() {
  return http.get('/api/switcher');
}

export function setSwitcher(config: SwitcherConfig) {
  return http.post('/api/switcher', config);
}
