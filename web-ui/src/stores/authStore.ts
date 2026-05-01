import type { AuthState } from '../types/api';

const AUTH_STATE_CHANGE_EVENT = 'assertoor_auth_state_change';
const RUNTIME_CONFIG_URL = '/api/v1/runtime-config';

type AuthStateListener = (state: AuthState) => void;

interface RuntimeConfig {
  authProviderURL?: string;
}

// AuthClientInfo is the shape returned by ethpandaops.authenticatoor's
// checkLogin() / state-change events. We model it loosely so future
// client additions (e.g. groups) flow through without changes here.
interface AuthClientInfo {
  authenticated: boolean;
  token?: string;
  user?: string;
  email?: string;
  exp?: number; // Unix seconds
}

interface AuthClient {
  checkLogin: () => Promise<AuthClientInfo>;
  login: () => void;
  getToken: () => string | null;
  isLoggedIn: () => boolean;
  onStateChange?: (cb: (info: AuthClientInfo) => void) => void;
}

declare global {
  interface Window {
    ethpandaops?: { authenticatoor?: AuthClient };
  }
}

class AuthStore {
  private state: AuthState;
  private listeners: Set<AuthStateListener> = new Set();
  private initialized = false;
  private initPromise: Promise<void> | null = null;
  private openMode = false;
  private client: AuthClient | null = null;

  constructor() {
    this.state = {
      isLoggedIn: false,
      user: null,
      token: null,
      expiresAt: null,
    };

    window.addEventListener(AUTH_STATE_CHANGE_EVENT, this.handleExternalStateChange);
  }

  private handleExternalStateChange = (event: Event) => {
    const customEvent = event as CustomEvent<AuthState>;
    if (customEvent.detail) {
      this.state = customEvent.detail;
      this.notifyListeners();
    }
  };

  private notifyListeners(): void {
    this.listeners.forEach((listener) => listener(this.state));
  }

  private broadcastStateChange(): void {
    window.dispatchEvent(
      new CustomEvent(AUTH_STATE_CHANGE_EVENT, { detail: this.state })
    );
  }

  // applyClientInfo mirrors the auth client's state into this store.
  private applyClientInfo(info: AuthClientInfo): void {
    this.state = {
      isLoggedIn: !!info.authenticated,
      user: info.user || info.email || null,
      token: info.token || null,
      expiresAt: info.exp ? info.exp * 1000 : null,
    };
    this.notifyListeners();
    this.broadcastStateChange();
  }

  // loadAuthClient injects the auth provider's client.js script and
  // resolves once window.ethpandaops.authenticatoor is available.
  private loadAuthClient(authProviderURL: string): Promise<AuthClient> {
    if (window.ethpandaops?.authenticatoor) {
      return Promise.resolve(window.ethpandaops.authenticatoor);
    }
    return new Promise((resolve, reject) => {
      const script = document.createElement('script');
      script.src = authProviderURL.replace(/\/$/, '') + '/client.js';
      script.async = true;
      script.onload = () => {
        const c = window.ethpandaops?.authenticatoor;
        if (c) {
          resolve(c);
        } else {
          reject(new Error('auth client.js loaded but ethpandaops.authenticatoor is missing'));
        }
      };
      script.onerror = () => reject(new Error('failed to load auth client.js'));
      document.head.appendChild(script);
    });
  }

  // initialize fetches the runtime config and either marks open mode (no
  // auth provider configured) or loads the auth provider's client.js and
  // runs checkLogin. Safe to call multiple times — second+ calls return
  // the in-flight promise.
  async initialize(): Promise<void> {
    if (this.initialized) return;
    if (this.initPromise) return this.initPromise;

    this.initPromise = (async () => {
      let cfg: RuntimeConfig = {};
      try {
        const r = await fetch(RUNTIME_CONFIG_URL, { credentials: 'same-origin' });
        if (r.ok) cfg = await r.json();
      } catch {
        // Treat fetch failures as open mode — the backend will reject
        // protected requests anyway if it actually requires auth.
      }

      if (!cfg.authProviderURL) {
        this.openMode = true;
        this.state = {
          isLoggedIn: true,
          user: null,
          token: null,
          expiresAt: null,
        };
        this.notifyListeners();
        this.initialized = true;
        return;
      }

      try {
        this.client = await this.loadAuthClient(cfg.authProviderURL);
        if (this.client.onStateChange) {
          this.client.onStateChange((info) => this.applyClientInfo(info));
        }
        const info = await this.client.checkLogin();
        if (info) this.applyClientInfo(info);
      } catch (err) {
        console.error('auth client bootstrap failed:', err);
      } finally {
        this.initialized = true;
      }
    })();

    return this.initPromise;
  }

  getState(): AuthState {
    return this.state;
  }

  // getAuthHeader returns the value to put in the Authorization header,
  // or null when no token is available (open mode or not logged in).
  getAuthHeader(): string | null {
    if (this.openMode) return null;
    if (this.client) {
      const t = this.client.getToken();
      if (t) return `Bearer ${t}`;
    }
    return null;
  }

  subscribe(listener: AuthStateListener): () => void {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  }

  // login starts the upstream auth flow. In open mode this is a no-op.
  login(): void {
    if (this.client) this.client.login();
  }

  // fetchToken re-runs the auth client's checkLogin (e.g. after a 401)
  // and resolves with the resulting AuthState.
  async fetchToken(): Promise<AuthState> {
    if (this.openMode || !this.client) return this.state;
    try {
      const info = await this.client.checkLogin();
      if (info) this.applyClientInfo(info);
    } catch (err) {
      console.error('auth re-check failed:', err);
    }
    return this.state;
  }

  destroy(): void {
    window.removeEventListener(AUTH_STATE_CHANGE_EVENT, this.handleExternalStateChange);
    this.listeners.clear();
  }
}

// Singleton instance
export const authStore = new AuthStore();
