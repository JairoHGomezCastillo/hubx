/**
 * Sign-in page for HUBX users.
 * Authenticates operators, ground partners, and customers.
 */

import { useEffect, useState } from 'react';
import type { FormEvent } from 'react';
import { useAuth } from '../contexts/AuthContext';

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

  // Redirect to dashboard if already authenticated
  useEffect(() => {
    if (isAuthenticated) {
      window.location.href = '/dashboard';
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
    <main style={{ maxWidth: '400px', margin: '100px auto', padding: '20px' }}>
      <h1>Sign In</h1>
      
      <form onSubmit={handleSubmit}>
        <div style={{ marginBottom: '16px' }}>
          <label htmlFor="email" style={{ display: 'block', marginBottom: '8px' }}>
            Email
          </label>
          <input
            id="email"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
            disabled={isLoading}
            style={{ width: '100%', padding: '8px', fontSize: '16px' }}
            autoComplete="email"
          />
        </div>

        <div style={{ marginBottom: '16px' }}>
          <label htmlFor="password" style={{ display: 'block', marginBottom: '8px' }}>
            Password
          </label>
          <input
            id="password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            disabled={isLoading}
            style={{ width: '100%', padding: '8px', fontSize: '16px' }}
            autoComplete="current-password"
          />
        </div>

        {error && (
          <div
            style={{
              padding: '12px',
              marginBottom: '16px',
              backgroundColor: '#fee',
              border: '1px solid #fcc',
              borderRadius: '4px',
              color: '#c00',
            }}
            role="alert"
          >
            {error}
          </div>
        )}

        <button
          type="submit"
          disabled={isLoading}
          style={{
            width: '100%',
            padding: '12px',
            fontSize: '16px',
            backgroundColor: '#007bff',
            color: 'white',
            border: 'none',
            borderRadius: '4px',
            cursor: isLoading ? 'not-allowed' : 'pointer',
            opacity: isLoading ? 0.6 : 1,
          }}
        >
          {isLoading ? 'Signing in...' : 'Sign In'}
        </button>
      </form>

      <div style={{ marginTop: '24px', fontSize: '14px', color: '#666' }}>
        <p>Demo users:</p>
        <ul>
          <li>operator@hubx.co</li>
          <li>rapido@hubx.co (ground partner)</li>
          <li>ana@hubx.co (customer)</li>
        </ul>
        <p>Password: Hubx-Demo-2026</p>
      </div>
    </main>
  );
}
