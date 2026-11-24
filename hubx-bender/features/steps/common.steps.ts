import { Given, Then, When } from '@cucumber/cucumber';
import { expect } from '@playwright/test';
import type { APIResponse } from 'playwright';
import type { HubxWorld } from '../support/world.ts';

// Reusable steps. Story steps go in features/steps/HUxx_<name>.steps.ts.

declare module '../support/world.ts' {
  interface HubxWorld {
    // Set only by HU-03's spec-derived expired-session step; lets the shared
    // HU-01 assertion distinguish the nested HU-03 error body without an
    // ambiguous duplicate step definition.
    hu03ExpiredSession?: boolean;
  }
}

When('I request {string} from the gateway', async function (this: HubxWorld, path: string) {
  this.response = await this.api.get(path);
});

Then('the response status is {int}', function (this: HubxWorld, status: number) {
  expect(this.response?.status()).toBe(status);
});

Then('the response body is {}', async function (this: HubxWorld, json: string) {
  expect(await this.response?.json()).toEqual(JSON.parse(json));
});

When('I open the web app', async function (this: HubxWorld) {
  await this.ui.goto('/');
});

Then('I see the heading {string}', async function (this: HubxWorld, text: string) {
  await expect(this.ui.getByRole('heading', { name: text })).toBeVisible();
});

// Runs fetch() inside the page: proves the browser -> MFE (nginx) -> gateway -> API path, independent of the UI.
Then(
  'the page can call {string} and gets {}',
  async function (this: HubxWorld, path: string, json: string) {
    const body = await this.ui.evaluate(async (p) => (await fetch(p)).json(), path);
    expect(body).toEqual(JSON.parse(json));
  },
);

// ---------------------------------------------------------------------------
// HU-03 shared helpers. Every HTTP call below uses `world.api`, whose baseURL is
// BENDER_BASE_URL (the gateway). Nothing here talks to hubx-api or PostgreSQL.
// The generic sign-in step below is shared with HU-02: it replaces HU-02's two
// overlapping step definitions so that "an operator has signed in" and
// "a {role} has signed in" have exactly one definition in the whole suite.
// ---------------------------------------------------------------------------

export type Role = 'operator' | 'ground partner' | 'customer';

export interface UpdateResponse {
  id: number;
  shipmentCode: string;
  sequence: number;
  stage: string;
  responsibleParty: { id: string; name: string; role: string };
  assignedPartner: { id: string; name: string } | null;
  recordedAt: string;
  previousFingerprint: string | null;
  fingerprint: string;
}

export interface UpdateWorld extends HubxWorld {
  shipmentCode?: string;
  currentStage?: string;
  actorRole?: Role;
  assignedPartnerId?: string;
  assignedPartnerEmail?: string;
  nonPartnerId?: string;
  previousFingerprint?: string | null;
  lastUpdate?: UpdateResponse;
  uiShipmentCode?: string;
}

// Demo data from migration 0001, not production secrets. Overridable in CI.
export const DEMO_PASSWORD = process.env.BENDER_DEMO_PASSWORD ?? 'Hubx-Demo-2026';
export const DEMO_CREDENTIALS: Record<Role, string> = {
  operator: process.env.BENDER_OPERATOR_EMAIL ?? 'operator@hubx.co',
  'ground partner': process.env.BENDER_PARTNER_EMAIL ?? 'rapido@hubx.co',
  customer: process.env.BENDER_CUSTOMER_EMAIL ?? 'ana@hubx.co',
};
export const OPERATOR_NAME = 'Laura Operator';
const OTHER_CUSTOMER_EMAIL = process.env.BENDER_OTHER_CUSTOMER_EMAIL ?? 'andina@hubx.co';
const OTHER_PARTNER_EMAIL = process.env.BENDER_OTHER_PARTNER_EMAIL ?? 'express@hubx.co';
const ASSIGNED_PARTNER_NAME = 'Transportes Rápido';
const NON_PARTNER_ID = '00000000-0000-4000-8000-000000000005';

// "the customer ... is registered" is used by both HU-02 and HU-03. Keep a
// single definition: HU-02 only needs the preloaded demo user, while HU-03 also
// records the id used by its "assign someone who is not a ground partner" case.
Given(/^the customer "([^"]+)" is registered$/, function (this: UpdateWorld, name: string) {
  expect(name, 'registered demo customer').toBe('Distribuciones Andina');
  this.nonPartnerId = NON_PARTNER_ID;
});

export const UPDATE_STAGES = [
  'In flight',
  'At destination airport',
  'Assigned to partner',
  'Out for delivery',
  'Delivered',
] as const;

const NEXT_STAGE: Record<string, string> = {
  Created: 'In flight',
  'In flight': 'At destination airport',
  'At destination airport': 'Assigned to partner',
  'Assigned to partner': 'Out for delivery',
  'Out for delivery': 'Delivered',
};

// Seed shipments from migration 0001, all in Created before HU-03 runs.
const SEED_CODES = ['HBX-SEED-0001', 'HBX-SEED-0002', 'HBX-SEED-0003', 'HBX-SEED-0004'] as const;
let fixtureIndex = 0;

export function authHeader(world: UpdateWorld): Record<string, string> {
  if (!world.token) throw new Error('No JWT: a "has signed in" step must run first');
  return { Authorization: `Bearer ${world.token}` };
}

export async function signInAs(world: UpdateWorld, role: Role, email = DEMO_CREDENTIALS[role]): Promise<void> {
  const response = await world.api.post('/api/auth/sign-in', {
    data: { email, password: DEMO_PASSWORD },
  });
  expect(response.status(), `HU-01 sign-in for ${email} through the gateway`).toBe(200);
  const body = (await response.json()) as Record<string, unknown>;
  const token = body.token ?? body.accessToken;
  expect(typeof token, `JWT in sign-in response; keys: ${Object.keys(body).join(', ')}`).toBe('string');
  world.token = token as string;
  world.actorRole = role;
}

export async function postUpdate(
  world: UpdateWorld,
  code: string,
  stage: string,
  assignedPartnerId: string | null = null,
): Promise<APIResponse> {
  const response = await world.api.post(`/api/shipments/${encodeURIComponent(code)}/updates`, {
    headers: authHeader(world),
    data: { stage, assignedPartnerId },
  });
  world.response = response;
  if (response.status() === 201) {
    world.lastUpdate = (await response.json()) as UpdateResponse;
  }
  return response;
}

export async function groundPartnerId(world: UpdateWorld, name: string): Promise<string> {
  const response = await world.api.get('/api/ground-partners', { headers: authHeader(world) });
  expect(response.status(), 'GET /api/ground-partners through the gateway').toBe(200);
  const partners = (await response.json()) as Array<{ id: string; name: string }>;
  const partner = partners.find((item) => item.name === name);
  expect(partner, `registered ground partner "${name}"`).toBeDefined();
  return partner!.id;
}

export async function assertError(
  response: APIResponse,
  status: number,
  code: string,
  message: string,
): Promise<void> {
  expect(response.status()).toBe(status);
  const body = (await response.json()) as { error?: { code?: string; message?: string } };
  expect(body.error?.code).toBe(code);
  expect(body.error?.message).toBe(message);
}

export function roleForActor(actor: string): Role {
  const text = actor.toLowerCase();
  if (text.includes('customer')) return 'customer';
  if (text.includes('ground partner')) return 'ground partner';
  if (text.includes('operator')) return 'operator';
  throw new Error(`Cannot map actor "${actor}" to a demo role`);
}

export function emailForActor(world: UpdateWorld, actor: string): string {
  const text = actor.toLowerCase();
  if (text.includes('not the one assigned')) return OTHER_PARTNER_EMAIL;
  if (text.includes('does not own')) {
    const ownedByAndina = world.shipmentCode === 'HBX-SEED-0001' || world.shipmentCode === 'HBX-SEED-0003';
    return ownedByAndina ? DEMO_CREDENTIALS.customer : OTHER_CUSTOMER_EMAIL;
  }
  return DEMO_CREDENTIALS[roleForActor(actor)];
}

// One definition for every "<actor> has signed in" phrasing used by HU-02 and
// HU-03. API scenarios go through the gateway; the HU-02 @e2e scenario uses
// only the browser (role/label locators, no world.api call).
Given(/^(.+) has signed in$/, async function (this: UpdateWorld, actor: string) {
  const role = roleForActor(actor);
  const email = emailForActor(this, actor);
  if (this.page) {
    await this.ui.goto('/sign-in');
    await this.ui.getByLabel(/email/i).fill(email);
    await this.ui.getByLabel(/password/i).fill(DEMO_PASSWORD);
    await this.ui.getByRole('button', { name: /sign in/i }).click();
    await expect(this.ui).toHaveURL(/\/dashboard$/);
    this.actorRole = role;
    return;
  }
  await signInAs(this, role, email);
});

// Prepares a fixture using only HU-03's own endpoint through the gateway: no
// direct DB writes. The seed for this scenario is chosen round-robin.
export async function prepareShipment(world: UpdateWorld, stage: string): Promise<string> {
  const code = SEED_CODES[fixtureIndex % SEED_CODES.length]!;
  fixtureIndex += 1;
  world.shipmentCode = code;
  world.currentStage = stage;
  world.previousFingerprint = null;
  world.lastUpdate = undefined;

  await signInAs(world, 'operator');

  const target = UPDATE_STAGES.indexOf(stage as (typeof UPDATE_STAGES)[number]);
  if (stage !== 'Created' && target < 0) throw new Error(`Unsupported fixture stage "${stage}"`);

  for (let i = 0; i <= target; i += 1) {
    const next = UPDATE_STAGES[i]!;
    const partnerId = next === 'Assigned to partner' ? await groundPartnerId(world, ASSIGNED_PARTNER_NAME) : null;
    const response = await postUpdate(world, code, next, partnerId);
    expect(response.status(), `fixture advance ${code} to ${next} through the gateway`).toBe(201);
    world.previousFingerprint = world.lastUpdate?.fingerprint ?? null;
    if (next === 'Assigned to partner') {
      world.assignedPartnerId = partnerId;
      world.assignedPartnerEmail = DEMO_CREDENTIALS['ground partner'];
    }
  }
  return code;
}

// With no shipment-read endpoint in HU-03, the observable proof that a rejected
// attempt left the projection untouched is that the next legitimate update is
// still accepted. The probe runs after the rejection and uses the gateway only.
export async function assertStageUnchanged(world: UpdateWorld): Promise<void> {
  const stage = world.currentStage;
  expect(stage, 'stage prepared by the Given step').toBeDefined();
  const next = NEXT_STAGE[stage!];
  expect(next, `there is a legitimate update after "${stage}"`).toBeDefined();
  const byPartner = stage === 'Assigned to partner' || stage === 'Out for delivery';

  await signInAs(
    world,
    byPartner ? 'ground partner' : 'operator',
    byPartner ? (world.assignedPartnerEmail ?? DEMO_CREDENTIALS['ground partner']) : DEMO_CREDENTIALS.operator,
  );
  const partnerId = next === 'Assigned to partner'
    ? (world.assignedPartnerId ?? (await groundPartnerId(world, ASSIGNED_PARTNER_NAME)))
    : null;
  const probe = await postUpdate(world, world.shipmentCode!, next!, partnerId);
  expect(probe.status(), `projection still at "${stage}": the next legitimate update must be accepted`).toBe(201);
}
