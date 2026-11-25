/**
 * Unit tests for integrity service.
 * Tests the API call shape without mocking fetchWithAuth (integration-style unit test).
 */

import { describe, expect, it, vi, beforeEach } from 'vitest';
import * as integrityService from './integrityService';
import * as httpClient from './httpClient';

vi.mock('./httpClient');

describe('verifyIntegrity', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('calls GET /api/shipments/{code}/integrity with authentication', async () => {
    const mockResponse = {
      ok: true,
      json: vi.fn().mockResolvedValue({
        shipmentCode: 'HBX-TEST-0001',
        intact: true,
        updatesChecked: 3,
        firstAlteredUpdate: null,
      }),
    };

    vi.mocked(httpClient.fetchWithAuth).mockResolvedValue(mockResponse as any);

    const result = await integrityService.verifyIntegrity('HBX-TEST-0001');

    expect(httpClient.fetchWithAuth).toHaveBeenCalledWith('/api/shipments/HBX-TEST-0001/integrity');
    expect(result).toEqual({
      shipmentCode: 'HBX-TEST-0001',
      intact: true,
      updatesChecked: 3,
      firstAlteredUpdate: null,
    });
  });

  it('throws with server message on HTTP error', async () => {
    const mockResponse = {
      ok: false,
      json: vi.fn().mockResolvedValue({
        error: 'SHIPMENT_NOT_FOUND',
        message: 'No shipment exists with that code',
      }),
    };

    vi.mocked(httpClient.fetchWithAuth).mockResolvedValue(mockResponse as any);

    await expect(integrityService.verifyIntegrity('HBX-UNKNOWN')).rejects.toThrow(
      'No shipment exists with that code',
    );
  });

  it('encodes the shipment code in the URL', async () => {
    const mockResponse = {
      ok: true,
      json: vi.fn().mockResolvedValue({
        shipmentCode: 'HBX-TEST/SPECIAL',
        intact: true,
        updatesChecked: 0,
        firstAlteredUpdate: null,
      }),
    };

    vi.mocked(httpClient.fetchWithAuth).mockResolvedValue(mockResponse as any);

    await integrityService.verifyIntegrity('HBX-TEST/SPECIAL');

    expect(httpClient.fetchWithAuth).toHaveBeenCalledWith('/api/shipments/HBX-TEST%2FSPECIAL/integrity');
  });
});
