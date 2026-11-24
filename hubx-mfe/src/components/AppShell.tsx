/**
 * Frame of every signed-in page: brand sign, current user and sign out.
 */

import type { ReactNode } from 'react';
import { useAuth } from '../contexts/AuthContext';
import { navigateTo } from '../services/navigation';

interface AppShellProps {
  children: ReactNode;
  /** Single-column width for forms and short content. */
  narrow?: boolean;
  /** Shows the way back to the dashboard. */
  showBack?: boolean;
}

export function AppShell({ children, narrow = false, showBack = false }: AppShellProps) {
  const { user, signOut } = useAuth();

  const handleSignOut = () => {
    signOut();
    navigateTo('/sign-in');
  };

  return (
    <>
      <header className="topbar">
        <div className="topbar-inner">
          <span className="sign">HUBX</span>
          {user && (
            <div className="identity">
              <p>{user.email}</p>
              <p className="identity-role">Role: {user.role}</p>
            </div>
          )}
          <button type="button" className="button button-on-asphalt" onClick={handleSignOut}>
            Sign Out
          </button>
        </div>
      </header>
      <main className={narrow ? 'page page-narrow' : 'page'}>
        {showBack && (
          <button
            type="button"
            className="button button-secondary back"
            onClick={() => navigateTo('/dashboard')}
          >
            Back to Dashboard
          </button>
        )}
        {children}
      </main>
    </>
  );
}
