/**
 * Sign-in page for HUBX users.
 * Authenticates operators, ground partners, and customers.
 */

import { useEffect, useState } from 'react';
import type { FormEvent } from 'react';
import { useAuth } from '../contexts/AuthContext';
import { navigateTo } from '../services/navigation';

export function SignIn() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  
  const { signIn, isAuthenticated } = useAuth();

  // Check for session expired message
  useEffect(() => {
    const message = sessionStorage.getItem('session_expired_message');
    if (message) {
      setError(message);
      sessionStorage.removeItem('session_expired_message');
    }
  }, []);

  // Navigate to dashboard if already authenticated (client-side, preserves in-memory token)
  useEffect(() => {
    if (isAuthenticated) {
      navigateTo('/dashboard');
    }
  }, [isAuthenticated]);

  const handleSubmit = async (e: FormEvent): Promise<void> => {
    e.preventDefault();
    setError(null);
    setIsLoading(true);

    try {
      // Trim email for better UX (backend handles normalization)
      await signIn(email.trim(), password);
      // Redirect happens via useEffect when isAuthenticated becomes true
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Sign in failed';
      setError(message);
      setIsLoading(false);
    }
  };

  return (
    <main className="sign-in">
      <div className="sign-in-apron">
        <span className="sign">HUBX</span>
        <p>Air cargo, traced from takeoff to delivery.</p>
        <svg viewBox="0 0 600 400" preserveAspectRatio="xMidYMax slice" aria-hidden="true">
          <path d="M-20 330H230C340 330 350 250 350 180S400 70 620 70" fill="none" stroke="#111617" strokeWidth="22" />
          <path d="M-20 330H230C340 330 350 250 350 180S400 70 620 70" fill="none" stroke="#f7bd16" strokeWidth="10" />
        </svg>
      </div>
      <div className="sign-in-form">
      <h1>Sign In</h1>
      
      <form className="form" onSubmit={handleSubmit}>
        <div className="field">
          <label htmlFor="email">
            Email
          </label>
          <input
            id="email"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
            disabled={isLoading}
            autoComplete="email"
          />
        </div>

        <div className="field">
          <label htmlFor="password">
            Password
          </label>
          <input
            id="password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            disabled={isLoading}
            autoComplete="current-password"
          />
        </div>

        {error && (
          <div className="message" data-tone="bad" role="alert">
            {error}
          </div>
        )}

        <button type="submit" className="button" disabled={isLoading}>
          {isLoading ? 'Signing in...' : 'Sign In'}
        </button>
      </form>

      <div className="demo-users">
        <p>Demo users:</p>
        <ul>
          <li>operator@hubx.co</li>
          <li>rapido@hubx.co (ground partner)</li>
          <li>ana@hubx.co (customer)</li>
        </ul>
        <p>Password: Hubx-Demo-2026</p>
      </div>
      </div>
    </main>
  );
}
