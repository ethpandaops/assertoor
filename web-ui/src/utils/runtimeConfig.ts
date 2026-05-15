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

// Minimal shape of window.ethpandaops.authenticatoor we rely on. Set by
// the authenticatoor's client.js script (loaded at runtime when an auth
// provider is configured).
export interface AuthenticatoorLib {
  checkLogin: () => Promise<{
    authenticated: boolean;
    token: string;
    exp: number;
    user: string;
  }>;
  login: () => void;
  logout: () => void;
  getToken: () => string | null;
  isLoggedIn: () => boolean;
  authServiceURL: () => string;
}

export function getRuntimeConfig(): RuntimeConfig {
  const cfg = (window.ethpandaops?.assertoor?.config ?? {}) as Partial<RuntimeConfig>;
  return {
    authProviderURL: cfg.authProviderURL ?? '',
  };
}
