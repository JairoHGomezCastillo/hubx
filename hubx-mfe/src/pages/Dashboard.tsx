/**
 * Dashboard: record updates and view the traceability of a shipment.
 */

import { AppShell } from '../components/AppShell';
import { useAuth } from '../contexts/AuthContext';
import { navigateTo } from '../services/navigation';
import { RecordUpdateContainer } from '../features/shipment-updates/RecordUpdateContainer';
import { ViewTraceabilityContainer } from '../features/traceability/ViewTraceabilityContainer';

export function Dashboard() {
  const { user, signOut } = useAuth();

  const handleSessionExpired = () => {
    signOut();
    navigateTo('/sign-in');
  };

  return (
    <AppShell>
      <div className="page-head">
        <h1>Dashboard</h1>
        <div className="actions">
          <a
            href="/verify-integrity"
            className="button button-secondary"
            onClick={(e) => {
              e.preventDefault();
              navigateTo('/verify-integrity');
            }}
          >
            Verify integrity
          </a>
          {user?.role === 'operator' && (
            <>
              <a
                href="/download-evidence"
                className="button button-secondary"
                onClick={(e) => {
                  e.preventDefault();
                  navigateTo('/download-evidence');
                }}
              >
                Download evidence
              </a>
              <button type="button" className="button" onClick={() => navigateTo('/shipments/new')}>
                Create shipment
              </button>
            </>
          )}
        </div>
      </div>

      <div className="workspace">
        <RecordUpdateContainer onSessionExpired={handleSessionExpired} />
        <ViewTraceabilityContainer onSessionExpired={handleSessionExpired} />
      </div>
    </AppShell>
  );
}
