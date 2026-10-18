import { useEffect, useState } from 'react';
import { useAuthContext } from '../context/AuthContext';
import { authStore } from '../stores/authStore';

// The server's /debug/pprof pages need a login. Tools send the bearer
// token; a browser gets this page instead, which hands the session's token
// to the server as a short-lived cookie scoped to /debug/pprof and reloads.
// Not in the nav: the server serves the SPA on this path only to a browser
// without the cookie. Must match debugTokenCookie in pkg/web/pprof.go.
const DEBUG_TOKEN_COOKIE = 'assertoor_debug_token';
const DEBUG_PATH = '/debug/pprof';
// The token tried last (its signature tail, never the token) and when: a
// token the server turned down right after its cookie was set must not
// reload the page in a loop. An older attempt is a cookie that expired
// before its token, which is simply set again.
const ATTEMPT_KEY = 'assertoor_debug_token_attempt';
const REFUSED_WITHIN_MS = 15_000;
// The cookie never outlives the token, nor this cap.
const MAX_COOKIE_AGE_SECONDS = 30 * 60;

function readAttempt(): string | null {
  try {
    return sessionStorage.getItem(ATTEMPT_KEY);
  } catch {
    return null;
  }
}

function writeAttempt(value: string) {
  try {
    sessionStorage.setItem(ATTEMPT_KEY, value);
  } catch {
    // Without storage a refused token shows the error after one reload.
  }
}

function DebugLogin() {
  const { authEnabled, isLoggedIn, loading, expiresAt } = useAuthContext();
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (loading || !isLoggedIn) return;

    let cancelled = false;

    void (async () => {
      const token = await authStore.getAuthToken();
      if (cancelled) return;

      if (!token) {
        setError(
          authEnabled
            ? 'Your session did not provide a login token.'
            : 'This server needs no login, but did not serve the page; try reloading.'
        );
        return;
      }

      const tail = token.slice(-16);
      const [lastTail, lastAt] = (readAttempt() ?? '').split(':');
      if (lastTail === tail && Date.now() - Number(lastAt) < REFUSED_WITHIN_MS) {
        setError('The server did not accept your login for the profiling pages.');
        return;
      }
      writeAttempt(`${tail}:${Date.now()}`);

      const remaining = expiresAt
        ? Math.floor((expiresAt - Date.now()) / 1000)
        : MAX_COOKIE_AGE_SECONDS;
      const maxAge = Math.max(1, Math.min(remaining, MAX_COOKIE_AGE_SECONDS));
      const secure = window.location.protocol === 'https:' ? '; Secure' : '';
      document.cookie = `${DEBUG_TOKEN_COOKIE}=${token}; Path=${DEBUG_PATH}; Max-Age=${maxAge}; SameSite=Strict${secure}`;
      // The index is the path with the trailing slash; the bare path is
      // the SPA's own route and would show this page again.
      if (window.location.pathname === DEBUG_PATH) {
        window.location.replace(`${DEBUG_PATH}/${window.location.search}`);
      } else {
        window.location.reload();
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [authEnabled, isLoggedIn, loading, expiresAt]);

  return (
    <div className="card mx-auto max-w-xl p-6">
      <h1 className="text-xl font-bold">Profiling</h1>
      {error ? (
        <p className="mt-4 text-sm text-error-600">{error}</p>
      ) : loading || isLoggedIn ? (
        <div className="mt-4 flex items-center gap-3 text-sm text-[var(--color-text-secondary)]">
          <div className="animate-spin rounded-full size-4 border-b-2 border-primary-600"></div>
          <span>Opening the profiling pages…</span>
        </div>
      ) : (
        <p className="mt-4 text-sm text-[var(--color-text-secondary)]">
          The profiling pages need a login. Log in with the button in the header; this page
          continues on its own afterwards.
        </p>
      )}
    </div>
  );
}

export default DebugLogin;
