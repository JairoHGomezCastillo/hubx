import { Given, Then, When } from '@cucumber/cucumber';
import { expect } from '@playwright/test';
import {
  assertError,
  assertStageUnchanged,
  DEMO_CREDENTIALS,
  DEMO_PASSWORD,
  groundPartnerId,
  OPERATOR_NAME,
  postUpdate,
  prepareShipment,
  type UpdateWorld,
} from './common.steps.ts';

const ASSIGNED_PARTNER_NAME = 'Transportes Rápido';
const NON_PARTNER_NAME = 'Distribuciones Andina';
const NON_PARTNER_ID = '00000000-0000-4000-8000-000000000005';
const FINGERPRINT = /^[0-9a-f]{64}$/;

// --- Fixtures and identities (API and @e2e scenarios) ----------------------

Given(/^a shipment is in the stage "([^"]+)"$/, async function (this: UpdateWorld, stage: string) {
  if (this.page) {
    // @e2e: provision a fresh shipment through the MFE (HU-02's Create Shipment page).
    // In the @e2e scenario this Given runs BEFORE "the operator has signed in", so we
    // must sign in here ourselves (same browser flow as common.steps.ts, role/label locators).
    const operatorEmail = DEMO_CREDENTIALS.operator;
    await this.ui.goto('/sign-in');
    await this.ui.getByLabel(/email/i).fill(operatorEmail);
    await this.ui.getByLabel(/password/i).fill(DEMO_PASSWORD);
    await this.ui.getByRole('button', { name: /sign in/i }).click();
    await expect(this.ui).toHaveURL(/\/dashboard$/);
    this.actorRole = 'operator';
    
    // Now create the shipment through the MFE.
    await this.ui.getByRole('button', { name: /create shipment/i }).click();
    await expect(this.ui).toHaveURL(/\/shipments\/new$/);
    
    await this.ui.getByLabel(/origin/i).fill('Bogotá');
    await this.ui.getByLabel(/destination/i).fill('Miami');
    await this.ui.getByLabel(/flight/i).fill('AV-999');
    await this.ui.getByLabel(/weight \(kg\)/i).fill('100');
    await this.ui.getByLabel(/customer email/i).fill('andina@hubx.co');
    
    await this.ui.getByRole('button', { name: /create shipment/i }).click();
    
    // The HU-02 MFE shows "Code: HBX-..." on the success page (Dashboard.tsx line ~43: "Code: {createdShipment.code}").
    await expect(this.ui.getByRole('heading', { name: /shipment created/i })).toBeVisible();
    const codeText = await this.ui.getByText(/code:/i).textContent();
    const match = codeText?.match(/HBX-[0-9A-F]{8}/);
    expect(match, 'fresh shipment code from the MFE').toBeTruthy();
    this.shipmentCode = match![0];
    this.currentStage = stage;
    
    // Navigate back to the dashboard where the Record update form is rendered (RecordUpdateContainer).
    // Use history navigation, not this.ui.goto('/dashboard'): the MFE keeps the JWT in memory
    // (AuthContext), so a full page load would drop the session and sign the operator out.
    await this.ui.goBack();
    await expect(this.ui).toHaveURL(/\/dashboard$/);
    return;
  }
  await prepareShipment(this, stage);
});

Given(/^a newly created shipment has no updates recorded$/, async function (this: UpdateWorld) {
  await prepareShipment(this, 'Created');
  expect(this.previousFingerprint).toBeNull();
});

Given(/^a shipment has the update "([^"]+)" recorded$/, async function (this: UpdateWorld, stage: string) {
  await prepareShipment(this, stage);
  expect(this.lastUpdate?.stage).toBe(stage);
  expect(this.lastUpdate?.fingerprint).toMatch(FINGERPRINT);
});

Given(/^the ground partner "([^"]+)" is registered$/, async function (this: UpdateWorld, name: string) {
  this.assignedPartnerId = await groundPartnerId(this, name);
});

// "the customer ... is registered" is defined once in common.steps.ts (shared
// with HU-02): it records the non-partner id used by the assignment error case.

// "<actor> has signed in" is defined once in common.steps.ts and shared with
// HU-02, to avoid an ambiguous definition for "an operator has signed in" and
// "a {role} has signed in".

// --- Recording an update (API and @e2e) -------------------------------------

When(/^(.+) records the update "([^"]+)"$/, async function (this: UpdateWorld, _actor: string, stage: string) {
  this.lastUpdate = undefined;
  if (this.page) {
    // @e2e: drive the MFE's Record update form (RecordUpdate.tsx). Scope to the
    // "Record an update" region to avoid strict-mode violation with HU-04's
    // "View traceability" section (both have a "Shipment code" label).
    const section = this.ui.getByRole('region', { name: 'Record an update' });
    await section.getByLabel(/shipment code/i).fill(this.shipmentCode!);
    await section.getByLabel(/stage/i).selectOption({ label: stage });
    // Ground partner is optional (only for Assigned to partner); leave it empty.
    await section.getByRole('button', { name: /record update/i }).click();
    return;
  }
  const response = await postUpdate(this, this.shipmentCode!, stage);
  expect(response.status(), `record "${stage}" through the gateway`).toBe(201);
});

When(/^(.+) tries to record the update "([^"]+)"$/, async function (this: UpdateWorld, _actor: string, stage: string) {
  this.lastUpdate = undefined;
  await postUpdate(this, this.shipmentCode!, stage);
});

When(/^the operator assigns "([^"]+)" as the party responsible for the delivery$/, async function (
  this: UpdateWorld,
  name: string,
) {
  expect(name).toBe(ASSIGNED_PARTNER_NAME);
  const partnerId = this.assignedPartnerId ?? (await groundPartnerId(this, name));
  const response = await postUpdate(this, this.shipmentCode!, 'Assigned to partner', partnerId);
  expect(response.status(), 'assignment through the gateway').toBe(201);
});

When(/^the operator tries to assign "([^"]+)" as the party responsible for the delivery$/, async function (
  this: UpdateWorld,
  name: string,
) {
  expect(name).toBe(NON_PARTNER_NAME);
  await postUpdate(this, this.shipmentCode!, 'Assigned to partner', this.nonPartnerId ?? NON_PARTNER_ID);
});

When(
  /^the operator tries to record the update "Assigned to partner" without naming the responsible ground partner$/,
  async function (this: UpdateWorld) {
    await postUpdate(this, this.shipmentCode!, 'Assigned to partner', null);
  },
);

// --- Assertions -------------------------------------------------------------

Then(/^the shipment moves to the stage "([^"]+)"$/, async function (this: UpdateWorld, stage: string) {
  if (this.page) {
    // The MFE exposes the outcome in an accessible status region (UI text: English).
    await expect(this.ui.getByRole('status')).toContainText(stage);
    return;
  }
  expect(this.lastUpdate?.stage, 'stage in the 201 update response').toBe(stage);
});

Then(/^the update is recorded with its responsible party and the date$/, async function (this: UpdateWorld) {
  if (this.page) {
    const status = this.ui.getByRole('status');
    await expect(status).toContainText(OPERATOR_NAME);
    await expect(status).toContainText(/\d{4}-\d{2}-\d{2}/);
    return;
  }
  const update = this.lastUpdate;
  expect(update, '201 update response').toBeDefined();
  expect(update!.responsibleParty.role).toBe(this.actorRole);
  expect(update!.responsibleParty.id).toBeTruthy();
  expect(Number.isNaN(Date.parse(update!.recordedAt))).toBe(false);
});

Then(/^the update is recorded with the operator as the responsible party$/, function (this: UpdateWorld) {
  expect(this.lastUpdate?.responsibleParty.role).toBe('operator');
  expect(this.lastUpdate?.responsibleParty.name).toBe(OPERATOR_NAME);
});

Then(/^"([^"]+)" becomes responsible for the delivery$/, function (this: UpdateWorld, name: string) {
  expect(name).toBe(ASSIGNED_PARTNER_NAME);
  expect(this.lastUpdate?.assignedPartner?.name).toBe(name);
  expect(this.lastUpdate?.assignedPartner?.id).toBeTruthy();
});

Then(/^the update becomes the start of the traceability chain$/, function (this: UpdateWorld) {
  expect(this.lastUpdate?.sequence).toBe(1);
  expect(this.lastUpdate?.fingerprint).toMatch(FINGERPRINT);
});

Then(/^it has no previous update$/, function (this: UpdateWorld) {
  expect(this.lastUpdate?.previousFingerprint).toBeNull();
});

Then(
  /^the new update is chained to the previous one through the previous update's fingerprint$/,
  function (this: UpdateWorld) {
    expect(this.previousFingerprint).toMatch(FINGERPRINT);
    expect(this.lastUpdate?.previousFingerprint).toBe(this.previousFingerprint);
    expect(this.lastUpdate?.fingerprint).toMatch(FINGERPRINT);
  },
);

Then(/^they are told that update does not follow the shipment's sequence$/, async function (this: UpdateWorld) {
  await assertError(this.response!, 409, 'OUT_OF_SEQUENCE', "That update does not follow the shipment's sequence.");
});

Then(/^they are told they are not responsible for that update$/, async function (this: UpdateWorld) {
  await assertError(this.response!, 403, 'NOT_RESPONSIBLE', 'You are not responsible for that update.');
});

Then(/^they are told that only a ground partner can be responsible for the delivery$/, async function (
  this: UpdateWorld,
) {
  await assertError(this.response!, 403, 'NOT_GROUND_PARTNER', 'Only a ground partner can be responsible for the delivery.');
});

Then(/^they are told they must name the ground partner responsible for the delivery$/, async function (
  this: UpdateWorld,
) {
  await assertError(
    this.response!,
    400,
    'PARTNER_REQUIRED',
    'You must name the ground partner responsible for the delivery.',
  );
});

// The identical "session expired" text belongs to HU-01; its single step in
// HU01_sign_in.steps.ts branches on hu03ExpiredSession to assert HU-03's nested
// SESSION_EXPIRED body instead of HU-01's flat UNAUTHORIZED body.

Then(/^the shipment keeps its stage$/, async function (this: UpdateWorld) {
  await assertStageUnchanged(this);
});
