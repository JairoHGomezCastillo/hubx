/**
 * Dashboard placeholder page.
 * Future: role-based views for operator, ground partner, and customer.
 */

import { useAuth } from '../contexts/AuthContext';

export function Dashboard() {
  const { user, signOut } = useAuth();

  return (
    <main style={{ maxWidth: '800px', margin: '40px auto', padding: '20px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '32px' }}>
        <h1>Dashboard</h1>
        <button
          onClick={signOut}
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

      <div style={{ marginTop: '32px' }}>
        <p style={{ color: '#666' }}>
          Welcome to HUBX. Role-specific features will be added in future stories.
        </p>
      </div>
    </main>
  );
}
