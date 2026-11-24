/**
 * Authentication service for HUBX.
 * Handles sign-in, token management, and authorization headers.
 * 
 * Token storage: JWT is kept in memory only (not localStorage or sessionStorage).
 * The token does not survive page refresh; users must sign in again.
 */

/**
 * In-memory token storage.
 * Cleared on page refresh or navigation.
 */
let token: string | null = null;

export interface SignInResponse {
  token: string;
  user: {
    email: string;
    role: 'operator' | 'ground partner' | 'customer';
  };
}

export interface SignInRequest {
  email: string;
  password: string;
}

/**
 * Sign in with email and password.
 * Returns JWT token and user information on success.
 * Throws on network error or authentication failure.
 */
export async function signIn(email: string, password: string): Promise<SignInResponse> {
  const response = await fetch('/api/auth/sign-in', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ email, password } satisfies SignInRequest),
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Sign in failed' }));
    throw new Error(error.message || 'Sign in failed');
  }

  const data = (await response.json()) as SignInResponse;
  return data;
}

/**
 * Get the JWT token from memory.
 * Returns null if no token is stored.
 */
export function getToken(): string | null {
  return token;
}

/**
 * Store the JWT token in memory.
 * Token is lost on page refresh.
 */
export function setToken(newToken: string): void {
  token = newToken;
}

/**
 * Clear the JWT token from memory.
 */
export function clearToken(): void {
  token = null;
}

/**
 * Get authorization headers for authenticated requests.
 * Returns empty object if no token is available.
 */
export function getAuthHeaders(): { Authorization: string } | Record<string, never> {
  const currentToken = getToken();
  if (!currentToken) {
    return {};
  }
  return { Authorization: `Bearer ${currentToken}` };
}
