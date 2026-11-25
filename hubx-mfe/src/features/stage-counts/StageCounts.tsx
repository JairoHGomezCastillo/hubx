/**
 * StageCounts: view the shipment count for all six journey stages.
 * Operator-only feature from HU-09.
 */

import { useState } from 'react';
import type { StageCountOutcome } from '../../services/stageCountService';

interface StageCountsProps {
  onGetStageCounts: () => Promise<StageCountOutcome>;
  onSessionExpired: () => void;
}

type ViewState =
  | { kind: 'idle' }
  | { kind: 'loading' }
  | { kind: 'success'; stages: Array<{ stage: string; count: number }>; total: number }
  | { kind: 'forbidden'; message: string }
  | { kind: 'error' };

export default function StageCounts({ onGetStageCounts, onSessionExpired }: StageCountsProps) {
  const [state, setState] = useState<ViewState>({ kind: 'idle' });

  const handleLoad = async () => {
    setState({ kind: 'loading' });
    const outcome = await onGetStageCounts();

    if (outcome.kind === 'success') {
      setState({ kind: 'success', stages: outcome.data.stages, total: outcome.data.total });
    } else if (outcome.kind === 'forbidden') {
      setState({ kind: 'forbidden', message: outcome.message });
    } else if (outcome.kind === 'session-expired') {
      onSessionExpired();
    } else {
      setState({ kind: 'error' });
    }
  };

  return (
    <div className="stage-counts">
      <h2>Shipment count by stage</h2>

      {state.kind === 'idle' && (
        <button type="button" onClick={handleLoad}>
          Load stage counts
        </button>
      )}

      {state.kind === 'loading' && <p>Loading stage counts...</p>}

      {state.kind === 'success' && (
        <section aria-label="Stage counts">
          <table>
            <thead>
              <tr>
                <th>Stage</th>
                <th>Count</th>
              </tr>
            </thead>
            <tbody>
              {state.stages.map((stageCount) => (
                <tr key={stageCount.stage} aria-label={`${stageCount.stage} ${stageCount.count}`}>
                  <td>{stageCount.stage}</td>
                  <td>{stageCount.count}</td>
                </tr>
              ))}
            </tbody>
          </table>
          <p>
            <strong>Total: {state.total} shipments</strong>
          </p>
        </section>
      )}

      {state.kind === 'forbidden' && (
        <div className="error" role="alert">
          {state.message}
        </div>
      )}

      {state.kind === 'error' && (
        <div className="error" role="alert">
          The stage counts could not be loaded. Please try again.
        </div>
      )}
    </div>
  );
}
