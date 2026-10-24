import { api } from './client';

export function registerPushToken(token: string) {
  return api.post<void>('/notifications/register-token', { token });
}
