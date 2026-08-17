import { http } from '@/lib/http';

export function login(username: string, password: string, code?: string) {
  const data = {
    username,
    password,
    ...(code ? { code } : {})
  };
  return http.post('/api/auth/login', data);
}

// two-factor authentication
export function getTotp() {
  return http.get('/api/auth/totp');
}

export function setupTotp() {
  return http.post('/api/auth/totp/setup');
}

export function enableTotp(code: string) {
  return http.post('/api/auth/totp/enable', { code });
}

export function disableTotp(password: string, code: string) {
  return http.post('/api/auth/totp/disable', { password, code });
}

export function logout() {
  return http.post('/api/auth/logout');
}

export function getAccount() {
  return http.get('/api/auth/account');
}

export function changePassword(username: string, password: string) {
  const data = {
    username,
    password
  };
  return http.post('/api/auth/password', data);
}

export function isPasswordUpdated() {
  return http.get('/api/auth/password');
}

export type SessionInfo = {
  id: string;
  username: string;
  ip: string;
  userAgent: string;
  createdAt: number;
  lastSeen: number;
  current: boolean;
};

export function getSessions() {
  return http.get('/api/auth/sessions');
}

// Pass all to end every session except this one.
export function revokeSession(payload: { id?: string; all?: boolean }) {
  return http.post('/api/auth/sessions/revoke', payload);
}
