/**
 * AssignedShipments: displays the list of shipments assigned to a ground partner.
 * Shows a table with shipment details, an empty message when applicable,
 * and a pagination button to load more shipments.
 */

import { useEffect, useState } from 'react';
import type { AssignedShipmentsResponse } from '../../services/shipmentService';

export interface AssignedShipmentsProps {
  onLoadAssignedShipments: (offset: number) => Promise<AssignedShipmentsResponse>;
  onSessionExpired: () => void;
}

type LoadState =
  | { kind: 'loading' }
  | { kind: 'loaded'; data: AssignedShipmentsResponse; allShipments: AssignedShipmentsResponse['shipments'] }
  | { kind: 'error'; message: string };

function formatDate(isoString: string): string {
  const date = new Date(isoString);
  const year = date.getUTCFullYear();
  const month = String(date.getUTCMonth() + 1).padStart(2, '0');
  const day = String(date.getUTCDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

function AssignedShipments({ onLoadAssignedShipments, onSessionExpired }: AssignedShipmentsProps) {
  const [state, setState] = useState<LoadState>({ kind: 'loading' });
  const [loadingMore, setLoadingMore] = useState(false);

  useEffect(() => {
    const loadInitial = async () => {
      try {
        const data = await onLoadAssignedShipments(0);
        setState({ kind: 'loaded', data, allShipments: data.shipments });
      } catch (error) {
        const message = error instanceof Error ? error.message : 'An error occurred';
        if (message.includes('Session expired') || message.includes('expired')) {
          onSessionExpired();
        } else {
          setState({ kind: 'error', message: 'The assigned shipments could not be loaded.' });
        }
      }
    };

    void loadInitial();
  }, [onLoadAssignedShipments, onSessionExpired]);

  const handleLoadMore = async () => {
    if (state.kind !== 'loaded' || state.data.nextOffset === null || loadingMore) {
      return;
    }

    setLoadingMore(true);
    try {
      const nextData = await onLoadAssignedShipments(state.data.nextOffset);
      setState({
        kind: 'loaded',
        data: nextData,
        allShipments: [...state.allShipments, ...nextData.shipments],
      });
    } catch (error) {
      const message = error instanceof Error ? error.message : 'An error occurred';
      if (message.includes('Session expired') || message.includes('expired')) {
        onSessionExpired();
      } else {
        setState({ kind: 'error', message: 'The assigned shipments could not be loaded.' });
      }
    } finally {
      setLoadingMore(false);
    }
  };

  return (
    <section className="panel" aria-labelledby="assigned-shipments-heading">
      <h2 id="assigned-shipments-heading">My assigned shipments</h2>

      {state.kind === 'loading' && <p className="message">Loading assigned shipments…</p>}

      {state.kind === 'error' && (
        <p className="message" data-tone="bad" role="alert">
          {state.message}
        </p>
      )}

      {state.kind === 'loaded' && state.allShipments.length === 0 && (
        <p className="message">{state.data.message || 'You have no assigned shipments'}</p>
      )}

      {state.kind === 'loaded' && state.allShipments.length > 0 && (
        <>
          <table className="data-table">
            <thead>
              <tr>
                <th>Code</th>
                <th>Stage</th>
                <th>Origin</th>
                <th>Destination</th>
                <th>Flight</th>
                <th>Weight</th>
                <th>Assigned</th>
              </tr>
            </thead>
            <tbody>
              {state.allShipments.map((shipment) => (
                <tr key={shipment.code}>
                  <td>{shipment.code}</td>
                  <td>{shipment.stage}</td>
                  <td>{shipment.origin}</td>
                  <td>{shipment.destination}</td>
                  <td>{shipment.flight}</td>
                  <td>{shipment.weightKg} kg</td>
                  <td>{formatDate(shipment.assignedAt)}</td>
                </tr>
              ))}
            </tbody>
          </table>

          {state.data.nextOffset !== null && (
            <button
              type="button"
              className="button button-secondary"
              onClick={() => void handleLoadMore()}
              disabled={loadingMore}
            >
              {loadingMore ? 'Loading…' : 'Show next 20'}
            </button>
          )}
        </>
      )}
    </section>
  );
}

export default AssignedShipments;
