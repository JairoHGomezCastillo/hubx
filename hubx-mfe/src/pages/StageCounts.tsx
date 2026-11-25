/**
 * Stage counts page: operator-only view of shipment counts by stage.
 */

import { AppShell } from '../components/AppShell';
import { useAuth } from '../contexts/AuthContext';
import { navigateTo } from '../services/navigation';
import { StageCountsContainer } from '../features/stage-counts/StageCountsContainer';

export function StageCountsPage() {
  const { signOut } = useAuth();

  const handleSessionExpired = () => {
    signOut();
    navigateTo('/sign-in');
  };

  return (
    <AppShell>
      <StageCountsContainer onSessionExpired={handleSessionExpired} />
    </AppShell>
  );
}
