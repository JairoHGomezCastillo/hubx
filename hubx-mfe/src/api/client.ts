/**
 * HU-03 /api client.
 *
 * Every request goes through the gateway's `/api` prefix. The JWT stays in
 * memory: we reuse the HU-01 `fetchWithAuth` wrapper, which reads the token
 * from `authService`, adds the Authorization header and handles a 401 by
 * clearing the token, navigating to sign-in with the history API and storing
 * the session-expired message. No direct call to hubx-api, no localStorage.
 */

import * as authService from '../services/authService';
import { fetchWithAuth } from '../services/httpClient';
import type {
  GroundPartner,
  RecordedUpdate,
  RecordUpdateBody,
  RecordUpdateOutcome,
} from '../features/shipment-updates/RecordUpdate';

/** Explicit states of the ground-partner lookup. */
export type GroundPartnersOutcome =
  | { kind: 'success'; partners: GroundPartner[] }
  | { kind: 'empty' }
  | { kind: 'httpError'; status: number; code: string; message: string }
  | { kind: 'noResponse' }
  | { kind: 'unauthorized' };

interface ErrorBody {
  error?: { code?: string; message?: string };
}

type RequestResult =
  | { response: Response }
  | { outcome: Extract<GroundPartnersOutcome, { kind: 'noResponse' | 'unauthorized' }> };

const SESSION_EXPIRED = 'Session expired; sign in again.';

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

function isGroundPartner(value: unknown): value is GroundPartner {
  return isRecord(value) && typeof value.id === 'string' && typeof value.name === 'string';
}

function isRecordedUpdate(value: unknown): value is RecordedUpdate {
  return (
    isRecord(value) &&
    typeof value.id === 'number' &&
    typeof value.shipmentCode === 'string' &&
    typeof value.sequence === 'number' &&
    typeof value.stage === 'string' &&
    isRecord(value.responsibleParty) &&
    typeof value.responsibleParty.name === 'string' &&
    typeof value.recordedAt === 'string' &&
    typeof value.fingerprint === 'string'
  );
}

async function errorFields(response: Response, fallback: string): Promise<{ code: string; message: string }> {
  const body = await response.json().catch(() => null) as ErrorBody | null;
  return {
    code: body?.error?.code ?? '',
    message: body?.error?.message ?? fallback,
  };
}

/**
 * fetchWithAuth throws only after handling a 401, where it clears the in-memory
 * token. Detecting that transition lets us distinguish an expired session from
 * a network failure without duplicating HU-01's session handling.
 */
async function request(method: 'GET' | 'POST', path: string, body?: unknown): Promise<RequestResult> {
  const hadToken = authService.getToken() !== null;
  try {
    const response = await fetchWithAuth(path, {
      method,
      ...(body === undefined
        ? {}
        : {
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body),
          }),
    });
    return { response };
  } catch {
    if (hadToken && authService.getToken() === null) {
      return { outcome: { kind: 'unauthorized' } };
    }
    return { outcome: { kind: 'noResponse' } };
  }
}

/** GET /api/ground-partners — names are labels, ids are form values. */
export async function listGroundPartners(): Promise<GroundPartnersOutcome> {
  const result = await request('GET', '/api/ground-partners');
  if ('outcome' in result) {
    return result.outcome;
  }

  const { response } = result;
  if (!response.ok) {
    const { code, message } = await errorFields(response, 'The ground partner list could not be loaded.');
    return { kind: 'httpError', status: response.status, code, message };
  }

  try {
    const data: unknown = await response.json();
    if (!Array.isArray(data) || !data.every(isGroundPartner)) {
      return { kind: 'noResponse' };
    }
    return data.length === 0 ? { kind: 'empty' } : { kind: 'success', partners: data };
  } catch {
    return { kind: 'noResponse' };
  }
}

/** POST /api/shipments/{code}/updates — the contract's success and error shape. */
export async function recordUpdate(shipmentCode: string, body: RecordUpdateBody): Promise<RecordUpdateOutcome> {
  const path = `/api/shipments/${encodeURIComponent(shipmentCode)}/updates`;
  const result = await request('POST', path, body);
  if ('outcome' in result) {
    return result.outcome.kind === 'unauthorized'
      ? { kind: 'httpError', status: 401, code: 'SESSION_EXPIRED', message: SESSION_EXPIRED }
      : { kind: 'noResponse' };
  }

  const { response } = result;
  if (!response.ok) {
    const { code, message } = await errorFields(response, 'The update could not be recorded.');
    return { kind: 'httpError', status: response.status, code, message };
  }

  try {
    const data: unknown = await response.json();
    if (response.status !== 201 || !isRecordedUpdate(data)) {
      return { kind: 'noResponse' };
    }
    return { kind: 'success', update: data };
  } catch {
    return { kind: 'noResponse' };
  }
}
