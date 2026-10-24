import { API_BASE_URL } from './config';
import { getAccessToken, getRefreshToken, saveAccessToken, clearTokens } from '../storage/tokens';

export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

let onUnauthorized: (() => void) | null = null;
export function setUnauthorizedHandler(handler: () => void) {
  onUnauthorized = handler;
}

async function rawRequest(path: string, options: RequestInit, token: string | null): Promise<Response> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string> | undefined),
  };
  if (token) headers.Authorization = `Bearer ${token}`;

  return fetch(`${API_BASE_URL}${path}`, { ...options, headers });
}

async function tryRefresh(): Promise<string | null> {
  const refreshToken = await getRefreshToken();
  if (!refreshToken) return null;

  const resp = await fetch(`${API_BASE_URL}/auth/refresh`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ refresh_token: refreshToken }),
  });
  if (!resp.ok) return null;

  const data = await resp.json();
  await saveAccessToken(data.access_token);
  return data.access_token as string;
}

// request performs an authenticated call, transparently refreshing the
// access token once on a 401 before giving up and signalling logout.
export async function request<T>(path: string, options: RequestInit = {}, auth = true): Promise<T> {
  let token = auth ? await getAccessToken() : null;
  let resp = await rawRequest(path, options, token);

  if (resp.status === 401 && auth) {
    const refreshed = await tryRefresh();
    if (refreshed) {
      resp = await rawRequest(path, options, refreshed);
    } else {
      await clearTokens();
      onUnauthorized?.();
    }
  }

  const text = await resp.text();
  const data = text ? JSON.parse(text) : undefined;

  if (!resp.ok) {
    const message = (data && data.error) || `Request failed with status ${resp.status}`;
    throw new ApiError(resp.status, message);
  }
  return data as T;
}

export const api = {
  get: <T>(path: string) => request<T>(path, { method: 'GET' }),
  post: <T>(path: string, body?: unknown, auth = true) =>
    request<T>(path, { method: 'POST', body: body !== undefined ? JSON.stringify(body) : undefined }, auth),
  put: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: 'PUT', body: body !== undefined ? JSON.stringify(body) : undefined }),
  delete: <T>(path: string) => request<T>(path, { method: 'DELETE' }),
};
