/**
 * Main application component with routing and authentication.
 */

import { useEffect, useState } from 'react';
import { AuthProvider } from './contexts/AuthContext';
import { ProtectedRoute } from './components/ProtectedRoute';
import { SignIn } from './pages/SignIn';
import { Dashboard } from './pages/Dashboard';

type Route = 'sign-in' | 'dashboard' | 'home';

function App() {
  const [route, setRoute] = useState<Route>('home');

  // Simple client-side routing based on pathname
  useEffect(() => {
    const updateRoute = () => {
      const path = window.location.pathname;
      if (path === '/sign-in') {
        setRoute('sign-in');
      } else if (path === '/dashboard') {
        setRoute('dashboard');
      } else {
        setRoute('home');
      }
    };

    updateRoute();

    // Listen for navigation events
    window.addEventListener('popstate', updateRoute);
    return () => window.removeEventListener('popstate', updateRoute);
  }, []);

  return (
    <AuthProvider>
      {route === 'sign-in' && <SignIn />}
      {route === 'dashboard' && (
        <ProtectedRoute>
          <Dashboard />
        </ProtectedRoute>
      )}
      {route === 'home' && (
        <main style={{ maxWidth: '800px', margin: '100px auto', padding: '20px', textAlign: 'center' }}>
          <h1>HUBX</h1>
          <p>Air cargo traceability platform</p>
          <div style={{ marginTop: '32px' }}>
            <a
              href="/sign-in"
              style={{
                padding: '12px 24px',
                fontSize: '16px',
                backgroundColor: '#007bff',
                color: 'white',
                textDecoration: 'none',
                borderRadius: '4px',
                display: 'inline-block',
              }}
            >
              Sign In
            </a>
          </div>
        </main>
      )}
    </AuthProvider>
  );
}

export default App;
