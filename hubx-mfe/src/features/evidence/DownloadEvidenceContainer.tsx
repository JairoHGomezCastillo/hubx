/**
 * Container for the HU-10 evidence view: wires the service layer and maps its
 * explicit outcomes to the messages shown by the presentational component.
 */

import DownloadEvidence from './DownloadEvidence';
import type { DownloadEvidenceOutcome } from './DownloadEvidence';
import { downloadEvidence } from '../../services/evidenceService';
import type { EvidenceDownloadOutcome } from '../../services/evidenceService';

export interface DownloadEvidenceContainerProps {
  onSessionExpired: () => void;
}

/** The 403/404 messages come verbatim from contracts/hu10-download-evidence.md. */
function toViewOutcome(outcome: EvidenceDownloadOutcome): DownloadEvidenceOutcome {
  switch (outcome.kind) {
    case 'downloaded':
      return { kind: 'downloaded', filename: outcome.filename };
    case 'unauthorized':
      // fetchWithAuth has already cleared the token and navigated to sign-in.
      return { kind: 'unauthorized' };
    case 'forbidden':
      return { kind: 'forbidden', message: 'You do not have permission to download evidence' };
    case 'not-found':
      return { kind: 'not-found', message: 'No shipment exists with that code' };
    case 'httpError':
      return { kind: 'httpError', status: outcome.status, message: outcome.message };
    case 'noResponse':
      return { kind: 'noResponse' };
  }
}

export function DownloadEvidenceContainer({ onSessionExpired }: DownloadEvidenceContainerProps) {
  const handleDownload = async (shipmentCode: string): Promise<DownloadEvidenceOutcome> => {
    try {
      return toViewOutcome(await downloadEvidence(shipmentCode));
    } catch {
      return { kind: 'noResponse' };
    }
  };

  return <DownloadEvidence onDownload={handleDownload} onSessionExpired={onSessionExpired} />;
}
