/**
 * Unit tests for shipmentService.
 */

import { describe, expect, it, vi, beforeEach } from 'vitest';
import { createShipment, listAssignedShipments } from './shipmentService';
import * as authService from './authService';
import * as httpClient from './httpClient';

describe('shipmentService', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    authService.clearToken();
  });

  describe('createShipment', () => {
    it('should send POST request to /api/shipments with correct body', async () => {
      authService.setToken('test.jwt.token');
      
      const input = {
        origin: 'Bogotá',
        destination: 'Miami',
        flight: 'AV-204',
        weightKg: 120,
        customerEmail: 'ana@hubx.co',
      };

      const mockResponse = {
        code: 'HBX-12345678',
        origin: 'Bogotá',
        destination: 'Miami',
        flight: 'AV-204',
        weightKg: 120,
        customer: {
          email: 'ana@hubx.co',
          name: 'Ana',
        },
        stage: 'Created',
        createdAt: '2026-11-24T10:00:00Z',
      };

      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: async () => mockResponse,
      });

      const result = await createShipment(input);

      expect(result).toEqual(mockResponse);
      expect(fetch).toHaveBeenCalledWith('/api/shipments', expect.objectContaining({
        method: 'POST',
        body: JSON.stringify(input),
      }));
      
      // Verify headers separately since they're passed as a Headers object
      const callArgs = (fetch as any).mock.calls[0];
      const headers = callArgs[1].headers as Headers;
      expect(headers.get('Content-Type')).toBe('application/json');
      expect(headers.get('Authorization')).toBe('Bearer test.jwt.token');
    });

    it('should send weightKg as a JSON number', async () => {
      authService.setToken('test.jwt.token');

      const input = {
        origin: 'Bogotá',
        destination: 'Miami',
        flight: 'AV-204',
        weightKg: 150.5,
        customerEmail: 'ana@hubx.co',
      };

      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          code: 'HBX-TEST',
          ...input,
          customer: { email: 'ana@hubx.co', name: 'Ana' },
          stage: 'Created',
          createdAt: '2026-11-24T10:00:00Z',
        }),
      });

      await createShipment(input);

      const callBody = (fetch as any).mock.calls[0][1].body;
      const parsed = JSON.parse(callBody);
      expect(typeof parsed.weightKg).toBe('number');
      expect(parsed.weightKg).toBe(150.5);
    });

    it('should throw error on 400 MISSING_REQUIRED_FIELDS', async () => {
      authService.setToken('test.jwt.token');

      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 400,
        json: async () => ({
          error: 'MISSING_REQUIRED_FIELDS',
          message: 'All fields are required',
        }),
      });

      await expect(createShipment({
        origin: '',
        destination: 'Miami',
        flight: 'AV-204',
        weightKg: 100,
        customerEmail: 'test@hubx.co',
      })).rejects.toThrow('All fields are required');
    });

    it('should throw error on 400 SAME_ORIGIN_DESTINATION', async () => {
      authService.setToken('test.jwt.token');

      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 400,
        json: async () => ({
          error: 'SAME_ORIGIN_DESTINATION',
          message: 'Origin and destination must be different',
        }),
      });

      await expect(createShipment({
        origin: 'Miami',
        destination: 'Miami',
        flight: 'AV-204',
        weightKg: 100,
        customerEmail: 'test@hubx.co',
      })).rejects.toThrow('Origin and destination must be different');
    });

    it('should throw error on 400 WEIGHT_OUT_OF_RANGE', async () => {
      authService.setToken('test.jwt.token');

      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 400,
        json: async () => ({
          error: 'WEIGHT_OUT_OF_RANGE',
          message: 'Weight must be between 0.1 and 5000 kg',
        }),
      });

      await expect(createShipment({
        origin: 'Bogotá',
        destination: 'Miami',
        flight: 'AV-204',
        weightKg: 0.05,
        customerEmail: 'test@hubx.co',
      })).rejects.toThrow('Weight must be between 0.1 and 5000 kg');
    });

    it('should throw error on 403 FORBIDDEN', async () => {
      authService.setToken('ground.partner.token');

      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 403,
        json: async () => ({
          error: 'FORBIDDEN',
          message: 'Only operators can create shipments',
        }),
      });

      await expect(createShipment({
        origin: 'Bogotá',
        destination: 'Miami',
        flight: 'AV-204',
        weightKg: 100,
        customerEmail: 'test@hubx.co',
      })).rejects.toThrow('Only operators can create shipments');
    });

    it('should throw error on 422 CUSTOMER_NOT_FOUND', async () => {
      authService.setToken('test.jwt.token');

      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 422,
        json: async () => ({
          error: 'CUSTOMER_NOT_FOUND',
          message: 'Customer not found',
        }),
      });

      await expect(createShipment({
        origin: 'Bogotá',
        destination: 'Miami',
        flight: 'AV-204',
        weightKg: 100,
        customerEmail: 'unknown@example.com',
      })).rejects.toThrow('Customer not found');
    });

    it('should throw error on 422 CUSTOMER_NOT_CUSTOMER_ROLE', async () => {
      authService.setToken('test.jwt.token');

      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 422,
        json: async () => ({
          error: 'CUSTOMER_NOT_CUSTOMER_ROLE',
          message: 'The indicated user is not a customer',
        }),
      });

      await expect(createShipment({
        origin: 'Bogotá',
        destination: 'Miami',
        flight: 'AV-204',
        weightKg: 100,
        customerEmail: 'operator@hubx.co',
      })).rejects.toThrow('The indicated user is not a customer');
    });
  });

  describe('listAssignedShipments', () => {
    it('fetches assigned shipments with default offset 0', async () => {
      const mockResponse = {
        shipments: [
          {
            code: 'HBX-TEST-001',
            stage: 'Assigned to partner',
            origin: 'Bogotá',
            destination: 'Miami',
            flight: 'AV-204',
            weightKg: 120,
            assignedAt: '2026-11-24T10:15:30.123456Z',
          },
        ],
        nextOffset: null,
        message: null,
      };

      const getJsonSpy = vi.spyOn(httpClient, 'getJson').mockResolvedValue(mockResponse);

      const result = await listAssignedShipments();

      expect(getJsonSpy).toHaveBeenCalledWith('/api/assigned-shipments?offset=0');
      expect(result).toEqual(mockResponse);

      getJsonSpy.mockRestore();
    });

    it('fetches assigned shipments with specified offset', async () => {
      const mockResponse = {
        shipments: [],
        nextOffset: null,
        message: 'You have no assigned shipments',
      };

      const getJsonSpy = vi.spyOn(httpClient, 'getJson').mockResolvedValue(mockResponse);

      const result = await listAssignedShipments(20);

      expect(getJsonSpy).toHaveBeenCalledWith('/api/assigned-shipments?offset=20');
      expect(result).toEqual(mockResponse);

      getJsonSpy.mockRestore();
    });

    it('handles empty list with message', async () => {
      const mockResponse = {
        shipments: [],
        nextOffset: null,
        message: 'You have no assigned shipments',
      };

      const getJsonSpy = vi.spyOn(httpClient, 'getJson').mockResolvedValue(mockResponse);

      const result = await listAssignedShipments();

      expect(result.shipments).toHaveLength(0);
      expect(result.message).toBe('You have no assigned shipments');
      expect(result.nextOffset).toBeNull();

      getJsonSpy.mockRestore();
    });

    it('handles pagination with nextOffset', async () => {
      const mockResponse = {
        shipments: Array.from({ length: 20 }, (_, i) => ({
          code: `HBX-TEST-${String(i).padStart(3, '0')}`,
          stage: 'Out for delivery',
          origin: 'Bogotá',
          destination: 'Miami',
          flight: 'AV-204',
          weightKg: 100 + i,
          assignedAt: '2026-11-24T10:15:30Z',
        })),
        nextOffset: 20,
        message: null,
      };

      const getJsonSpy = vi.spyOn(httpClient, 'getJson').mockResolvedValue(mockResponse);

      const result = await listAssignedShipments();

      expect(result.shipments).toHaveLength(20);
      expect(result.nextOffset).toBe(20);
      expect(result.message).toBeNull();

      getJsonSpy.mockRestore();
    });
  });
});
