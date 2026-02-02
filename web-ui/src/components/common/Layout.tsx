import { Outlet, Link, useLocation, useMatch, matchPath } from 'react-router-dom';
import { useTheme } from '../../hooks/useTheme';
import { UserDisplay } from '../auth/UserDisplay';

const navItems = [
  { path: '/', label: 'Dashboard' },
  { path: '/registry', label: 'Registry' },
  { path: '/builder', label: 'Builder' },
  { path: '/clients', label: 'Clients' },
];

function Layout() {
  const location = useLocation();
  const { theme, toggleTheme } = useTheme();
  const isTestRunPage = useMatch('/run/:runId');
  const isBuilderPage = matchPath('/builder', location.pathname);

  return (
    <div className="min-h-screen flex flex-col">
      {/* Header */}
      <header className="bg-[var(--color-bg-secondary)] border-b border-[var(--color-border)] sticky top-0 z-50">
        <div className="max-w-screen-2xl mx-auto px-3 sm:px-4 lg:px-6">
          <div className="flex items-center justify-between h-14">
            {/* Logo */}
            <Link to="/" className="flex items-center space-x-2">
              <div className="w-7 h-7 bg-primary-600 rounded-md flex items-center justify-center">
                <span className="text-white font-bold text-sm">A</span>
              </div>
              <span className="text-lg font-semibold">Assertoor</span>
            </Link>

            {/* Navigation */}
            <nav className="hidden md:flex items-center space-x-1">
              {navItems.map((item) => (
                <Link
                  key={item.path}
                  to={item.path}
                  className={`px-3 py-1.5 rounded-md text-sm font-medium transition-colors ${
                    location.pathname === item.path
                      ? 'bg-primary-600 text-white'
                      : 'text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-tertiary)]'
                  }`}
                >
                  {item.label}
                </Link>
              ))}
            </nav>

            {/* Right side controls */}
            <div className="flex items-center gap-4">
              {/* User display */}
              <UserDisplay />

              {/* Theme toggle */}
              <button
                onClick={toggleTheme}
                className="p-2 rounded-md hover:bg-[var(--color-bg-tertiary)] transition-colors"
                aria-label="Toggle theme"
              >
                {theme === 'dark' ? (
                  <SunIcon className="w-5 h-5" />
                ) : (
                  <MoonIcon className="w-5 h-5" />
                )}
              </button>
            </div>
          </div>
        </div>
      </header>

      {/* Main content */}
      <main className="flex-1">
        <div className={`mx-auto px-3 sm:px-4 lg:px-6 py-6 ${isTestRunPage || isBuilderPage ? '' : 'max-w-screen-2xl'}`}>
          <Outlet />
        </div>
      </main>

      {/* Footer */}
      <footer className="bg-[var(--color-bg-secondary)] border-t border-[var(--color-border)] py-3">
        <div className="max-w-screen-2xl mx-auto px-3 sm:px-4 lg:px-6">
          <p className="text-center text-sm text-[var(--color-text-tertiary)]">
            Assertoor - Ethereum Test Orchestration
          </p>
        </div>
      </footer>
    </div>
  );
}

function SunIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z"
      />
    </svg>
  );
}

function MoonIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z"
      />
    </svg>
  );
}

export default Layout;
