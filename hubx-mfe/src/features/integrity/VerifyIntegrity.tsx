import { useState } from 'react';
import type { ChangeEvent, FormEvent } from 'react';

/** 200 response of GET /api/shipments/{code}/integrity. */
export interface VerificationResult {
  shipmentCode: string;
  intact: boolean;
  updatesChecked: number;
  firstAlteredUpdate: number | null;
}

/** Outcome returned by the injected transport; this component makes no HTTP call. */
export type VerifyIntegrityOutcome =
  | { kind: 'success'; result: VerificationResult }
  | { kind: 'httpError'; status: number; code: string; message: string }
  | { kind: 'noResponse' };

export interface VerifyIntegrityProps {
  onVerify: (shipmentCode: string) => Promise<VerifyIntegrityOutcome>;
  onSessionExpired: () => void;
}

type SubmissionState =
  | { kind: 'idle' }
  | { kind: 'submitting' }
  | { kind: 'unexpectedError' }
  | VerifyIntegrityOutcome;

function submissionMessage(state: SubmissionState): string {
  switch (state.kind) {
    case 'idle':
      return 'Enter a shipment code to verify its integrity.';
    case 'submitting':
      return 'Verifying…';
    case 'success': {
      const { result } = state;
      if (result.intact) {
        if (result.updatesChecked === 0) {
          return 'The chain is intact. There are no updates to check.';
        }
        return `The chain is intact. ${result.updatesChecked} updates checked.`;
      }
      return `The chain is not intact. The first alteration is in update ${result.firstAlteredUpdate}.`;
    }
    case 'httpError':
      return state.status === 401 ? 'Session expired. Please sign in again.' : state.message;
    case 'noResponse':
      return 'No response from the server. Try again.';
    case 'unexpectedError':
      return 'The verification could not be completed.';
  }
}

function submissionTone(state: SubmissionState): 'ok' | 'bad' | undefined {
  switch (state.kind) {
    case 'success':
      return state.result.intact ? 'ok' : 'bad';
    case 'idle':
    case 'submitting':
      return undefined;
    default:
      return 'bad';
  }
}

function VerifyIntegrity({ onVerify, onSessionExpired }: VerifyIntegrityProps) {
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
        code: 'INVALID_REQUEST',
        message: 'Please enter a shipment code.',
      });
      return;
    }

    setSubmission({ kind: 'submitting' });
    try {
      const outcome = await onVerify(code);
      setSubmission(outcome);
      if (outcome.kind === 'httpError' && outcome.status === 401) {
        onSessionExpired();
      }
    } catch {
      setSubmission({ kind: 'unexpectedError' });
    }
  };

  const handleFormSubmit = (event: FormEvent<HTMLFormElement>) => {
    void handleSubmit(event);
  };

  return (
    <section className="panel" aria-labelledby="verify-integrity-heading">
      <h2 id="verify-integrity-heading">Verify integrity</h2>
      <form className="form-inline" noValidate onSubmit={handleFormSubmit}>
        <div className="field">
          <label htmlFor="shipment-code">Shipment code</label>
          <input
            id="shipment-code"
            name="shipmentCode"
            type="text"
            value={shipmentCode}
            onChange={handleCodeChange}
            disabled={busy}
            required
          />
        </div>
        <button className="button" type="submit" disabled={busy}>
          Verify integrity
        </button>
      </form>
      <p className="message" data-tone={submissionTone(submission)} role="status" aria-live="polite">
        {submissionMessage(submission)}
      </p>
    </section>
  );
}

export default VerifyIntegrity;
