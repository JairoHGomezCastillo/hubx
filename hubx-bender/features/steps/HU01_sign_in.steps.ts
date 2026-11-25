import { Given, When, Then, defineParameterType } from '@cucumber/cucumber';
import { expect } from '@playwright/test';
import * as crypto from 'crypto';
import type { HubxWorld } from '../support/world.ts';
import { assertError } from './common.steps.ts';

// Demo users (from migration 0001_baseline_schema.sql)
const DEMO_PASSWORD = 'Hubx-Demo-2026';

const USERS = {
  operator: 'operator@hubx.co',
  'ground partner': 'rapido@hubx.co',
  customer: 'ana@hubx.co',
};

// ========== PARAMETER TYPE ==========

// Custom parameter type to capture "ground partner" (two words) as a single value
defineParameterType({
  name: 'role',
  regexp: /operator|ground partner|customer/,
  transformer: (s: string) => s,
});

// ========== UNIFIED STEPS (API or E2E depending on this.page presence) ==========

Given(
  'a user with the role of {role} is registered in HUBX',
  function (this: HubxWorld, role: string) {
    // Demo users are preloaded; store role for later use
    this.role = role;
  },
);

When(
  'that user signs in with their correct email and password',
  async function (this: HubxWorld) {
    const role = this.role!;
    const email = USERS[role as keyof typeof USERS];

    if (this.page) {
      // E2E: browser through the MFE
      await this.ui.goto('/sign-in');
      await this.ui.getByLabel(/email/i).fill(email);
      await this.ui.getByLabel(/password/i).fill(DEMO_PASSWORD);
      await this.ui.getByRole('button', { name: /sign in/i }).click();
    } else {
      // API: request through the gateway
      this.response = await this.api.post('/api/auth/sign-in', {
        data: { email, password: DEMO_PASSWORD },
      });
    }
  },
);

Then(
  'they access HUBX with the functions of their role',
  async function (this: HubxWorld) {
    const role = this.role!;
    const email = USERS[role as keyof typeof USERS];

    if (this.page) {
      // E2E: verify the dashboard shows email and role
      await expect(this.ui).toHaveURL(/\/dashboard$/);
      await expect(this.ui.getByText(email)).toBeVisible();
      await expect(this.ui.getByText(new RegExp(`Role:\\s*${role}`))).toBeVisible();
    } else {
      // API: verify 200, token, user object and JWT structure
      expect(this.response?.status()).toBe(200);
      const body = await this.response?.json();

      expect(body).toHaveProperty('token');
      expect(body).toHaveProperty('user');

      expect(body.user.email).toBe(email);
      expect(body.user.role).toBe(role);

      // Verify JWT structure
      const payload = JSON.parse(Buffer.from(body.token.split('.')[1], 'base64url').toString());
      expect(payload.email).toBe(email);
      expect(payload.role).toBe(role);
      expect(payload).toHaveProperty('sub');
      expect(payload).toHaveProperty('iat');
      expect(payload).toHaveProperty('exp');

      this.token = body.token;
    }
  },
);

// ========== REMAINING API STEPS ==========

Given(
  'a customer is registered with the email {string}',
  function (this: HubxWorld, email: string) {
    // Demo user ana@hubx.co is preloaded; store for later
    this.email = email;
  },
);

When(
  'the customer signs in typing {string} and their correct password',
  async function (this: HubxWorld, typedEmail: string) {
    this.response = await this.api.post('/api/auth/sign-in', {
      data: { email: typedEmail, password: DEMO_PASSWORD },
    });
  },
);

Then('the customer accesses HUBX with the functions of their role', async function (this: HubxWorld) {
  expect(this.response?.status()).toBe(200);
  const body = await this.response?.json();

  expect(body).toHaveProperty('token');
  expect(body).toHaveProperty('user');
  expect(body.user.email).toBe(this.email); // Normalized
  expect(body.user.role).toBe('customer');
});

Given('a customer is registered in HUBX', function (this: HubxWorld) {
  // Demo user ana@hubx.co is preloaded
});

Given('an operator is registered in HUBX', function (this: HubxWorld) {
  // Demo user operator@hubx.co is preloaded
});

When(
  'the customer tries to sign in with an incorrect password',
  async function (this: HubxWorld) {
    this.response = await this.api.post('/api/auth/sign-in', {
      data: { email: USERS.customer, password: 'wrong-password' },
    });
  },
);

Then(
  'they are told the credentials are not valid, without saying which piece of data failed',
  async function (this: HubxWorld) {
    expect(this.response?.status()).toBe(401);
    const body = await this.response?.json();
    expect(body.error).toBe('INVALID_CREDENTIALS');
    expect(body.message).toBe('Invalid credentials');
  },
);

Then('they do not access HUBX', function (this: HubxWorld) {
  // Verified by the 401 status in the previous step
});

Given('no user exists with the email {string}', function (this: HubxWorld, email: string) {
  // Store for later use
  this.email = email;
});

When('a person tries to sign in with that email', async function (this: HubxWorld) {
  this.response = await this.api.post('/api/auth/sign-in', {
    data: { email: this.email, password: 'any-password' },
  });
});

When(
  'the operator tries to sign in without entering their password',
  async function (this: HubxWorld) {
    this.response = await this.api.post('/api/auth/sign-in', {
      data: { email: USERS.operator },
    });
  },
);

Then('they are asked to complete the missing data', async function (this: HubxWorld) {
  expect(this.response?.status()).toBe(400);
  const body = await this.response?.json();
  expect(body.error).toBe('MISSING_REQUIRED_FIELDS');
  expect(body.message).toBe('Email and password are required');
});

Given('an operator signed in more than one hour ago', function (this: HubxWorld) {
  const jwtSecret = process.env.JWT_SECRET;

  if (!jwtSecret) {
    throw new Error(
      'JWT_SECRET environment variable not set. ' +
        'Integration tests need JWT_SECRET to forge expired tokens. ' +
        'Run "bash scripts/env.sh" to generate .env, then source it.',
    );
  }

  // Forge expired JWT (exp 1 hour in the past)
  const now = Math.floor(Date.now() / 1000);
  const header = { alg: 'HS256', typ: 'JWT' };
  const payload = {
    sub: '00000000-0000-0000-0000-000000000001', // Demo UUID
    email: USERS.operator,
    role: 'operator',
    iat: now - 7200, // Issued 2 hours ago
    exp: now - 3600, // Expired 1 hour ago
  };

  const encodedHeader = base64urlEncode(JSON.stringify(header));
  const encodedPayload = base64urlEncode(JSON.stringify(payload));
  const signatureInput = `${encodedHeader}.${encodedPayload}`;

  const signature = crypto.createHmac('sha256', jwtSecret).update(signatureInput).digest('base64url');

  this.token = `${encodedHeader}.${encodedPayload}.${signature}`;
});

When('the operator tries to view a shipment', async function (this: HubxWorld) {
  this.response = await this.api.get('/api/shipments/HBX-SEED-0001', {
    headers: { Authorization: `Bearer ${this.token}` },
  });
});

Then(
  'they are told their session expired and are asked to sign in again',
  async function (this: HubxWorld) {
    expect(this.response?.status()).toBe(401);
    const body = await this.response?.json();
    expect(body.error).toBe('UNAUTHORIZED');
    expect(body.message).toBe('Session expired. Please sign in again.');
  },
);

Then('the lookup is not carried out', function (this: HubxWorld) {
  // Verified by the 401 status in the previous step
});

// ========== HELPER ==========

function base64urlEncode(str: string): string {
  return Buffer.from(str).toString('base64').replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '');
}
