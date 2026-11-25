/**
 * Wires the accessible RecordUpdate form to the HU-03 /api client.
 * Kept separate from the presentational form so that every query state is
 * explicit and the form can be tested with an injected transport.
 */

import { useEffect, useRef, useState } from 'react';
import { listGroundPartners, recordUpdate } from '../../api/client';
import type { GroundPartner } from './RecordUpdate';
import RecordUpdate from './RecordUpdate';

type PartnerLoadState =
  | { kind: 'loading' }
  | { kind: 'success'; partners: GroundPartner[] }
  | { kind: 'empty' }
  | { kind: 'httpError'; status: number; code: string; message: string }
  | { kind: 'noResponse' }
  | { kind: 'unauthorized' };

/** Errors use `alert`: the form's single `status` region belongs to the submission. */
function loadFailureMessage(state: PartnerLoadState): string | null {
  switch (state.kind) {
    case 'httpError':
      return state.status === 401 || state.code === 'SESSION_EXPIRED'
        ? 'Session expired; sign in again.'
        : state.message;
    case 'noResponse':
      return 'No response from the server. Try again.';
    default:
      return null;
  }
}

interface RecordUpdateContainerProps {
  onSessionExpired: () => void;
}

export function RecordUpdateContainer({ onSessionExpired }: RecordUpdateContainerProps) {
  const [partners, setPartners] = useState<PartnerLoadState>({ kind: 'loading' });

  // The callback from the page is not stable; a ref keeps the lookup effect
  // from re-running on every render.
  const onSessionExpiredRef = useRef(onSessionExpired);
  useEffect(() => {
    onSessionExpiredRef.current = onSessionExpired;
  }, [onSessionExpired]);

  useEffect(() => {
    let active = true;
    void (async () => {
      const outcome = await listGroundPartners();
      if (!active) {
        return;
      }
      if (outcome.kind === 'unauthorized') {
        // fetchWithAuth already navigated to sign-in with the expired message.
        onSessionExpiredRef.current();
        return;
      }
      setPartners(outcome);
    })();
    return () => {
      active = false;
    };
  }, []);

  const availablePartners = partners.kind === 'success' ? partners.partners : [];
  const loadError = loadFailureMessage(partners);

  return (
    <div className="stack">
      {partners.kind === 'loading' && <p className="note">Loading ground partners…</p>}
      {loadError !== null && <p className="message" data-tone="bad" role="alert">{loadError}</p>}
      <RecordUpdate
        groundPartners={availablePartners}
        onRecordUpdate={recordUpdate}
        onSessionExpired={onSessionExpired}
      />
    </div>
  );
}
