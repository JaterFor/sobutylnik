import { api } from './client';
import { Profile } from './types';

export function getProfile() {
  return api.get<Profile>('/profile/');
}

export interface UpsertProfileInput {
  name: string;
  age: number;
  gender: 'male' | 'female' | 'other';
  bio: string;
  preferred_drink?: string;
  readiness_status: 'now' | 'today' | 'this_week';
}

export function upsertProfile(input: UpsertProfileInput) {
  return api.put<Profile>('/profile/', input);
}

export function updateLocation(latitude: number, longitude: number) {
  return api.put<void>('/profile/location', { latitude, longitude });
}

export function deletePhoto(photoId: string) {
  return api.delete<void>(`/profile/photos/${photoId}`);
}

// uploadPhoto: ask backend for a presigned R2 URL, PUT the file bytes
// directly to R2, then tell backend the upload is confirmed.
export async function uploadPhoto(localUri: string, contentType: string): Promise<void> {
  const { upload_url, object_key } = await api.post<{ upload_url: string; object_key: string }>(
    '/profile/photos/presign',
    { content_type: contentType }
  );

  const fileResp = await fetch(localUri);
  const blob = await fileResp.blob();

  const putResp = await fetch(upload_url, {
    method: 'PUT',
    headers: { 'Content-Type': contentType },
    body: blob,
  });
  if (!putResp.ok) {
    throw new Error('Не удалось загрузить фото в хранилище');
  }

  await api.post<void>('/profile/photos/confirm', { object_key });
}

