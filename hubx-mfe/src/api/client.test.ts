/** Unit tests for the HU-03 /api client: gateway paths, JWT and explicit states. */

import { beforeEach, describe, expect, it, vi } from 'vitest';
import * as authService from '../services/authService';
import { listGroundPartners, recordUpdate } from './client';

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}

const UPDATE = {
  id: 1,
  shipmentCode: 'HBX-SEED-0001',
  sequence: 1,
  stage: 'In flight',
  responsibleParty: {
    id: '00000000-0000-4000-8000-000000000001',
    name: 'Laura Operator',
    role: 'operator',
  },
  assignedPartner: null,
  recordedAt: '2026-11-24T10:15:30Z',
  previousFingerprint: null,
  fingerprint: 'b'.repeat(64),
};

describe('HU-03 /api client', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    sessionStorage.clear();
    authService.setToken('mock.jwt.token');
  });

  it('lists ground partners from /api with the in-memory JWT', async () => {
    const partners = [
      { id: 'partner-1', name: 'Logística Express' },
      { id: 'partner-2', name: 'Transportes Rápido' },
    ];
    globalThis.fetch = vi.fn().mockResolvedValue(jsonResponse(200, partners));

    const outcome = await listGroundPartners();

    expect(outcome).toEqual({ kind: 'success', partners });
    expect(fetch).toHaveBeenCalledWith('/api/ground-partners', expect.objectContaining({ method: 'GET' }));
    const [, init] = vi.mocked(fetch).mock.calls[0];
    expect(new Headers(init?.headers).get('Authorization')).toBe('Bearer mock.jwt.token');
  });

  it('distinguishes an empty partner list from a failed one', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue(jsonResponse(200, []));
    expect(await listGroundPartners()).toEqual({ kind: 'empty' });

    globalThis.fetch = vi.fn().mockResolvedValue(
      jsonResponse(500, { error: { code: 'INTERNAL_ERROR', message: 'The partner list could not be loaded.' } }),
    );
    expect(await listGroundPartners()).toEqual({
      kind: 'httpError',
      status: 500,
      code: 'INTERNAL_ERROR',
      message: 'The partner list could not be loaded.',
    });
  });

  it('reports a missing response without inventing an HTTP status', async () => {
    globalThis.fetch = vi.fn().mockRejectedValue(new Error('Network error'));

    expect(await listGroundPartners()).toEqual({ kind: 'noResponse' });
    expect(authService.getToken()).toBe('mock.jwt.token');
  });

  it('posts the update to the gateway and returns the 201 body', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue(jsonResponse(201, UPDATE));

    const outcome = await recordUpdate('HBX-SEED-0001', { stage: 'In flight', assignedPartnerId: null });

    expect(outcome).toEqual({ kind: 'success', update: UPDATE });
    expect(fetch).toHaveBeenCalledWith(
      '/api/shipments/HBX-SEED-0001/updates',
      expect.objectContaining({ method: 'POST' }),
    );
    const [, init] = vi.mocked(fetch).mock.calls[0];
    expect(JSON.parse(String(init?.body))).toEqual({ stage: 'In flight', assignedPartnerId: null });
  });

  it('preserves the contract error code and message on a 400', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue(
      jsonResponse(400, {
        error: {
          code: 'PARTNER_REQUIRED',
          message: 'You must name the ground partner responsible for the delivery.',
        },
      }),
    );

    expect(
      await recordUpdate('HBX-SEED-0001', { stage: 'Assigned to partner', assignedPartnerId: null }),
    ).toEqual({
      kind: 'httpError',
      status: 400,
      code: 'PARTNER_REQUIRED',
      message: 'You must name the ground partner responsible for the delivery.',
    });
  });

  it('maps a 401 to SESSION_EXPIRED after the shared wrapper clears the session', async () => {
    globalThis.fetch = vi.fn().mockImplementation(async () => {
      authService.clearToken();
      return jsonResponse(401, {
        error: { code: 'SESSION_EXPIRED', message: 'Session expired; sign in again.' },
      });
    });

    expect(await recordUpdate('HBX-SEED-0001', { stage: 'In flight', assignedPartnerId: null })).toEqual({
      kind: 'httpError',
      status: 401,
      code: 'SESSION_EXPIRED',
      message: 'Session expired; sign in again.',
    });
    expect(sessionStorage.getItem('session_expired_message')).toBe('Session expired. Please sign in again.');
    expect(window.location.pathname).toBe('/sign-in');
  });

  it('reports a network failure on the update as noResponse', async () => {
    globalThis.fetch = vi.fn().mockRejectedValue(new Error('Network error'));

    expect(await recordUpdate('HBX-SEED-0001', { stage: 'In flight', assignedPartnerId: null })).toEqual({
      kind: 'noResponse',
    });
  });
});
