import { Given, When, Then } from '@cucumber/cucumber';
import { expect } from '@playwright/test';
import {
  authHeader,
  prepareShipment,
  roleForActor,
  emailForActor,
  signInAs,
  type UpdateWorld,
  DEMO_CREDENTIALS,
  DEMO_PASSWORD,
} from './common.steps.ts';
import { tamperUpdate } from '../support/tamper.ts';

interface VerificationWorld extends UpdateWorld {
  verificationResult?: {
    shipmentCode: string;
    intact: boolean;
    updatesChecked: number;
    firstAlteredUpdate: number | null;
  };
}

// --- Fixtures (shipments with and without alterations) ---------------------

Given(/^a shipment has three updates recorded without alterations$/, async function (this: VerificationWorld) {
  // Prepare a fresh shipment in Assigned to partner (3 updates: In flight, At destination airport, Assigned to partner).
  await prepareShipment(this, 'Assigned to partner');
  expect(this.shipmentCode, 'fixture shipment with 3 updates').toMatch(/^HBX-[0-9A-F]{8}$/);
});

// "a shipment has no updates recorded" is now shared in common.steps.ts (also used by HU-04).

Given(/^the second of a shipment's three updates was improperly altered$/, async function (this: VerificationWorld) {
  // Prepare a fresh shipment with 3 updates, then tamper the second one (sequence=2).
  await prepareShipment(this, 'Assigned to partner');
  expect(this.shipmentCode, 'fixture shipment before tampering').toMatch(/^HBX-[0-9A-F]{8}$/);
  
  // Tamper: change the stage of the second update (the tampering bypasses the immutability trigger on purpose).
  await tamperUpdate(this.shipmentCode!, 2, 'Out for delivery');
});

Given(/^a shipment that belongs to another customer exists$/, async function (this: VerificationWorld) {
  // Prepare a shipment owned by one demo customer; the step "a customer who does not own it has signed in"
  // will sign in as the other demo customer (emailForActor handles this).
  await prepareShipment(this, 'Assigned to partner');
  expect(this.shipmentCode, 'fixture shipment for a different customer').toMatch(/^HBX-[0-9A-F]{8}$/);
});

// "<participant> has signed in" is defined once in common.steps.ts and shared
// with HU-02 and HU-03; it uses emailForActor to choose the right demo user.

// --- Verification requests (API and @e2e) -----------------------------------

When(/^(.+) requests verification of (?:the|that) shipment's integrity$/, async function (
  this: VerificationWorld,
  _participant: string,
) {
  const code = this.shipmentCode!;
  this.response = await this.api.get(`/api/shipments/${encodeURIComponent(code)}/integrity`, {
    headers: authHeader(this),
  });
  if (this.response.status() === 200) {
    this.verificationResult = (await this.response.json()) as typeof this.verificationResult;
  }
});

When(/^(.+) requests verification of a code that matches no shipment$/, async function (
  this: VerificationWorld,
  _participant: string,
) {
  const unknownCode = 'HBX-99999999';
  this.response = await this.api.get(`/api/shipments/${encodeURIComponent(unknownCode)}/integrity`, {
    headers: authHeader(this),
  });
});

When(/^the operator verifies the shipment's integrity in the browser$/, async function (this: VerificationWorld) {
  // @e2e: the MFE's Verify Integrity view has an input labelled 'Shipment code',
  // a button 'Verify integrity', and shows the result as text.
  const code = this.shipmentCode!;
  
  // Navigate to the Verify Integrity page (the route is /verify-integrity, linked from Dashboard).
  await this.ui.getByRole('link', { name: /verify integrity/i }).click();
  await expect(this.ui).toHaveURL(/\/verify-integrity$/);
  
  await this.ui.getByLabel(/shipment code/i).fill(code);
  await this.ui.getByRole('button', { name: /verify integrity/i }).click();
  
  // The result appears in a region marked with role="status" or as visible text; we'll check both.
});

// --- Assertions -------------------------------------------------------------

Then(/^they are told that the chain is intact$/, async function (this: VerificationWorld) {
  expect(this.response!.status(), 'verification response status').toBe(200);
  expect(this.verificationResult?.intact, 'chain is intact').toBe(true);
  expect(this.verificationResult?.firstAlteredUpdate, 'no altered update').toBeNull();
});

Then(/^that three updates were checked$/, async function (this: VerificationWorld) {
  expect(this.verificationResult?.updatesChecked, 'updates checked').toBe(3);
});

Then(/^that there are no updates to check$/, async function (this: VerificationWorld) {
  expect(this.verificationResult?.updatesChecked, 'updates checked').toBe(0);
});

Then(/^they are told that the chain is not intact$/, async function (this: VerificationWorld) {
  expect(this.response!.status(), 'verification response status').toBe(200);
  expect(this.verificationResult?.intact, 'chain is not intact').toBe(false);
});

Then(/^that the first alteration is in the second update$/, async function (this: VerificationWorld) {
  expect(this.verificationResult?.firstAlteredUpdate, 'first altered update position').toBe(2);
});

// "they are told that no shipment exists with that code" is now shared in common.steps.ts
// (HU-04 and HU-05 use the same Gherkin text but assert different contracts based on the URL).

// --- @e2e assertions ---------------------------------------------------------

Then(/^the browser shows that the chain is intact$/, async function (this: VerificationWorld) {
  // The MFE renders "The chain is intact" as visible text.
  await expect(this.ui.getByText(/the chain is intact/i)).toBeVisible();
});

Then(/^the browser shows that three updates were checked$/, async function (this: VerificationWorld) {
  // The MFE renders "3 updates checked" or similar.
  await expect(this.ui.getByText(/3 updates checked/i)).toBeVisible();
});
