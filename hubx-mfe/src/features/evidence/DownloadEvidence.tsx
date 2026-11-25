/**
 * HU-10 evidence download view: presentational only, with an injected
 * transport. Locators mirror the @e2e scenario (region, label, button).
 */

import { useState } from 'react';
import type { ChangeEvent, FormEvent } from 'react';

/** Explicit outcome shown by the view; the container maps service states to it. */
export type DownloadEvidenceOutcome =
  | { kind: 'downloaded'; filename: string }
  | { kind: 'unauthorized' }
  | { kind: 'forbidden'; message: string }
  | { kind: 'not-found'; message: string }
  | { kind: 'httpError'; status: number; message: string }
  | { kind: 'noResponse' };

export interface DownloadEvidenceProps {
  onDownload: (shipmentCode: string) => Promise<DownloadEvidenceOutcome>;
  onSessionExpired: () => void;
}

type SubmissionState =
  | { kind: 'idle' }
  | { kind: 'submitting' }
  | DownloadEvidenceOutcome;

function submissionMessage(state: SubmissionState): string {
  switch (state.kind) {
    case 'idle':
      return 'Enter a shipment code to download its traceability evidence.';
    case 'submitting':
      return 'Downloading evidence…';
    case 'downloaded':
      return `The evidence was saved as ${state.filename}.`;
    case 'unauthorized':
      return 'Session expired. Please sign in again.';
    case 'forbidden':
    case 'not-found':
    case 'httpError':
      return state.message;
    case 'noResponse':
      return 'No response from the server. Try again.';
  }
}

function submissionTone(state: SubmissionState): 'ok' | 'bad' | undefined {
  switch (state.kind) {
    case 'downloaded':
      return 'ok';
    case 'idle':
    case 'submitting':
      return undefined;
    default:
      return 'bad';
  }
}

function DownloadEvidence({ onDownload, onSessionExpired }: DownloadEvidenceProps) {
  const [shipmentCode, setShipmentCode] = useState('');
  const [submission, setSubmission] = useState<SubmissionState>({ kind: 'idle' });

  const busy = submission.kind === 'submitting';

  const handleCodeChange = (event: ChangeEvent<HTMLInputElement>) => {
    setShipmentCode(event.target.value);
    setSubmission({ kind: 'idle' });
  };

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (busy) {
      return;
    }

    const code = shipmentCode.trim();
    if (!code) {
      setSubmission({
        kind: 'httpError',
        status: 400,
        message: 'Please enter a shipment code.',
      });
      return;
    }

    setSubmission({ kind: 'submitting' });
    try {
      const outcome = await onDownload(code);
      setSubmission(outcome);
      if (outcome.kind === 'unauthorized') {
        onSessionExpired();
      }
    } catch {
      setSubmission({ kind: 'noResponse' });
    }
  };

  const handleFormSubmit = (event: FormEvent<HTMLFormElement>) => {
    void handleSubmit(event);
  };

  return (
    <section className="panel" aria-labelledby="download-evidence-heading">
      <h2 id="download-evidence-heading">Download evidence</h2>
      <form className="form-inline" noValidate onSubmit={handleFormSubmit}>
        <div className="field">
          <label htmlFor="download-evidence-shipment-code">Shipment code</label>
          <input
            id="download-evidence-shipment-code"
            name="shipmentCode"
            type="text"
            value={shipmentCode}
            onChange={handleCodeChange}
            disabled={busy}
            required
          />
        </div>
        <button className="button" type="submit" disabled={busy}>
          Download evidence
        </button>
      </form>
      <p className="message" data-tone={submissionTone(submission)} role="status" aria-live="polite">
        {submissionMessage(submission)}
      </p>
    </section>
  );
}

export default DownloadEvidence;
