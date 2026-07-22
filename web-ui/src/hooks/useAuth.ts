import { useState, useEffect, useCallback } from 'react';
import { authStore } from '../stores/authStore';
import type { AuthState } from '../types/api';

export function useAuth() {
  const [authState, setAuthState] = useState<AuthState>(authStore.getState());
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // Subscribe to auth state changes
    const unsubscribe = authStore.subscribe((newState) => {
      setAuthState(newState);
      setLoading(false);
    });

    // Initialize the auth store
    authStore.initialize().then(() => {
      setAuthState(authStore.getState());
      setLoading(false);
    });

    return unsubscribe;
  }, []);

  const getAuthHeader = useCallback((): Promise<string | null> => {
    return authStore.getAuthHeader();
  }, []);

  const login = useCallback(() => {
    authStore.login();
  }, []);

  const logout = useCallback(() => {
    authStore.logout();
  }, []);

  const refreshToken = useCallback(async () => {
    await authStore.refreshToken();
  }, []);

  return {
    ...authState,
    loading,
    getAuthHeader,
    login,
    logout,
    refreshToken,
  };
}
