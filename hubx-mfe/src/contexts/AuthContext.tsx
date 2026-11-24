/**
 * Authentication context for HUBX.
 * Provides user state and authentication methods throughout the app.
 */

import { createContext, useContext, useEffect, useState } from 'react';
import type { ReactNode } from 'react';
import * as authService from '../services/authService';

interface User {
  email: string;
  role: 'operator' | 'ground partner' | 'customer';
}

interface AuthContextValue {
  user: User | null;
  isAuthenticated: boolean;
  signIn: (email: string, password: string) => Promise<void>;
  signOut: () => void;
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

interface AuthProviderProps {
  children: ReactNode;
}

/**
 * AuthProvider manages authentication state and provides sign-in/sign-out methods.
 * Token is stored in memory only; no rehydration from storage on mount.
 */
export function AuthProvider({ children }: AuthProviderProps) {
  const [user, setUser] = useState<User | null>(null);

  // Check if token exists in memory on mount (e.g., after navigating between routes)
  // With in-memory storage, token is lost on page refresh
  useEffect(() => {
    const token = authService.getToken();
    if (token) {
      // Token exists in memory; decode to extract user info
      // JWT structure: header.payload.signature
      // payload is base64url encoded JSON with { email, role, exp, ... }
      try {
        const payloadBase64 = token.split('.')[1];
        if (!payloadBase64) {
          authService.clearToken();
          return;
        }
        
        // Decode base64url to JSON
        const payload = JSON.parse(atob(payloadBase64.replace(/-/g, '+').replace(/_/g, '/')));
        
        // Check if token is expired
        const now = Math.floor(Date.now() / 1000);
        if (payload.exp && payload.exp < now) {
          // Token expired, clear it
          authService.clearToken();
          return;
        }
        
        // Restore user from token claims
        if (payload.email && payload.role) {
          setUser({ email: payload.email, role: payload.role });
        } else {
          authService.clearToken();
        }
      } catch {
        // Invalid token format, clear it
        authService.clearToken();
      }
    }
  }, []);

  const handleSignIn = async (email: string, password: string): Promise<void> => {
    const response = await authService.signIn(email, password);
    authService.setToken(response.token);
    setUser(response.user);
  };

  const handleSignOut = (): void => {
    authService.clearToken();
    setUser(null);
  };

  const value: AuthContextValue = {
    user,
    isAuthenticated: user !== null,
    signIn: handleSignIn,
    signOut: handleSignOut,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

/**
 * Hook to access authentication context.
 * Must be used within an AuthProvider.
 */
export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}
