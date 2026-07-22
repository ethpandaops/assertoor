// Runtime config injected by the Go backend into <head> of index.html
// (see pkg/web/handlers/spa.go injectHead). Exposed at boot via
// window.ethpandaops.assertoor.config so the SPA can read it
// synchronously without a roundtrip.
export interface RuntimeConfig {
  authProviderURL: string;
}

declare global {
  interface Window {
    ethpandaops?: {
      authenticatoor?: AuthenticatoorLib;
      assertoor?: {
        config?: Partial<RuntimeConfig>;
      };
    };
  }
}

// TokenInfo pushed by the v2 authenticatoor client on every session
// change ("status" events) and returned by getStatus().
export interface AuthTokenInfo {
  status: 'unauthenticated' | 'authenticated' | 'refreshing';
  authenticated: boolean;
  user: string;
  exp: number;
}

// Minimal shape of the v2 window.ethpandaops.authenticatoor we rely on.
// Set by the authenticatoor's client.js?v=2 script (loaded at runtime
// when an auth provider is configured). The client owns the session —
// shared across all ethpandaops apps via a hidden iframe — and refreshes
// tokens before expiry; getToken() always resolves to a fresh token.
export interface AuthenticatoorLib {
  version?: number;
  addEventListener: (type: 'status', cb: (info: AuthTokenInfo) => void) => void;
  removeEventListener: (type: 'status', cb: (info: AuthTokenInfo) => void) => void;
  getStatus: () => Promise<AuthTokenInfo>;
  getToken: () => Promise<string | null>;
  login: () => Promise<boolean>;
  logout: () => Promise<void>;
  authServiceURL: () => string;
}

export function getRuntimeConfig(): RuntimeConfig {
  const cfg = (window.ethpandaops?.assertoor?.config ?? {}) as Partial<RuntimeConfig>;
  return {
    authProviderURL: cfg.authProviderURL ?? '',
  };
}
