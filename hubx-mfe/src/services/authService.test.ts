/**
 * Unit tests for authService.
 * Tests sign-in, token management, and authorization headers.
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';
import * as authService from './authService';

describe('authService', () => {
  beforeEach(() => {
    // Clear in-memory token before each test
    authService.clearToken();
    // Clear fetch mocks
    vi.clearAllMocks();
  });

  describe('signIn', () => {
    it('should return token and user on successful sign-in', async () => {
      const mockResponse = {
        token: 'mock.jwt.token',
        user: {
          email: 'operator@hubx.co',
          role: 'operator' as const,
        },
      };

      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: async () => mockResponse,
      });

      const result = await authService.signIn('operator@hubx.co', 'password123');

      expect(result).toEqual(mockResponse);
      expect(fetch).toHaveBeenCalledWith('/api/auth/sign-in', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          email: 'operator@hubx.co',
          password: 'password123',
        }),
      });
    });

    it('should throw error on 401 Unauthorized', async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 401,
        json: async () => ({
          error: 'INVALID_CREDENTIALS',
          message: 'Invalid credentials',
        }),
      });

      await expect(authService.signIn('wrong@hubx.co', 'wrong')).rejects.toThrow(
        'Invalid credentials'
      );
    });

    it('should throw error on 400 Bad Request', async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 400,
        json: async () => ({
          error: 'MISSING_REQUIRED_FIELDS',
          message: 'Email and password are required',
        }),
      });

      await expect(authService.signIn('', '')).rejects.toThrow(
        'Email and password are required'
      );
    });

    it('should handle network errors', async () => {
      globalThis.fetch = vi.fn().mockRejectedValue(new Error('Network error'));

      await expect(authService.signIn('test@hubx.co', 'pass')).rejects.toThrow();
    });

    it('should handle ground partner role with space', async () => {
      const mockResponse = {
        token: 'mock.jwt.token',
        user: {
          email: 'rapido@hubx.co',
          role: 'ground partner' as const,
        },
      };

      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: async () => mockResponse,
      });

      const result = await authService.signIn('rapido@hubx.co', 'password');

      expect(result.user.role).toBe('ground partner');
    });
  });

  describe('token management', () => {
    it('should store token in memory', () => {
      authService.setToken('test.token.value');
      expect(authService.getToken()).toBe('test.token.value');
    });

    it('should retrieve token from memory', () => {
      authService.setToken('stored.token');
      expect(authService.getToken()).toBe('stored.token');
    });

    it('should return null when no token exists', () => {
      expect(authService.getToken()).toBeNull();
    });

    it('should clear token from memory', () => {
      authService.setToken('test.token');
      authService.clearToken();
      expect(authService.getToken()).toBeNull();
    });
  });

  describe('getAuthHeaders', () => {
    it('should return Authorization header when token exists', () => {
      authService.setToken('valid.jwt.token');
      const headers = authService.getAuthHeaders();
      
      expect(headers).toEqual({
        Authorization: 'Bearer valid.jwt.token',
      });
    });

    it('should return empty object when no token exists', () => {
      const headers = authService.getAuthHeaders();
      expect(headers).toEqual({});
    });
  });
});
