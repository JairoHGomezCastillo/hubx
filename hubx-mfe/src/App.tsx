/**
 * Main application component with routing and authentication.
 */

import { useEffect, useState } from 'react';
import { AuthProvider, useAuth } from './contexts/AuthContext';
import { ProtectedRoute } from './components/ProtectedRoute';
import { SignIn } from './pages/SignIn';
import { Dashboard } from './pages/Dashboard';
import { CreateShipment } from './pages/CreateShipment';
import { navigateTo } from './services/navigation';

type Route = 'sign-in' | 'dashboard' | 'home' | 'create-shipment';

/** The root path is not a destination: enter the app through sign-in or the dashboard. */
function HomeRedirect() {
  const { isAuthenticated } = useAuth();

  useEffect(() => {
    navigateTo(isAuthenticated ? '/dashboard' : '/sign-in');
  }, [isAuthenticated]);

  return null;
}

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
      } else if (path === '/shipments/new') {
        setRoute('create-shipment');
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
      {route === 'create-shipment' && (
        <ProtectedRoute>
          <CreateShipment />
        </ProtectedRoute>
      )}
      {route === 'home' && <HomeRedirect />}
    </AuthProvider>
  );
}

export default App;
