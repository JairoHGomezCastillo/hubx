/**
 * Unit tests for stageCountService.
 * Tests stage count retrieval through the gateway API.
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';
import * as stageCountService from './stageCountService';
import * as authService from './authService';

describe('stageCountService', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    authService.setToken('mock.jwt.token');
  });

  describe('getStageCounts', () => {
    it('should send GET request to /api/shipments/stage-counts', async () => {
      const mockResponse = {
        stages: [
          { stage: 'Created', count: 2 },
          { stage: 'In flight', count: 3 },
          { stage: 'At destination airport', count: 1 },
          { stage: 'Assigned to partner', count: 0 },
          { stage: 'Out for delivery', count: 2 },
          { stage: 'Delivered', count: 5 },
        ],
        total: 13,
      };

      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => mockResponse,
      });

      const result = await stageCountService.getStageCounts();

      expect(result).toEqual({ kind: 'success', data: mockResponse });
      expect(fetch).toHaveBeenCalledWith('/api/shipments/stage-counts', expect.objectContaining({
        method: 'GET',
      }));

      const callArgs = (fetch as ReturnType<typeof vi.fn>).mock.calls[0];
      const headers = callArgs[1].headers as Headers;
      expect(headers.get('Authorization')).toBe('Bearer mock.jwt.token');
    });

    it('should return success with six stages in journey order', async () => {
      const mockResponse = {
        stages: [
          { stage: 'Created', count: 0 },
          { stage: 'In flight', count: 0 },
          { stage: 'At destination airport', count: 0 },
          { stage: 'Assigned to partner', count: 0 },
          { stage: 'Out for delivery', count: 0 },
          { stage: 'Delivered', count: 0 },
        ],
        total: 0,
      };

      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => mockResponse,
      });

      const result = await stageCountService.getStageCounts();

      expect(result.kind).toBe('success');
      if (result.kind === 'success') {
        expect(result.data.stages).toHaveLength(6);
        expect(result.data.stages[0].stage).toBe('Created');
        expect(result.data.stages[5].stage).toBe('Delivered');
        expect(result.data.total).toBe(0);
      }
    });

    it('should return forbidden on 403 response', async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 403,
        json: async () => ({
          error: 'FORBIDDEN',
          message: 'You do not have permission to see that information',
        }),
      });

      const result = await stageCountService.getStageCounts();

      expect(result).toEqual({
        kind: 'forbidden',
        message: 'You do not have permission to see that information',
      });
    });

    it('should return session-expired on 401 when token is cleared', async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 401,
        json: async () => ({
          error: { code: 'SESSION_EXPIRED', message: 'Session expired' },
        }),
      });

      const result = await stageCountService.getStageCounts();

      expect(result).toEqual({ kind: 'session-expired' });
      expect(authService.getToken()).toBeNull();
    });

    it('should return error on network failure', async () => {
      globalThis.fetch = vi.fn().mockRejectedValue(new Error('Network error'));

      const result = await stageCountService.getStageCounts();

      expect(result.kind).toBe('error');
    });

    it('should return error on 500 response', async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
        json: async () => ({
          error: 'INTERNAL_SERVER_ERROR',
          message: 'An unexpected error occurred',
        }),
      });

      const result = await stageCountService.getStageCounts();

      expect(result.kind).toBe('error');
    });
  });
});
