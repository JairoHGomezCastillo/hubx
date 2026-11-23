import { test, expect } from '@playwright/test';
import * as crypto from 'crypto';

/**
 * HU-01: Sign in
 * Integration tests for authentication through the gateway.
 * All tests go through POST /api/auth/sign-in; session expiration tests use a forged expired JWT.
 */

// Demo users (from migration 0001_baseline_schema.sql)
const DEMO_PASSWORD = 'Hubx-Demo-2026';
const OPERATOR_EMAIL = 'operator@hubx.co';
const PARTNER_EMAIL = 'rapido@hubx.co';
const CUSTOMER_EMAIL = 'ana@hubx.co';

test.describe('HU01: Sign in', () => {
  /**
   * T001: Sign in as operator
   * Scenario: operator signs in with correct credentials
   * Expected: 200 OK, JWT token with role=operator
   */
  test('HU01_sign_in_operator', async ({ request }) => {
    const res = await request.post('/api/auth/sign-in', {
      data: {
        email: OPERATOR_EMAIL,
        password: DEMO_PASSWORD,
      },
    });

    expect(res.status()).toBe(200);
    const body = await res.json();
    
    expect(body).toHaveProperty('token');
    expect(body).toHaveProperty('user');
    expect(body.user.email).toBe(OPERATOR_EMAIL);
    expect(body.user.role).toBe('operator');
    
    // Verify JWT structure (base64url-encoded payload)
    const payload = JSON.parse(
      Buffer.from(body.token.split('.')[1], 'base64url').toString()
    );
    expect(payload.email).toBe(OPERATOR_EMAIL);
    expect(payload.role).toBe('operator');
    expect(payload).toHaveProperty('sub');
    expect(payload).toHaveProperty('iat');
    expect(payload).toHaveProperty('exp');
  });

  /**
   * T002: Sign in as ground partner
   * Scenario: ground partner signs in with correct credentials
   * Expected: 200 OK, JWT token with role="ground partner" (with space)
   */
  test('HU01_sign_in_partner', async ({ request }) => {
    const res = await request.post('/api/auth/sign-in', {
      data: {
        email: PARTNER_EMAIL,
        password: DEMO_PASSWORD,
      },
    });

    expect(res.status()).toBe(200);
    const body = await res.json();
    
    expect(body).toHaveProperty('token');
    expect(body).toHaveProperty('user');
    expect(body.user.email).toBe(PARTNER_EMAIL);
    expect(body.user.role).toBe('ground partner'); // With space per implement clarification
    
    const payload = JSON.parse(
      Buffer.from(body.token.split('.')[1], 'base64url').toString()
    );
    expect(payload.role).toBe('ground partner');
  });

  /**
   * T003: Sign in as customer
   * Scenario: customer signs in with correct credentials
   * Expected: 200 OK, JWT token with role=customer
   */
  test('HU01_sign_in_customer', async ({ request }) => {
    const res = await request.post('/api/auth/sign-in', {
      data: {
        email: CUSTOMER_EMAIL,
        password: DEMO_PASSWORD,
      },
    });

    expect(res.status()).toBe(200);
    const body = await res.json();
    
    expect(body).toHaveProperty('token');
    expect(body).toHaveProperty('user');
    expect(body.user.email).toBe(CUSTOMER_EMAIL);
    expect(body.user.role).toBe('customer');
    
    const payload = JSON.parse(
      Buffer.from(body.token.split('.')[1], 'base64url').toString()
    );
    expect(payload.role).toBe('customer');
  });

  /**
   * T004: Email normalization
   * Scenario: customer signs in with capitalized email and surrounding spaces
   * Expected: 200 OK (normalized to ana@hubx.co)
   */
  test('HU01_email_normalization', async ({ request }) => {
    const res = await request.post('/api/auth/sign-in', {
      data: {
        email: ' ANA@hubx.co ',
        password: DEMO_PASSWORD,
      },
    });

    expect(res.status()).toBe(200);
    const body = await res.json();
    
    expect(body.user.email).toBe(CUSTOMER_EMAIL); // Normalized
    expect(body.user.role).toBe('customer');
  });

  /**
   * T005: Security - non-revealing error messages
   * Scenario: sign in with wrong password AND with unknown email
   * Expected: Both return 401 with identical generic message "Invalid credentials"
   */
  test('HU01_security_non_revealing_errors', async ({ request }) => {
    // Wrong password for existing user
    const wrongPasswordRes = await request.post('/api/auth/sign-in', {
      data: {
        email: CUSTOMER_EMAIL,
        password: 'wrong-password',
      },
    });

    expect(wrongPasswordRes.status()).toBe(401);
    const wrongPasswordBody = await wrongPasswordRes.json();
    expect(wrongPasswordBody.error).toBe('INVALID_CREDENTIALS');
    expect(wrongPasswordBody.message).toBe('Invalid credentials');

    // Unknown email
    const unknownEmailRes = await request.post('/api/auth/sign-in', {
      data: {
        email: 'nobody@hubx.co',
        password: 'any-password',
      },
    });

    expect(unknownEmailRes.status()).toBe(401);
    const unknownEmailBody = await unknownEmailRes.json();
    expect(unknownEmailBody.error).toBe('INVALID_CREDENTIALS');
    expect(unknownEmailBody.message).toBe('Invalid credentials');

    // Both responses must be identical (same structure, same message)
    expect(wrongPasswordBody).toEqual(unknownEmailBody);
  });

  /**
   * T006: Validation of required fields
   * Scenario: sign in without password
   * Expected: 400 Bad Request with message "Email and password are required"
   */
  test('HU01_validation_required_fields', async ({ request }) => {
    // Missing password
    const missingPasswordRes = await request.post('/api/auth/sign-in', {
      data: {
        email: OPERATOR_EMAIL,
      },
    });

    expect(missingPasswordRes.status()).toBe(400);
    const missingPasswordBody = await missingPasswordRes.json();
    expect(missingPasswordBody.error).toBe('MISSING_REQUIRED_FIELDS');
    expect(missingPasswordBody.message).toBe('Email and password are required');

    // Missing email
    const missingEmailRes = await request.post('/api/auth/sign-in', {
      data: {
        password: DEMO_PASSWORD,
      },
    });

    expect(missingEmailRes.status()).toBe(400);
    const missingEmailBody = await missingEmailRes.json();
    expect(missingEmailBody.error).toBe('MISSING_REQUIRED_FIELDS');
    expect(missingEmailBody.message).toBe('Email and password are required');

    // Both missing
    const bothMissingRes = await request.post('/api/auth/sign-in', {
      data: {},
    });

    expect(bothMissingRes.status()).toBe(400);
    const bothMissingBody = await bothMissingRes.json();
    expect(bothMissingBody.error).toBe('MISSING_REQUIRED_FIELDS');
    expect(bothMissingBody.message).toBe('Email and password are required');
  });

  /**
   * T007: Session expiration
   * Scenario: operator with expired JWT (exp < now) tries to access a protected endpoint
   * Expected: 401 with message "Session expired. Please sign in again."
   * 
   * This test forges an expired JWT using JWT_SECRET from environment.
   * Protected endpoint: GET /api/shipments/HBX-SEED-0001 (future HU-02)
   */
  test('HU01_session_expiration', async ({ request }) => {
    const jwtSecret = process.env.JWT_SECRET;
    
    if (!jwtSecret) {
      throw new Error(
        'JWT_SECRET environment variable not set. ' +
        'Integration tests need JWT_SECRET to forge expired tokens. ' +
        'Run "bash scripts/env.sh" to generate .env, then source it.'
      );
    }

    // Forge expired JWT (exp 1 hour in the past)
    const now = Math.floor(Date.now() / 1000);
    const header = { alg: 'HS256', typ: 'JWT' };
    const payload = {
      sub: '00000000-0000-0000-0000-000000000001', // Demo UUID
      email: OPERATOR_EMAIL,
      role: 'operator',
      iat: now - 7200, // Issued 2 hours ago
      exp: now - 3600, // Expired 1 hour ago
    };

    const encodedHeader = base64urlEncode(JSON.stringify(header));
    const encodedPayload = base64urlEncode(JSON.stringify(payload));
    const signatureInput = `${encodedHeader}.${encodedPayload}`;
    
    const signature = crypto
      .createHmac('sha256', Buffer.from(jwtSecret, 'hex'))
      .update(signatureInput)
      .digest('base64url');

    const expiredToken = `${encodedHeader}.${encodedPayload}.${signature}`;

    // Try to access protected endpoint with expired token
    const res = await request.get('/api/shipments/HBX-SEED-0001', {
      headers: {
        Authorization: `Bearer ${expiredToken}`,
      },
    });

    expect(res.status()).toBe(401);
    const body = await res.json();
    expect(body.error).toBe('UNAUTHORIZED');
    expect(body.message).toBe('Session expired. Please sign in again.');
  });
});

/**
 * Base64url encode helper (JWT spec: RFC 7515)
 */
function base64urlEncode(str: string): string {
  return Buffer.from(str)
    .toString('base64')
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=/g, '');
}
