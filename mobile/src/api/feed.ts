import { api } from './client';
import { Candidate, SwipeResult } from './types';

export interface FeedFilter {
  radiusKm?: number;
  gender?: string;
  readiness?: string;
}

export function getFeed(filter: FeedFilter = {}) {
  const params = new URLSearchParams();
  if (filter.radiusKm) params.set('radius_km', String(filter.radiusKm));
  if (filter.gender) params.set('gender', filter.gender);
  if (filter.readiness) params.set('readiness', filter.readiness);
  const query = params.toString();
  return api.get<Candidate[]>(`/feed/${query ? `?${query}` : ''}`);
}

export function swipe(targetUserId: string, liked: boolean) {
  return api.post<SwipeResult>('/feed/swipe', { target_user_id: targetUserId, liked });
}
