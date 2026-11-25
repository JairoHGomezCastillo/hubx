/**
 * Download evidence page - operator-only, reachable from the Dashboard.
 */

import { AppShell } from '../components/AppShell';
import { useAuth } from '../contexts/AuthContext';
import { navigateTo } from '../services/navigation';
import { DownloadEvidenceContainer } from '../features/evidence/DownloadEvidenceContainer';

export function DownloadEvidence() {
  const { signOut } = useAuth();

  const handleSessionExpired = () => {
    signOut();
    navigateTo('/sign-in');
  };

  return (
    <AppShell narrow showBack>
      <h1>Download evidence</h1>
      <DownloadEvidenceContainer onSessionExpired={handleSessionExpired} />
    </AppShell>
  );
}
