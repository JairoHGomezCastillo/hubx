import { Given, Then, When } from '@cucumber/cucumber';
import { expect } from '@playwright/test';
import {
  assertError,
  authHeader,
  DEMO_CREDENTIALS,
  emailForActor,
  OPERATOR_NAME,
  prepareShipment,
  roleForActor,
  signInAs,
  type UpdateWorld,
} from './common.steps.ts';

// HU-04: View a shipment's traceability. API calls through the gateway only.

interface TraceabilityWorld extends UpdateWorld {
  traceabilityResponse?: TraceabilityResponse;
}

interface TraceabilityResponse {
  shipmentCode: string;
  stage: string;
  updates: Array<{
    sequence: number;
    stage: string;
    responsibleParty: { id: string; name: string; role: string };
    assignedPartner: { id: string; name: string } | null;
    recordedAt: string;
    previousFingerprint: string | null;
    fingerprint: string;
  }>;
}

// --- Fixtures --------------------------------------------------------------

Given(/^a shipment has three updates recorded$/, async function (this: TraceabilityWorld) {
  if (this.page) {
    // @e2e: Create the shipment and advance it to "Assigned to partner" (three
    // updates) through the gateway (the @e2e fixture uses the API for setup,
    // not the browser; the scenario's When step drives the MFE).
    await prepareShipment(this, 'Assigned to partner');
    return;
  }
  await prepareShipment(this, 'Assigned to partner');
});

Given(/^a shipment has no updates recorded$/, async function (this: TraceabilityWorld) {
  await prepareShipment(this, 'Created');
});

Given(/^a shipment with updates recorded exists$/, async function (this: TraceabilityWorld) {
  await prepareShipment(this, 'Assigned to partner');
});

// "<actor> has signed in" is shared from common.steps.ts (HU-01, HU-02, HU-03,
// HU-04), it handles both API and @e2e modes.

// --- View traceability -----------------------------------------------------

When(/^(.+) views the shipment's traceability using its code$/, async function (
  this: TraceabilityWorld,
  actor: string,
) {
  if (this.page) {
    // @e2e: drive the MFE's traceability view on Dashboard.tsx (plan: label
    // "Shipment code", button "View traceability", then heading "Traceability of
    // <code>" and the results).
    await this.ui.getByLabel(/shipment code/i).fill(this.shipmentCode!);
    await this.ui.getByRole('button', { name: /view traceability/i }).click();
    // The route may not exist yet; expect a visible element only in Then.
    return;
  }
  const response = await this.api.get(`/api/shipments/${encodeURIComponent(this.shipmentCode!)}/traceability`, {
    headers: authHeader(this),
  });
  this.response = response;
  if (response.status() === 200) {
    this.traceabilityResponse = (await response.json()) as TraceabilityResponse;
  }
});

When(/^the customer views the shipment's traceability$/, async function (this: TraceabilityWorld) {
  // The step "the customer who owns the shipment has signed in" ran before;
  // world.actorRole is 'customer'. Just call the endpoint.
  const response = await this.api.get(`/api/shipments/${encodeURIComponent(this.shipmentCode!)}/traceability`, {
    headers: authHeader(this),
  });
  this.response = response;
  if (response.status() === 200) {
    this.traceabilityResponse = (await response.json()) as TraceabilityResponse;
  }
});

When(/^the operator views the traceability of a code that matches no shipment$/, async function (
  this: TraceabilityWorld,
) {
  const unknownCode = 'HBX-99999999';
  const response = await this.api.get(`/api/shipments/${encodeURIComponent(unknownCode)}/traceability`, {
    headers: authHeader(this),
  });
  this.response = response;
});

When(/^(.+) views the traceability of that shipment$/, async function (this: TraceabilityWorld, actor: string) {
  // The fixture created a shipment owned by one demo customer and assigned to
  // the demo ground partner. This step signs in as the *other* customer or the
  // *other* partner (emailForActor handles that logic).
  const role = roleForActor(actor);
  const email = emailForActor(this, actor);
  await signInAs(this, role, email);
  
  const response = await this.api.get(`/api/shipments/${encodeURIComponent(this.shipmentCode!)}/traceability`, {
    headers: authHeader(this),
  });
  this.response = response;
});

// --- Assertions ------------------------------------------------------------

Then(/^they see the three updates, from the oldest to the newest$/, async function (this: TraceabilityWorld) {
  if (this.page) {
    // The MFE renders a list role='list' name='Updates' with three items.
    const list = this.ui.getByRole('list', { name: /updates/i });
    await expect(list).toBeVisible();
    const items = list.getByRole('listitem');
    await expect(items).toHaveCount(3);
    return;
  }
  expect(this.traceabilityResponse, '200 traceability response').toBeDefined();
  const updates = this.traceabilityResponse!.updates;
  expect(updates.length).toBe(3);
  expect(updates[0]!.sequence).toBe(1);
  expect(updates[1]!.sequence).toBe(2);
  expect(updates[2]!.sequence).toBe(3);
  // Oldest first: sequence is ascending.
  expect(updates[0]!.stage).toBe('In flight');
  expect(updates[1]!.stage).toBe('At destination airport');
  expect(updates[2]!.stage).toBe('Assigned to partner');
});

Then(/^for each one they see its stage, its responsible party and its date$/, async function (
  this: TraceabilityWorld,
) {
  if (this.page) {
    // Each item in the Updates list shows stage, responsible party name and a date.
    const list = this.ui.getByRole('list', { name: /updates/i });
    const items = list.getByRole('listitem');
    
    // Check the first item (In flight by Laura Operator).
    const first = items.nth(0);
    await expect(first).toContainText('In flight');
    await expect(first).toContainText(OPERATOR_NAME);
    await expect(first).toContainText(/\d{4}-\d{2}-\d{2}/); // Date pattern
    return;
  }
  const updates = this.traceabilityResponse!.updates;
  for (const update of updates) {
    expect(update.stage).toBeTruthy();
    expect(update.responsibleParty.name).toBeTruthy();
    expect(update.responsibleParty.role).toBeTruthy();
    expect(update.recordedAt).toMatch(/^\d{4}-\d{2}-\d{2}T/); // RFC3339Nano
    expect(Number.isNaN(Date.parse(update.recordedAt))).toBe(false);
  }
});

Then(/^they see that the shipment is in the stage "([^"]+)"$/, async function (
  this: TraceabilityWorld,
  stage: string,
) {
  expect(this.traceabilityResponse, '200 traceability response').toBeDefined();
  expect(this.traceabilityResponse!.stage).toBe(stage);
});

Then(/^they see that it has no updates yet$/, async function (this: TraceabilityWorld) {
  expect(this.traceabilityResponse, '200 traceability response').toBeDefined();
  expect(this.traceabilityResponse!.updates).toEqual([]);
});

Then(/^they are told that no shipment exists with that code$/, async function (this: TraceabilityWorld) {
  await assertError(
    this.response!,
    404,
    'SHIPMENT_NOT_FOUND',
    'No shipment exists with that code.',
  );
});
