import React from 'react';
import { useAuthContext } from '../../context/AuthContext';

export const UserDisplay: React.FC = () => {
  const { isLoggedIn, user, loading, login } = useAuthContext();

  if (loading) {
    return (
      <div className="flex items-center">
        <div className="w-4 h-4 border-2 border-[var(--color-text-tertiary)] border-t-transparent rounded-full animate-spin" />
      </div>
    );
  }

  if (!isLoggedIn) {
    return (
      <button
        type="button"
        className="flex items-center gap-2 px-3 py-1.5 text-sm font-medium rounded-md bg-primary-600 text-white hover:bg-primary-700 transition-colors"
        onClick={login}
      >
        <LoginIcon className="w-4 h-4" />
        <span>Login</span>
      </button>
    );
  }

  return (
    <div className="flex items-center gap-2 text-sm text-[var(--color-text-secondary)]">
      <UserIcon className="w-4 h-4" />
      <span>{user}</span>
    </div>
  );
};

function LoginIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M11 16l-4-4m0 0l4-4m-4 4h14m-5 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h7a3 3 0 013 3v1"
      />
    </svg>
  );
}

function UserIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"
      />
    </svg>
  );
}

export default UserDisplay;
