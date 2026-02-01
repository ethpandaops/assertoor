import type { AuthTokenResponse, AuthState } from '../types/api';

const AUTH_STORAGE_KEY = 'assertoor_auth';
const AUTH_STATE_CHANGE_EVENT = 'assertoor_auth_state_change';
const REFRESH_BUFFER_MS = 60 * 1000; // Refresh 1 minute before expiration

interface StoredAuth {
  token: string;
  user: string;
  expiresAt: number;
}

type AuthStateListener = (state: AuthState) => void;

class AuthStore {
  private state: AuthState;
  private listeners: Set<AuthStateListener> = new Set();
  private refreshTimeoutId: number | null = null;
  private initialized = false;

  constructor() {
    this.state = {
      isLoggedIn: false,
      user: null,
      token: null,
      expiresAt: null,
    };

    // Listen for auth state changes from other components/roots
    window.addEventListener(AUTH_STATE_CHANGE_EVENT, this.handleExternalStateChange);
  }

  private handleExternalStateChange = (event: Event) => {
    const customEvent = event as CustomEvent<AuthState>;
    if (customEvent.detail) {
      this.state = customEvent.detail;
      this.notifyListeners();
    }
  };

  private getStoredAuth(): StoredAuth | null {
    try {
      const stored = sessionStorage.getItem(AUTH_STORAGE_KEY);
      if (!stored) return null;
      return JSON.parse(stored) as StoredAuth;
    } catch {
      return null;
    }
  }

  private setStoredAuth(auth: StoredAuth): void {
    sessionStorage.setItem(AUTH_STORAGE_KEY, JSON.stringify(auth));
  }

  private clearStoredAuth(): void {
    sessionStorage.removeItem(AUTH_STORAGE_KEY);
  }

  private notifyListeners(): void {
    this.listeners.forEach(listener => listener(this.state));
  }

  private broadcastStateChange(): void {
    window.dispatchEvent(new CustomEvent(AUTH_STATE_CHANGE_EVENT, { detail: this.state }));
  }

  private scheduleRefresh(expiresAt: number): void {
    if (this.refreshTimeoutId) {
      window.clearTimeout(this.refreshTimeoutId);
    }

    const timeUntilExpiry = expiresAt - Date.now();
    const refreshIn = Math.max(timeUntilExpiry - REFRESH_BUFFER_MS, 0);

    if (refreshIn > 0) {
      this.refreshTimeoutId = window.setTimeout(() => {
        this.fetchToken();
      }, refreshIn);
    }
  }

  async fetchToken(): Promise<AuthState> {
    try {
      const response = await fetch('/auth/token', {
        redirect: 'manual',  // Don't follow redirects
      });

      // Treat redirects (3xx) or opaque responses as not logged in
      if (response.type === 'opaqueredirect' || response.status >= 300 || !response.ok) {
        throw new Error('Not authenticated');
      }

      const data: AuthTokenResponse = await response.json();

      // Calculate local expiration time
      const serverExpr = parseInt(data.expr, 10);
      const serverNow = parseInt(data.now, 10);
      const validForSeconds = serverExpr - serverNow;
      const localExpiresAt = Date.now() + validForSeconds * 1000;

      // Store in session storage
      this.setStoredAuth({
        token: data.token,
        user: data.user,
        expiresAt: localExpiresAt,
      });

      // Having a valid token means the user is logged in (even if username is "unauthenticated")
      this.state = {
        isLoggedIn: true,
        user: data.user,
        token: data.token,
        expiresAt: localExpiresAt,
      };

      this.scheduleRefresh(localExpiresAt);
      this.notifyListeners();
      this.broadcastStateChange();

      return this.state;
    } catch (error) {
      console.error('Error fetching auth token:', error);
      this.clearStoredAuth();

      this.state = {
        isLoggedIn: false,
        user: null,
        token: null,
        expiresAt: null,
      };

      this.notifyListeners();
      this.broadcastStateChange();

      return this.state;
    }
  }

  async initialize(): Promise<void> {
    if (this.initialized) return;
    this.initialized = true;

    // Check if we have a valid stored token
    const stored = this.getStoredAuth();
    if (stored && stored.expiresAt > Date.now() + REFRESH_BUFFER_MS) {
      // Token is still valid - having a valid token means logged in
      this.state = {
        isLoggedIn: true,
        user: stored.user,
        token: stored.token,
        expiresAt: stored.expiresAt,
      };
      this.scheduleRefresh(stored.expiresAt);
      this.notifyListeners();
    } else {
      // Fetch new token
      await this.fetchToken();
    }
  }

  getState(): AuthState {
    return this.state;
  }

  getAuthHeader(): string | null {
    if (!this.state.token) return null;

    // Check if token is expired
    if (this.state.expiresAt && this.state.expiresAt < Date.now()) {
      return null;
    }

    return `Bearer ${this.state.token}`;
  }

  subscribe(listener: AuthStateListener): () => void {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  }

  login(): void {
    window.location.href = '/auth/login';
  }

  destroy(): void {
    if (this.refreshTimeoutId) {
      window.clearTimeout(this.refreshTimeoutId);
    }
    window.removeEventListener(AUTH_STATE_CHANGE_EVENT, this.handleExternalStateChange);
    this.listeners.clear();
  }
}

// Singleton instance
export const authStore = new AuthStore();
