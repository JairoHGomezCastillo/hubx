/**
 * Verify Integrity page - accessible to all signed-in users.
 */

import { navigateTo } from '../services/navigation';
import { VerifyIntegrityContainer } from '../features/integrity/VerifyIntegrityContainer';

export function VerifyIntegrity() {
  const handleSessionExpired = () => {
    navigateTo('/sign-in');
  };

  return (
    <main style={{ maxWidth: '800px', margin: '40px auto', padding: '20px' }}>
      <div style={{ marginBottom: '32px' }}>
        <button
          onClick={() => navigateTo('/dashboard')}
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
          Back to Dashboard
        </button>
      </div>

      <VerifyIntegrityContainer onSessionExpired={handleSessionExpired} />
    </main>
  );
}
