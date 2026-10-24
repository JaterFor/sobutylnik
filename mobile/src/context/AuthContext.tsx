import React, { createContext, useContext, useEffect, useState, useCallback } from 'react';
import { api, setUnauthorizedHandler } from '../api/client';
import { saveTokens, clearTokens, getAccessToken } from '../storage/tokens';
import { TokenPair } from '../api/types';

interface AuthContextValue {
  isAuthenticated: boolean;
  isLoading: boolean;
  loginWithEmail: (email: string, password: string) => Promise<void>;
  registerWithEmail: (email: string, password: string) => Promise<void>;
  loginWithTelegram: (payload: Record<string, unknown>) => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    getAccessToken().then((token) => {
      setIsAuthenticated(!!token);
      setIsLoading(false);
    });
    setUnauthorizedHandler(() => setIsAuthenticated(false));
  }, []);

  const handleTokens = useCallback(async (tokens: TokenPair) => {
    await saveTokens(tokens.access_token, tokens.refresh_token);
    setIsAuthenticated(true);
  }, []);

  const loginWithEmail = useCallback(
    async (email: string, password: string) => {
      const tokens = await api.post<TokenPair>('/auth/login', { email, password }, false);
      await handleTokens(tokens);
    },
    [handleTokens]
  );

  const registerWithEmail = useCallback(
    async (email: string, password: string) => {
      const tokens = await api.post<TokenPair>('/auth/register', { email, password }, false);
      await handleTokens(tokens);
    },
    [handleTokens]
  );

  const loginWithTelegram = useCallback(
    async (payload: Record<string, unknown>) => {
      const tokens = await api.post<TokenPair>('/auth/telegram', payload, false);
      await handleTokens(tokens);
    },
    [handleTokens]
  );

  const logout = useCallback(async () => {
    await clearTokens();
    setIsAuthenticated(false);
  }, []);

  return (
    <AuthContext.Provider
      value={{ isAuthenticated, isLoading, loginWithEmail, registerWithEmail, loginWithTelegram, logout }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}
