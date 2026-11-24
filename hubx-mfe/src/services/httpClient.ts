/**
 * HTTP client wrapper for HUBX API calls.
 * Automatically includes authorization headers and handles 401 responses.
 */

import * as authService from './authService';
import { navigateTo } from './navigation';

export interface ApiError {
  error: string;
  message: string;
}

/** HU-01 sends `message` at the root; HU-03's contract nests it under `error`. */
interface UnauthorizedBody {
  message?: string;
  error?: { code?: string; message?: string };
}

interface FetchOptions extends RequestInit {
  headers?: HeadersInit;
}

/**
 * Fetch wrapper that includes Authorization header and handles session expiration.
 * On 401 response, clears token and redirects to sign-in page.
 * 
 * @param url - API endpoint (relative path starting with /api)
 * @param options - Fetch options (method, body, headers, etc.)
 * @returns Response object
 * @throws Error if network fails or response includes error message
 */
export async function fetchWithAuth(url: string, options: FetchOptions = {}): Promise<Response> {
  // Merge authorization headers with any provided headers
  const authHeaders = authService.getAuthHeaders();
  const headers = new Headers(options.headers);
  
  if ('Authorization' in authHeaders) {
    headers.set('Authorization', authHeaders.Authorization);
  }

  const response = await fetch(url, {
    ...options,
    headers,
  });

  // Handle 401 Unauthorized - session expired or invalid token
  if (response.status === 401) {
    const errorData = await response.json().catch(() => null) as UnauthorizedBody | null;
    
    // Clear token and redirect to sign-in
    authService.clearToken();
    
    // Accept both error shapes so the expired-session notice is always stored.
    const message = errorData?.message ?? errorData?.error?.message;
    const isSessionExpired = message?.includes('Session expired') === true ||
                            message?.includes('expired') === true;
    
    if (isSessionExpired) {
      // Show notification (in a real app, this would use a toast/notification system)
      // For now, we'll store a flag that the sign-in page can read
      sessionStorage.setItem('session_expired_message', 'Session expired. Please sign in again.');
    }
    
    // Redirect to sign-in page (client-side, never a full reload)
    navigateTo('/sign-in');
    
    // Throw error to prevent further processing
    throw new Error(message || 'Unauthorized');
  }

  return response;
}

/**
 * Helper to make a JSON GET request with authentication.
 */
export async function getJson<T>(url: string): Promise<T> {
  const response = await fetchWithAuth(url);
  
  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Request failed' })) as ApiError;
    throw new Error(error.message || 'Request failed');
  }
  
  return response.json() as Promise<T>;
}

/**
 * Helper to make a JSON POST request with authentication.
 */
export async function postJson<T>(url: string, body: unknown): Promise<T> {
  const response = await fetchWithAuth(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(body),
  });
  
  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Request failed' })) as ApiError;
    throw new Error(error.message || 'Request failed');
  }
  
  return response.json() as Promise<T>;
}
