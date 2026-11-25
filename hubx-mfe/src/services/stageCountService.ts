/**
 * Stage count service for viewing shipment counts by stage.
 * Calls GET /api/shipments/stage-counts through the gateway.
 */

import { fetchWithAuth } from './httpClient';
import * as authService from './authService';

export interface StageCount {
  stage: string;
  count: number;
}

export interface StageCountReport {
  stages: StageCount[];
  total: number;
}

export type StageCountOutcome =
  | { kind: 'success'; data: StageCountReport }
  | { kind: 'forbidden'; message: string }
  | { kind: 'session-expired' }
  | { kind: 'error' };

/**
 * Get the shipment count for all six stages (operator-only).
 * 
 * @returns Outcome with six stages in journey order, or error/forbidden/session-expired
 */
export async function getStageCounts(): Promise<StageCountOutcome> {
  const hadToken = authService.getToken() !== null;

  try {
    const response = await fetchWithAuth('/api/shipments/stage-counts', {
      method: 'GET',
    });

    if (!response.ok) {
      if (response.status === 403) {
        const error = await response.json().catch(() => ({ message: 'You do not have permission to see that information' }));
        return { kind: 'forbidden', message: error.message || 'You do not have permission to see that information' };
      }
      return { kind: 'error' };
    }

    const data: StageCountReport = await response.json();
    return { kind: 'success', data };
  } catch {
    // fetchWithAuth throws after handling 401 by clearing the token
    if (hadToken && authService.getToken() === null) {
      return { kind: 'session-expired' };
    }
    return { kind: 'error' };
  }
}
