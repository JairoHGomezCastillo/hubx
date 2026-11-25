/**
 * Verify Integrity page - accessible to all signed-in users.
 */

import { AppShell } from '../components/AppShell';
import { navigateTo } from '../services/navigation';
import { VerifyIntegrityContainer } from '../features/integrity/VerifyIntegrityContainer';

export function VerifyIntegrity() {
  const handleSessionExpired = () => {
    navigateTo('/sign-in');
  };

  return (
    <AppShell narrow showBack>
      <VerifyIntegrityContainer onSessionExpired={handleSessionExpired} />
    </AppShell>
  );
}
