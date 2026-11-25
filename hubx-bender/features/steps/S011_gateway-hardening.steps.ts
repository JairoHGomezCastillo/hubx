import { createHmac, randomUUID } from 'node:crypto';
import { After, Given, Then, When } from '@cucumber/cucumber';
import { expect } from '@playwright/test';
import { request, type APIRequestContext, type APIResponse, type Response as PageResponse } from 'playwright';
import type { HubxWorld } from '../support/world.ts';
import { DEMO_PASSWORD, signInAs } from './common.steps.ts';

// Story 011 steps. Limits are proven against the limited gateway (a real gateway: 5 requests/min per user,
// 3 sign-ins/min per email); everything else uses the main gateway (`world.api`, high limits).
// Every scenario is rerunnable inside the same minute: users are minted with random UUIDs and sign-in emails
// are random, except the one registered email of the existence scenario, which tolerates earlier attempts.

const LIMITED_URL = process.env.BENDER_LIMITED_BASE_URL ?? 'http://localhost:8090';
const USER_LIMIT = 5;
const SIGN_IN_LIMIT = 3;
const PROTECTED_PATH = '/api/assigned-shipments';
const REGISTERED_EMAIL = process.env.BENDER_OTHER_PARTNER_EMAIL ?? 'express@hubx.co';
const APPROVED_ORIGIN = 'http://localhost:5173';
const BAD_ORIGIN = 'https://evil.example';

const RATE_LIMITED_BODY = { error: 'RATE_LIMITED', message: 'Too many requests. Please try again later.' };
const SECURITY_HEADERS: Record<string, string> = {
  'x-content-type-options': 'nosniff',
  'x-frame-options': 'DENY',
  'content-security-policy': "default-src 'none'; frame-ancestors 'none'",
  'referrer-policy': 'no-referrer',
  'strict-transport-security': 'max-age=31536000',
  'cache-control': 'no-store',
};

interface Answer {
  label: string;
  status: number;
  headers: Array<{ name: string; value: string }>;
}

interface PerimeterWorld extends HubxWorld {
  limited?: APIRequestContext;
  userToken?: string;
  userSub?: string;
  responses?: APIResponse[];
  other?: APIResponse;
  email?: string;
  attempt?: APIResponse;
  registered429?: APIResponse;
  unregistered429?: APIResponse;
  answers?: Answer[];
  preflight?: APIResponse;
  plain?: APIResponse;
  browserAnswers?: Answer[];
  browserPending?: Array<Promise<void>>;
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function b64url(input: Buffer | string): string {
  return Buffer.from(input).toString('base64url');
}

// HS256 token like the API issues. `secret` defaults to the real one; another secret forges the session.
function mintToken(sub: string, secret?: string): string {
  const key = secret ?? process.env.JWT_SECRET;
  if (!key) throw new Error('JWT_SECRET is not set: the suite needs it to mint HS256 session tokens');
  const now = Math.floor(Date.now() / 1000);
  const header = b64url(JSON.stringify({ alg: 'HS256', typ: 'JWT' }));
  const payload = b64url(
    JSON.stringify({ sub, email: `${sub}@qa.example`, role: 'ground partner', iat: now, exp: now + 3600 }),
  );
  const signature = createHmac('sha256', key).update(`${header}.${payload}`).digest('base64url');
  return `${header}.${payload}.${signature}`;
}

async function limitedContext(world: PerimeterWorld): Promise<APIRequestContext> {
  world.limited ??= await request.newContext({
    baseURL: LIMITED_URL,
    extraHTTPHeaders: { Accept: 'application/json' },
  });
  return world.limited;
}

function getAs(ctx: APIRequestContext, token: string, headers: Record<string, string> = {}): Promise<APIResponse> {
  return ctx.get(PROTECTED_PATH, { headers: { Authorization: `Bearer ${token}`, ...headers } });
}

async function expectRefusal(response: APIResponse, what: string): Promise<void> {
  expect(response.status(), `${what}: status`).toBe(429);
  expect(response.headers()['content-type'], `${what}: content-type`).toBe('application/json');
  expect(await response.json(), `${what}: body`).toEqual(RATE_LIMITED_BODY);
  expectRetryAfter(response, what);
}

function expectRetryAfter(response: APIResponse, what: string): void {
  const entries = response.headersArray().filter((h) => h.name.toLowerCase() === 'retry-after');
  expect(entries, `${what}: Retry-After present exactly once`).toHaveLength(1);
  const value = entries[0]!.value;
  expect(value, `${what}: Retry-After is whole seconds`).toMatch(/^\d+$/);
  expect(Number(value), `${what}: Retry-After within 1..60`).toBeGreaterThanOrEqual(1);
  expect(Number(value), `${what}: Retry-After within 1..60`).toBeLessThanOrEqual(60);
}

async function answerOf(label: string, response: APIResponse): Promise<Answer> {
  return { label, status: response.status(), headers: response.headersArray() };
}

function expectBrowserProtections(answer: Answer): void {
  for (const [name, value] of Object.entries(SECURITY_HEADERS)) {
    const entries = answer.headers.filter((h) => h.name.toLowerCase() === name);
    expect(entries, `${answer.label} (${answer.status}): ${name} exactly once`).toHaveLength(1);
    expect(entries[0]!.value, `${answer.label} (${answer.status}): ${name} value`).toBe(value);
  }
}

function accessControlHeaders(response: APIResponse): string[] {
  return response
    .headersArray()
    .map((h) => h.name.toLowerCase())
    .filter((name) => name.startsWith('access-control-'));
}

function header(response: APIResponse, name: string): string | undefined {
  const entries = response.headersArray().filter((h) => h.name.toLowerCase() === name.toLowerCase());
  return entries.length === 0 ? undefined : entries.map((h) => h.value).join(', ');
}

// Exhausts a fresh user: USER_LIMIT answered requests, then the refused one.
async function exhaustUser(world: PerimeterWorld): Promise<void> {
  const ctx = await limitedContext(world);
  world.userSub = randomUUID();
  world.userToken = mintToken(world.userSub);
  for (let i = 1; i <= USER_LIMIT; i += 1) {
    const response = await getAs(ctx, world.userToken);
    expect(response.status(), `request ${i} of ${USER_LIMIT} within the limit`).toBe(200);
  }
  await expectRefusal(await getAs(ctx, world.userToken), `request ${USER_LIMIT + 1}`);
}

async function signInAttempt(ctx: APIRequestContext, email: string): Promise<APIResponse> {
  return ctx.post('/api/auth/sign-in', { data: { email, password: 'not-the-password' } });
}

// Attempts with one email until refused; the hard bound tolerates attempts made earlier in the same minute.
async function exhaustEmail(ctx: APIRequestContext, email: string): Promise<APIResponse> {
  for (let i = 1; i <= SIGN_IN_LIMIT + 1; i += 1) {
    const response = await signInAttempt(ctx, email);
    if (response.status() === 429) return response;
    expect(response.status(), `sign-in attempt ${i} of ${email} before the limit`).toBe(401);
  }
  throw new Error(`No 429 after ${SIGN_IN_LIMIT + 1} sign-in attempts of ${email}: sign-in is not limited`);
}

After({ tags: '@needs-limited-gateway' }, async function (this: PerimeterWorld) {
  await this.limited?.dispose();
});

// ---------------------------------------------------------------------------
// Request limit per user
// ---------------------------------------------------------------------------

Given(/^an? (?:ground partner|user) has signed in at the limited entry point$/, async function (this: PerimeterWorld) {
  // The session is minted with the real signing secret and a random sub: no sign-in counter is consumed and the
  // scenario can be rerun within the same minute.
  await limitedContext(this);
  this.userSub = randomUUID();
  this.userToken = mintToken(this.userSub);
});

When(/^they make more requests in one minute than the limit allows$/, async function (this: PerimeterWorld) {
  const ctx = await limitedContext(this);
  this.responses = [];
  for (let i = 1; i <= USER_LIMIT + 1; i += 1) {
    this.responses.push(await getAs(ctx, this.userToken!));
  }
});

Then(/^the requests within the limit are answered$/, function (this: PerimeterWorld) {
  expect(this.responses, 'responses recorded').toHaveLength(USER_LIMIT + 1);
  this.responses!.slice(0, USER_LIMIT).forEach((response, index) => {
    expect(response.status(), `request ${index + 1} of ${USER_LIMIT} within the limit`).toBe(200);
  });
});

Then(/^the request over the limit is refused and they are told when to try again$/, async function (this: PerimeterWorld) {
  await expectRefusal(this.responses![USER_LIMIT]!, `request ${USER_LIMIT + 1}`);
});

// ---------------------------------------------------------------------------
// Isolation between users and forged addresses
// ---------------------------------------------------------------------------

Given(/^a user has exhausted their request limit$/, async function (this: PerimeterWorld) {
  await exhaustUser(this);
});

When(/^a different signed-in user makes a request$/, async function (this: PerimeterWorld) {
  const ctx = await limitedContext(this);
  this.other = await getAs(ctx, mintToken(randomUUID()));
});

Then(/^that request is answered$/, function (this: PerimeterWorld) {
  expect(this.other?.status(), 'another user is not affected').toBe(200);
});

When(/^they repeat the request claiming a different network address$/, async function (this: PerimeterWorld) {
  const ctx = await limitedContext(this);
  this.responses = [];
  for (const forged of [
    { 'X-Forwarded-For': '203.0.113.7' },
    { 'X-Real-IP': '203.0.113.8' },
    { Forwarded: 'for=203.0.113.9' },
  ]) {
    this.responses.push(await getAs(ctx, this.userToken!, forged));
  }
});

Then(/^the request is refused and they are told when to try again$/, async function (this: PerimeterWorld) {
  expect(this.responses, 'forged-address requests recorded').toHaveLength(3);
  for (const [index, response] of this.responses!.entries()) {
    await expectRefusal(response, `forged-address request ${index + 1}`);
  }
});

// ---------------------------------------------------------------------------
// Forged sessions never consume the user's limit
// ---------------------------------------------------------------------------

When(
  /^more requests than the limit arrive with a forged session of that user$/,
  async function (this: PerimeterWorld) {
    const ctx = await limitedContext(this);
    const forged = mintToken(this.userSub!, `not-the-real-secret-${randomUUID()}`);
    this.responses = [];
    for (let i = 1; i <= USER_LIMIT + 3; i += 1) {
      this.responses.push(await getAs(ctx, forged));
    }
  },
);

Then(/^every forged request is refused as not signed in$/, async function (this: PerimeterWorld) {
  expect(this.responses!.length, 'more forged requests than the limit').toBeGreaterThan(USER_LIMIT);
  for (const [index, response] of this.responses!.entries()) {
    expect(response.status(), `forged request ${index + 1}`).toBe(401);
    const body = (await response.json()) as { error?: string };
    expect(body.error, `forged request ${index + 1}: error code`).toBe('UNAUTHORIZED');
  }
});

Then(/^the user's own request is answered$/, async function (this: PerimeterWorld) {
  const own = await getAs(await limitedContext(this), this.userToken!);
  expect(own.status(), "the user's own request after the forged ones").toBe(200);
});

// ---------------------------------------------------------------------------
// Sign-in limit per email
// ---------------------------------------------------------------------------

Given(/^the sign-in attempts of one email have been exhausted$/, async function (this: PerimeterWorld) {
  const ctx = await limitedContext(this);
  this.email = `qa-${randomUUID()}@example.test`;
  for (let i = 1; i <= SIGN_IN_LIMIT; i += 1) {
    const response = await signInAttempt(ctx, this.email);
    expect(response.status(), `sign-in attempt ${i} of ${SIGN_IN_LIMIT} of an unknown email`).toBe(401);
  }
});

When(
  /^another attempt is made with that email in capital letters and with extra spaces$/,
  async function (this: PerimeterWorld) {
    const ctx = await limitedContext(this);
    this.attempt = await signInAttempt(ctx, `  ${this.email!.toUpperCase()}  `);
  },
);

Then(/^the attempt is refused and they are told when to try again$/, async function (this: PerimeterWorld) {
  await expectRefusal(this.attempt!, 'normalised sign-in attempt');
});

When(
  /^the sign-in attempts of a registered email and of an unregistered email are exhausted$/,
  async function (this: PerimeterWorld) {
    const ctx = await limitedContext(this);
    this.registered429 = await exhaustEmail(ctx, REGISTERED_EMAIL);
    this.unregistered429 = await exhaustEmail(ctx, `qa-${randomUUID()}@example.test`);
  },
);

Then(/^both are refused with exactly the same answer$/, async function (this: PerimeterWorld) {
  await expectRefusal(this.registered429!, 'registered email');
  await expectRefusal(this.unregistered429!, 'unregistered email');
  expect(this.registered429!.status(), 'same status').toBe(this.unregistered429!.status());
  expect(await this.registered429!.text(), 'same body').toBe(await this.unregistered429!.text());
  expect(this.registered429!.headers()['content-type'], 'same content-type').toBe(
    this.unregistered429!.headers()['content-type'],
  );
});

// ---------------------------------------------------------------------------
// Health is never limited
// ---------------------------------------------------------------------------

When(/^health is checked more times than the limit allows$/, async function (this: PerimeterWorld) {
  const ctx = await limitedContext(this);
  this.responses = [];
  for (const path of ['/health', '/api/health']) {
    for (let i = 1; i <= USER_LIMIT + 3; i += 1) {
      this.responses.push(await ctx.get(path));
    }
  }
});

Then(/^every check is answered$/, function (this: PerimeterWorld) {
  expect(this.responses!.length, 'more checks than the limit on each health route').toBe(2 * (USER_LIMIT + 3));
  this.responses!.forEach((response, index) => {
    expect(response.status(), `health check ${index + 1}`).toBe(200);
  });
});

// ---------------------------------------------------------------------------
// Browser protections and CORS (main gateway)
// ---------------------------------------------------------------------------

When(/^answers are requested with success, without a session and for health$/, async function (this: PerimeterWorld) {
  await signInAs(this, 'ground partner');
  const success = await this.api.get(PROTECTED_PATH, { headers: { Authorization: `Bearer ${this.token}` } });
  const unauthorized = await this.api.get(PROTECTED_PATH);
  const health = await this.api.get('/health');
  const apiHealth = await this.api.get('/api/health');
  this.answers = [
    await answerOf('success', success),
    await answerOf('without a session', unauthorized),
    await answerOf('/health', health),
    await answerOf('/api/health', apiHealth),
  ];
  expect(this.answers.map((a) => a.status), 'statuses of the four answers').toEqual([200, 401, 200, 200]);
});

Then(/^every answer carries the browser protections$/, function (this: PerimeterWorld) {
  expect(this.answers, 'answers recorded').toHaveLength(4);
  this.answers!.forEach(expectBrowserProtections);
});

When(
  /^a browser on an approved origin asks for permission to call HUBX without a session$/,
  async function (this: PerimeterWorld) {
    this.preflight = await this.api.fetch('/api/shipments/stage-counts', {
      method: 'OPTIONS',
      headers: {
        Origin: APPROVED_ORIGIN,
        'Access-Control-Request-Method': 'GET',
        'Access-Control-Request-Headers': 'authorization',
      },
    });
  },
);

Then(/^the permission is granted for that origin only, without credentials$/, function (this: PerimeterWorld) {
  const response = this.preflight!;
  expect(response.status(), 'preflight status').toBe(204);
  expect(header(response, 'access-control-allow-origin'), 'allowed origin is the caller, not *').toBe(APPROVED_ORIGIN);
  expect(header(response, 'access-control-allow-methods'), 'allowed methods').toBe('GET, POST');
  expect(header(response, 'access-control-allow-headers'), 'allowed headers').toBe('Authorization, Content-Type');
  expect(header(response, 'access-control-allow-credentials'), 'no credentials').toBeUndefined();
  expect(header(response, 'vary') ?? '', 'Vary: Origin').toMatch(/\bOrigin\b/i);
});

When(
  /^a browser on an origin that is not approved asks for permission to call HUBX$/,
  async function (this: PerimeterWorld) {
    this.preflight = await this.api.fetch('/api/shipments/stage-counts', {
      method: 'OPTIONS',
      headers: {
        Origin: BAD_ORIGIN,
        'Access-Control-Request-Method': 'GET',
        'Access-Control-Request-Headers': 'authorization',
      },
    });
    this.plain = await this.api.get('/health', { headers: { Origin: BAD_ORIGIN } });
  },
);

Then(/^no permission is granted$/, function (this: PerimeterWorld) {
  expect(this.preflight!.status(), 'preflight from an origin that is not approved').toBe(403);
  expect(accessControlHeaders(this.preflight!), 'no Access-Control-* on the refused preflight').toEqual([]);
  expect(this.plain!.status(), 'a plain request from that origin is processed').toBe(200);
  expect(accessControlHeaders(this.plain!), 'no Access-Control-* on a plain request from that origin').toEqual([]);
});

// ---------------------------------------------------------------------------
// @e2e: browser only, no direct API call
// ---------------------------------------------------------------------------

Given(/^an operator opens HUBX in a browser$/, async function (this: PerimeterWorld) {
  this.browserAnswers = [];
  this.browserPending = [];
  // headersArray() keeps duplicates, so "exactly once" is checkable; collected for /api/ answers only.
  this.ui.on('response', (response: PageResponse) => {
    if (!new URL(response.url()).pathname.startsWith('/api/')) return;
    this.browserPending!.push(
      response.headersArray().then((headers) => {
        this.browserAnswers!.push({
          label: `${response.request().method()} ${new URL(response.url()).pathname}`,
          status: response.status(),
          headers,
        });
      }),
    );
  });
  await this.ui.goto('/sign-in');
});

When(/^the operator signs in and views the shipment count by stage$/, async function (this: PerimeterWorld) {
  await this.ui.getByLabel(/email/i).fill(process.env.BENDER_OPERATOR_EMAIL ?? 'operator@hubx.co');
  await this.ui.getByLabel(/password/i).fill(DEMO_PASSWORD);
  await this.ui.getByRole('button', { name: /sign in/i }).click();
  await expect(this.ui).toHaveURL(/\/dashboard$/);
  await this.ui.getByRole('link', { name: /shipment count by stage/i }).click();
  await expect(this.ui.getByRole('heading', { name: /shipment count by stage/i })).toBeVisible();
});

Then(/^they see the total for each stage in the browser$/, async function (this: PerimeterWorld) {
  const table = this.ui.getByRole('region', { name: /stage counts/i }).getByRole('table');
  await expect(table).toBeVisible();
  for (const stage of [
    'Created',
    'In flight',
    'At destination airport',
    'Assigned to partner',
    'Out for delivery',
    'Delivered',
  ]) {
    const row = table.getByRole('row', { name: new RegExp(`^${stage}\\b`, 'i') });
    await expect(row, `row of ${stage}`).toBeVisible();
    await expect(row.getByRole('cell').last(), `total of ${stage} is a number`).toHaveText(/^\s*\d+\s*$/);
  }
});

Then(/^the answers their browser received carry the browser protections$/, async function (this: PerimeterWorld) {
  await Promise.all(this.browserPending ?? []);
  const answers = this.browserAnswers ?? [];
  expect(answers.some((a) => a.label.includes('/api/auth/sign-in')), 'the browser received the sign-in answer').toBe(true);
  expect(
    answers.some((a) => a.label.includes('/api/shipments/stage-counts')),
    'the browser received the stage-counts answer',
  ).toBe(true);
  answers.forEach(expectBrowserProtections);
});
