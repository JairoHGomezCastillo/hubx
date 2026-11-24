/**
 * Dashboard placeholder page.
 * Future: role-based views for operator, ground partner, and customer.
 */

import { useAuth } from '../contexts/AuthContext';
import { navigateTo } from '../services/navigation';
import { RecordUpdateContainer } from '../features/shipment-updates/RecordUpdateContainer';
import { ViewTraceabilityContainer } from '../features/traceability/ViewTraceabilityContainer';

export function Dashboard() {
  const { user, signOut } = useAuth();

  const handleSignOut = () => {
    signOut();
    navigateTo('/sign-in');
  };

  return (
    <main style={{ maxWidth: '800px', margin: '40px auto', padding: '20px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '32px' }}>
        <h1>Dashboard</h1>
        <button
          onClick={handleSignOut}
          style={{
            padding: '8px 16px',
            fontSize: '14px',
            backgroundColor: '#6c757d',
            color: 'white',
            border: 'none',
            borderRadius: '4px',
            cursor: 'pointer',
          }}
        >
          Sign Out
        </button>
      </div>

      {user && (
        <div
          style={{
            padding: '16px',
            backgroundColor: '#f8f9fa',
            border: '1px solid #dee2e6',
            borderRadius: '4px',
          }}
        >
          <p style={{ margin: '0 0 8px 0' }}>
            <strong>Email:</strong> {user.email}
          </p>
          <p style={{ margin: '0' }}>
            <strong>Role:</strong> {user.role}
          </p>
        </div>
      )}

      <RecordUpdateContainer onSessionExpired={handleSignOut} />

      <ViewTraceabilityContainer onSessionExpired={handleSignOut} />

      {user?.role === 'operator' && (
        <div style={{ marginTop: '32px' }}>
          <button
            onClick={() => navigateTo('/shipments/new')}
            style={{
              padding: '12px 24px',
              fontSize: '16px',
              backgroundColor: '#28a745',
              color: 'white',
              border: 'none',
              borderRadius: '4px',
              cursor: 'pointer',
            }}
          >
            Create shipment
          </button>
        </div>
      )}

      <div style={{ marginTop: '32px' }}>
        <a
          href="/verify-integrity"
          onClick={(e) => {
            e.preventDefault();
            navigateTo('/verify-integrity');
          }}
          style={{
            display: 'inline-block',
            padding: '12px 24px',
            fontSize: '16px',
            backgroundColor: '#007bff',
            color: 'white',
            border: 'none',
            borderRadius: '4px',
            cursor: 'pointer',
            textDecoration: 'none',
          }}
        >
          Verify integrity
        </a>
      </div>

      <div style={{ marginTop: '32px' }}>
        <p style={{ color: '#666' }}>
          Welcome to HUBX. Role-specific features will be added in future stories.
        </p>
      </div>
    </main>
  );
}
