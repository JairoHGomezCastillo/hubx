import { useState } from 'react';
import type { ChangeEvent, FormEvent } from 'react';
import type { TraceabilityData, TraceabilityOutcome } from '../../api/client';
import { StageTrack } from './StageTrack';

export interface ViewTraceabilityProps {
  onViewTraceability: (code: string) => Promise<TraceabilityOutcome>;
  onSessionExpired: () => void;
}

type ViewState =
  | { kind: 'idle' }
  | { kind: 'loading' }
  | { kind: 'success'; data: TraceabilityData }
  | { kind: 'not-found' }
  | { kind: 'error' };

function formatDate(isoString: string): string {
  // The @e2e asserts /\d{4}-\d{2}-\d{2}/, 'In flight' and 'Laura Operator' in the first item.
  // Plan: Date MUST contain an ISO-style date YYYY-MM-DD and state the time zone.
  const date = new Date(isoString);
  const year = date.getUTCFullYear();
  const month = String(date.getUTCMonth() + 1).padStart(2, '0');
  const day = String(date.getUTCDate()).padStart(2, '0');
  const hours = String(date.getUTCHours()).padStart(2, '0');
  const minutes = String(date.getUTCMinutes()).padStart(2, '0');
  return `${year}-${month}-${day} ${hours}:${minutes} UTC`;
}

function ViewTraceability({ onViewTraceability, onSessionExpired }: ViewTraceabilityProps) {
  const [shipmentCode, setShipmentCode] = useState('');
  const [state, setState] = useState<ViewState>({ kind: 'idle' });

  const busy = state.kind === 'loading';

  const handleCodeChange = (event: ChangeEvent<HTMLInputElement>) => {
    setShipmentCode(event.target.value);
  };

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (busy) {
      return;
    }

    const code = shipmentCode.trim();
    if (!code) {
      return;
    }

    setState({ kind: 'loading' });
    try {
      const outcome = await onViewTraceability(code);
      switch (outcome.kind) {
        case 'ok':
          setState({ kind: 'success', data: outcome.data });
          break;
        case 'not-found':
          setState({ kind: 'not-found' });
          break;
        case 'session-expired':
          onSessionExpired();
          break;
        case 'error':
          setState({ kind: 'error' });
          break;
      }
    } catch {
      setState({ kind: 'error' });
    }
  };

  const handleFormSubmit = (event: FormEvent<HTMLFormElement>) => {
    void handleSubmit(event);
  };

  return (
    <section className="panel" aria-labelledby="view-traceability-heading">
      <h2 id="view-traceability-heading">View traceability</h2>
      <form className="form-inline" noValidate onSubmit={handleFormSubmit}>
        <div className="field">
          <label htmlFor="traceability-shipment-code">Shipment code</label>
          <input
            id="traceability-shipment-code"
            name="shipmentCode"
            type="text"
            value={shipmentCode}
            onChange={handleCodeChange}
            disabled={busy}
            required
          />
        </div>
        <button className="button" type="submit" disabled={busy}>
          View traceability
        </button>
      </form>

      {state.kind === 'loading' && <p className="message">Loading traceability…</p>}

      {state.kind === 'success' && (
        <div className="traceability-result">
          <h3>Traceability of {state.data.shipmentCode}</h3>
          <p>Current stage: {state.data.stage}</p>
          <StageTrack
            currentStage={state.data.stage}
            recordedStages={state.data.updates.map((update) => update.stage)}
          />
          {state.data.updates.length === 0 ? (
            <p className="message">This shipment has no updates yet.</p>
          ) : (
            <ol className="updates" aria-label="Updates">
              {state.data.updates.map((update) => (
                <li key={update.sequence}>
                  <div className="update-stage">Stage: {update.stage}</div>
                  <div>
                    Responsible party: {update.responsibleParty.name} ({update.responsibleParty.role})
                  </div>
                  {update.assignedPartner && (
                    <div>Assigned partner: {update.assignedPartner.name}</div>
                  )}
                  <div>Date: {formatDate(update.recordedAt)}</div>
                </li>
              ))}
            </ol>
          )}
        </div>
      )}

      {state.kind === 'not-found' && (
        <p className="message" data-tone="bad" role="alert">No shipment exists with that code.</p>
      )}

      {state.kind === 'error' && (
        <p className="message" data-tone="bad" role="alert">The traceability could not be retrieved.</p>
      )}
    </section>
  );
}

export default ViewTraceability;
