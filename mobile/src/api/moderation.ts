import { api } from './client';

export function reportUser(userId: string, reason: string) {
  return api.post<void>('/moderation/report', { user_id: userId, reason });
}

export function blockUser(userId: string) {
  return api.post<void>('/moderation/block', { user_id: userId });
}
