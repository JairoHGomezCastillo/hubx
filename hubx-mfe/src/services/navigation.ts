/**
 * Client-side navigation utilities.
 * Navigates within the app without full-page reload, preserving in-memory token.
 */

/**
 * Navigate to a path within the app using history API.
 * This keeps the React app mounted and preserves in-memory state (JWT token).
 * 
 * @param path - Target path (e.g., '/dashboard', '/sign-in')
 */
export function navigateTo(path: string): void {
  window.history.pushState({}, '', path);
  // Dispatch popstate event so App.tsx's useEffect picks up the route change
  window.dispatchEvent(new PopStateEvent('popstate'));
}
