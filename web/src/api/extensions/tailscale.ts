import { http } from '@/lib/http.ts';

// install tailscale
export function install() {
  return http.post('/api/extensions/tailscale/install');
}

// uninstall tailscale
export function uninstall() {
  return http.post('/api/extensions/tailscale/uninstall');
}

// get tailscale status
export function getStatus() {
  return http.get('/api/extensions/tailscale/status');
}

// start tailscale
export function start() {
  return http.post('/api/extensions/tailscale/start');
}

// restart tailscale
export function restart() {
  return http.post('/api/extensions/tailscale/restart');
}

// stop tailscale
export function stop() {
  return http.post('/api/extensions/tailscale/stop');
}

// run tailscale up
export function up() {
  return http.post('/api/extensions/tailscale/up');
}

// run tailscale down
export function down() {
  return http.post('/api/extensions/tailscale/down');
}

// login tailscale
export function login() {
  return http.post('/api/extensions/tailscale/login');
}

// logout tailscale
export function logout() {
  return http.post('/api/extensions/tailscale/logout');
}

export type TailscaleVersion = {
  installed: string;
  latest: string;
  updateAvailable: boolean;
};

export function getVersion() {
  return http.get('/api/tailscale/version');
}

// Reinstalls at the latest version. Login state lives outside the binaries, so
// the node stays connected.
export function update() {
  return http.post('/api/tailscale/update');
}
