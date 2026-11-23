/**
 * Unit tests for AuthContext.
 * Tests AuthProvider state management and useAuth hook.
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { act } from 'react';
import { AuthProvider, useAuth } from './AuthContext';
import * as authService from '../services/authService';

// Mock authService
vi.mock('../services/authService');

// Test component that uses useAuth
function TestComponent() {
  const { user, isAuthenticated, signIn, signOut } = useAuth();

  return (
    <div>
      <div data-testid="auth-status">{isAuthenticated ? 'authenticated' : 'not-authenticated'}</div>
      {user && (
        <div>
          <div data-testid="user-email">{user.email}</div>
          <div data-testid="user-role">{user.role}</div>
        </div>
      )}
      <button
        onClick={() => signIn('test@hubx.co', 'password')}
        data-testid="sign-in-button"
      >
        Sign In
      </button>
      <button onClick={signOut} data-testid="sign-out-button">
        Sign Out
      </button>
    </div>
  );
}

describe('AuthContext', () => {
  beforeEach(() => {
    localStorage.clear();
    vi.clearAllMocks();
  });

  describe('AuthProvider', () => {
    it('should start with unauthenticated state when no token exists', () => {
      vi.mocked(authService.getToken).mockReturnValue(null);

      render(
        <AuthProvider>
          <TestComponent />
        </AuthProvider>
      );

      expect(screen.getByTestId('auth-status')).toHaveTextContent('not-authenticated');
    });

    it('should restore user from valid token on mount', () => {
      const validToken = btoa(
        JSON.stringify({ alg: 'HS256', typ: 'JWT' })
      ) + '.' + btoa(
        JSON.stringify({
          email: 'operator@hubx.co',
          role: 'operator',
          exp: Math.floor(Date.now() / 1000) + 3600, // expires in 1 hour
        })
      ) + '.signature';

      vi.mocked(authService.getToken).mockReturnValue(validToken);

      render(
        <AuthProvider>
          <TestComponent />
        </AuthProvider>
      );

      expect(screen.getByTestId('auth-status')).toHaveTextContent('authenticated');
      expect(screen.getByTestId('user-email')).toHaveTextContent('operator@hubx.co');
      expect(screen.getByTestId('user-role')).toHaveTextContent('operator');
    });

    it('should clear expired token on mount', () => {
      const expiredToken = btoa(
        JSON.stringify({ alg: 'HS256', typ: 'JWT' })
      ) + '.' + btoa(
        JSON.stringify({
          email: 'operator@hubx.co',
          role: 'operator',
          exp: Math.floor(Date.now() / 1000) - 3600, // expired 1 hour ago
        })
      ) + '.signature';

      vi.mocked(authService.getToken).mockReturnValue(expiredToken);
      vi.mocked(authService.clearToken).mockImplementation(() => {});

      render(
        <AuthProvider>
          <TestComponent />
        </AuthProvider>
      );

      expect(authService.clearToken).toHaveBeenCalled();
      expect(screen.getByTestId('auth-status')).toHaveTextContent('not-authenticated');
    });

    it('should clear invalid token on mount', () => {
      vi.mocked(authService.getToken).mockReturnValue('invalid.token');
      vi.mocked(authService.clearToken).mockImplementation(() => {});

      render(
        <AuthProvider>
          <TestComponent />
        </AuthProvider>
      );

      expect(authService.clearToken).toHaveBeenCalled();
      expect(screen.getByTestId('auth-status')).toHaveTextContent('not-authenticated');
    });
  });

  describe('signIn', () => {
    it('should update user state on successful sign-in', async () => {
      vi.mocked(authService.getToken).mockReturnValue(null);
      vi.mocked(authService.signIn).mockResolvedValue({
        token: 'new.jwt.token',
        user: {
          email: 'test@hubx.co',
          role: 'customer',
        },
      });
      vi.mocked(authService.setToken).mockImplementation(() => {});

      render(
        <AuthProvider>
          <TestComponent />
        </AuthProvider>
      );

      const signInButton = screen.getByTestId('sign-in-button');
      
      await act(async () => {
        signInButton.click();
      });

      await waitFor(() => {
        expect(screen.getByTestId('auth-status')).toHaveTextContent('authenticated');
      });

      expect(screen.getByTestId('user-email')).toHaveTextContent('test@hubx.co');
      expect(screen.getByTestId('user-role')).toHaveTextContent('customer');
      expect(authService.setToken).toHaveBeenCalledWith('new.jwt.token');
    });

    it('should handle ground partner role with space', async () => {
      vi.mocked(authService.getToken).mockReturnValue(null);
      vi.mocked(authService.signIn).mockResolvedValue({
        token: 'partner.jwt.token',
        user: {
          email: 'rapido@hubx.co',
          role: 'ground partner',
        },
      });
      vi.mocked(authService.setToken).mockImplementation(() => {});

      render(
        <AuthProvider>
          <TestComponent />
        </AuthProvider>
      );

      const signInButton = screen.getByTestId('sign-in-button');
      
      await act(async () => {
        signInButton.click();
      });

      await waitFor(() => {
        expect(screen.getByTestId('user-role')).toHaveTextContent('ground partner');
      });
    });
  });

  describe('signOut', () => {
    it('should clear user state and token on sign-out', async () => {
      const validToken = btoa(
        JSON.stringify({ alg: 'HS256', typ: 'JWT' })
      ) + '.' + btoa(
        JSON.stringify({
          email: 'operator@hubx.co',
          role: 'operator',
          exp: Math.floor(Date.now() / 1000) + 3600,
        })
      ) + '.signature';

      vi.mocked(authService.getToken).mockReturnValue(validToken);
      vi.mocked(authService.clearToken).mockImplementation(() => {});

      render(
        <AuthProvider>
          <TestComponent />
        </AuthProvider>
      );

      expect(screen.getByTestId('auth-status')).toHaveTextContent('authenticated');

      const signOutButton = screen.getByTestId('sign-out-button');
      
      await act(async () => {
        signOutButton.click();
      });

      expect(authService.clearToken).toHaveBeenCalled();
      expect(screen.getByTestId('auth-status')).toHaveTextContent('not-authenticated');
    });
  });

  describe('useAuth hook', () => {
    it('should throw error when used outside AuthProvider', () => {
      // Suppress console.error for this test
      const originalError = console.error;
      console.error = vi.fn();

      expect(() => {
        render(<TestComponent />);
      }).toThrow('useAuth must be used within an AuthProvider');

      console.error = originalError;
    });
  });
});
