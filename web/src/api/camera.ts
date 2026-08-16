import { http } from '@/lib/http.ts';

export type CameraConfig = {
  enabled: boolean;
  token: string;
};

export function getCamera() {
  return http.get('/api/camera');
}

// Enabling mints a fresh token, so toggling off and on invalidates any URL
// handed out previously.
export function setCamera(enabled: boolean) {
  return http.post('/api/camera', { enabled });
}
