/**
 * Unit tests for shipmentService.
 * Tests shipment creation through the gateway API.
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';
import * as shipmentService from './shipmentService';
import * as authService from './authService';

describe('shipmentService', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    authService.setToken('mock.jwt.token');
  });

  describe('createShipment', () => {
    it('should send POST request to /api/shipments with correct body', async () => {
      const mockResponse = {
        code: 'HBX-7F3A9C21',
        origin: 'Bogotá',
        destination: 'Miami',
        flight: 'AV-204',
        weightKg: 120,
        customer: {
          email: 'andina@hubx.co',
          name: 'Distribuciones Andina',
        },
        stage: 'Created',
        createdAt: '2026-11-24T15:04:05Z',
      };

      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 201,
        json: async () => mockResponse,
      });

      const input = {
        origin: 'Bogotá',
        destination: 'Miami',
        flight: 'AV-204',
        weightKg: 120,
        customerEmail: 'andina@hubx.co',
      };

      const result = await shipmentService.createShipment(input);

      expect(result).toEqual(mockResponse);
      expect(fetch).toHaveBeenCalledWith('/api/shipments', expect.objectContaining({
        method: 'POST',
        body: JSON.stringify(input),
      }));

      // Verify headers through Headers API
      const callArgs = (fetch as ReturnType<typeof vi.fn>).mock.calls[0];
      const headers = callArgs[1].headers as Headers;
      expect(headers.get('Content-Type')).toBe('application/json');
      expect(headers.get('Authorization')).toBe('Bearer mock.jwt.token');
    });

    it('should send weightKg as a JSON number', async () => {
      const mockResponse = {
        code: 'HBX-12345678',
        origin: 'Bogotá',
        destination: 'Miami',
        flight: 'AV-100',
        weightKg: 0.1,
        customer: {
          email: 'andina@hubx.co',
          name: 'Distribuciones Andina',
        },
        stage: 'Created',
        createdAt: '2026-11-24T15:04:05Z',
      };

      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 201,
        json: async () => mockResponse,
      });

      const input = {
        origin: 'Bogotá',
        destination: 'Miami',
        flight: 'AV-100',
        weightKg: 0.1,
        customerEmail: 'andina@hubx.co',
      };

      await shipmentService.createShipment(input);

      const callBody = JSON.parse(
        (fetch as ReturnType<typeof vi.fn>).mock.calls[0][1].body
      );
      expect(typeof callBody.weightKg).toBe('number');
      expect(callBody.weightKg).toBe(0.1);
    });

    it('should throw error with message on 400 MISSING_REQUIRED_FIELDS', async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 400,
        json: async () => ({
          error: 'MISSING_REQUIRED_FIELDS',
          message: 'Required data is missing',
        }),
      });

      const input = {
        origin: '',
        destination: 'Miami',
        flight: 'AV-100',
        weightKg: 100,
        customerEmail: 'andina@hubx.co',
      };

      await expect(shipmentService.createShipment(input)).rejects.toThrow(
        'Required data is missing'
      );
    });

    it('should throw error with message on 400 SAME_ORIGIN_DESTINATION', async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 400,
        json: async () => ({
          error: 'SAME_ORIGIN_DESTINATION',
          message: 'The origin and the destination must be different',
        }),
      });

      const input = {
        origin: 'Bogotá',
        destination: 'Bogotá',
        flight: 'AV-100',
        weightKg: 100,
        customerEmail: 'andina@hubx.co',
      };

      await expect(shipmentService.createShipment(input)).rejects.toThrow(
        'The origin and the destination must be different'
      );
    });

    it('should throw error with message on 400 WEIGHT_OUT_OF_RANGE', async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 400,
        json: async () => ({
          error: 'WEIGHT_OUT_OF_RANGE',
          message: 'The weight must be between 0.1 and 5000 kg',
        }),
      });

      const input = {
        origin: 'Bogotá',
        destination: 'Miami',
        flight: 'AV-100',
        weightKg: 0,
        customerEmail: 'andina@hubx.co',
      };

      await expect(shipmentService.createShipment(input)).rejects.toThrow(
        'The weight must be between 0.1 and 5000 kg'
      );
    });

    it('should throw error with message on 403 FORBIDDEN', async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 403,
        json: async () => ({
          error: 'FORBIDDEN',
          message: 'You do not have permission to create shipments',
        }),
      });

      const input = {
        origin: 'Bogotá',
        destination: 'Miami',
        flight: 'AV-100',
        weightKg: 100,
        customerEmail: 'andina@hubx.co',
      };

      await expect(shipmentService.createShipment(input)).rejects.toThrow(
        'You do not have permission to create shipments'
      );
    });

    it('should throw error with message on 422 CUSTOMER_NOT_FOUND', async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 422,
        json: async () => ({
          error: 'CUSTOMER_NOT_FOUND',
          message: 'The customer indicated does not exist',
        }),
      });

      const input = {
        origin: 'Bogotá',
        destination: 'Miami',
        flight: 'AV-100',
        weightKg: 100,
        customerEmail: 'nobody@hubx.co',
      };

      await expect(shipmentService.createShipment(input)).rejects.toThrow(
        'The customer indicated does not exist'
      );
    });

    it('should throw error with message on 422 CUSTOMER_NOT_CUSTOMER_ROLE', async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 422,
        json: async () => ({
          error: 'CUSTOMER_NOT_CUSTOMER_ROLE',
          message: 'The customer indicated does not have the customer role',
        }),
      });

      const input = {
        origin: 'Bogotá',
        destination: 'Miami',
        flight: 'AV-100',
        weightKg: 100,
        customerEmail: 'rapido@hubx.co',
      };

      await expect(shipmentService.createShipment(input)).rejects.toThrow(
        'The customer indicated does not have the customer role'
      );
    });
  });
});
