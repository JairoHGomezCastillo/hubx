import { Given, Then, When } from '@cucumber/cucumber';
import { expect } from '@playwright/test';
import type { HubxWorld } from '../support/world.ts';
import { authHeader, prepareShipment, signInAs, type UpdateWorld } from './common.steps.ts';

interface StageCountWorld extends UpdateWorld {
  stageCountsResponse?: {
    stages: Array<{ stage: string; count: number }>;
    total: number;
  };
  // Baseline counts captured before fixtures are added, to compute deltas.
  baselineCounts?: Map<string, number>;
}

const JOURNEY_ORDER = [
  'Created',
  'In flight',
  'At destination airport',
  'Assigned to partner',
  'Out for delivery',
  'Delivered',
];

// ---------------------------------------------------------------------------
// Fixtures: build the stage distribution through the gateway only.
// ---------------------------------------------------------------------------

Given(
  /^there are (\d+) shipments in flight, (\d+) out for delivery and (\d+) delivered$/,
  async function (this: StageCountWorld, inFlight: string, outForDelivery: string, delivered: string) {
    const inFlightCount = parseInt(inFlight, 10);
    const outForDeliveryCount = parseInt(outForDelivery, 10);
    const deliveredCount = parseInt(delivered, 10);

    // Capture baseline from the real endpoint before adding fixtures (pass 1: shared DB with seed shipments).
    await signInAs(this, 'operator');
    const baseline = await this.api.get('/api/shipments/stage-counts', { headers: authHeader(this) });
    if (baseline.status() !== 200) {
      throw new Error(
        `Baseline request failed with status ${baseline.status()} (endpoint not implemented yet; correct red reason)`,
      );
    }
    const data = (await baseline.json()) as { stages: Array<{ stage: string; count: number }> };
    this.baselineCounts = new Map(data.stages.map((s) => [s.stage, s.count]));

    // Create fresh shipments and advance them to the target stages.
    for (let i = 0; i < inFlightCount; i++) {
      await prepareShipment(this, 'In flight');
    }
    for (let i = 0; i < outForDeliveryCount; i++) {
      await prepareShipment(this, 'Out for delivery');
    }
    for (let i = 0; i < deliveredCount; i++) {
      await prepareShipment(this, 'Delivered');
    }
  },
);

Given(/^no shipment exists in the stage "([^"]+)"$/, async function (this: StageCountWorld, stage: string) {
  // @empty-db pass: after truncation, no shipment exists anywhere. This Given is a documented no-op
  // that relies on the empty pass having truncated the DB. We cannot call the endpoint here to verify
  // the precondition because the endpoint is not implemented yet (red phase).
  // In verify phase, if needed, we can assert the precondition, but only if it does not depend on the
  // endpoint under test being implemented.
  await signInAs(this, 'operator');
});

Given(/^no shipment exists yet$/, async function (this: StageCountWorld) {
  // @empty-db pass: scripts/test-integration.sh runs pass 2 with --tags '@empty-db' after truncating
  // shipments and updates in the isolated hubx-it database (users are kept for sign-in). This Given
  // is a documented no-op that relies on the empty pass having cleared the DB. We do NOT call the
  // endpoint here to compute a baseline, because the endpoint is not implemented yet (red phase).
  await signInAs(this, 'operator');
});

// ---------------------------------------------------------------------------
// Actions: call the endpoint through the gateway.
// ---------------------------------------------------------------------------

When(
  /^the operator views the shipment count by stage$/,
  async function (this: StageCountWorld) {
    // Branch on this.page: if @e2e (browser), navigate through the MFE; otherwise, call the API.
    if (this.page) {
      // E2E: Operator is already on /dashboard (signed in by the Given step).
      // The Dashboard.tsx should offer a "Shipment count by stage" link for operators.
      const link = this.ui.getByRole('link', { name: /shipment count by stage/i });
      await link.click();
      // Wait for the view to load (the stage counts table should appear).
      await expect(this.ui.getByRole('heading', { name: /shipment count by stage/i })).toBeVisible();
    } else {
      // API: call the endpoint through the gateway.
      this.response = await this.api.get('/api/shipments/stage-counts', { headers: authHeader(this) });
      if (this.response.status() === 200) {
        this.stageCountsResponse = (await this.response.json()) as {
          stages: Array<{ stage: string; count: number }>;
          total: number;
        };
      }
    }
  },
);

When(
  /^the (.+) tries to view the shipment count by stage$/,
  async function (this: StageCountWorld, actor: string) {
    this.response = await this.api.get('/api/shipments/stage-counts', { headers: authHeader(this) });
  },
);

// ---------------------------------------------------------------------------
// Assertions: API response shape, journey order, totals, errors.
// ---------------------------------------------------------------------------

Then(/^they see the total for each stage$/, async function (this: StageCountWorld) {
  if (this.page) {
    // E2E: assert the browser shows six rows with stage names and counts.
    // Scope to the stage-counts region to avoid conflicts with other dashboard elements.
    const region = this.ui.getByRole('region', { name: /stage counts/i });
    const table = region.getByRole('table');
    await expect(table).toBeVisible();

    // Check that all six stages appear with their counts.
    for (const stage of JOURNEY_ORDER) {
      const row = table.getByRole('row', { name: new RegExp(stage, 'i') });
      await expect(row).toBeVisible();
      // The count cell should contain a number (asserting it's visible is enough for red phase).
      await expect(row.getByRole('cell').last()).toBeVisible();
    }
    return;
  }

  // API: assert the response contains exactly six stages, each with a count.
  expect(this.response?.status(), 'GET /api/shipments/stage-counts').toBe(200);
  expect(this.stageCountsResponse, 'response body').toBeDefined();
  expect(this.stageCountsResponse!.stages, 'stages array').toHaveLength(6);

  const stages = this.stageCountsResponse!.stages;
  for (const entry of stages) {
    expect(entry.stage, 'stage name').toBeTruthy();
    expect(typeof entry.count, 'count is a number').toBe('number');
    expect(entry.count, 'count is non-negative').toBeGreaterThanOrEqual(0);
  }

  // Verify totals match the fixture (using baseline + delta approach for pass 1 scenarios).
  if (this.baselineCounts) {
    const expectedInFlight = (this.baselineCounts.get('In flight') ?? 0) + 3;
    const expectedOutForDelivery = (this.baselineCounts.get('Out for delivery') ?? 0) + 2;
    const expectedDelivered = (this.baselineCounts.get('Delivered') ?? 0) + 5;

    const actualInFlight = stages.find((s) => s.stage === 'In flight')?.count ?? 0;
    const actualOutForDelivery = stages.find((s) => s.stage === 'Out for delivery')?.count ?? 0;
    const actualDelivered = stages.find((s) => s.stage === 'Delivered')?.count ?? 0;

    expect(actualInFlight, 'In flight count').toBe(expectedInFlight);
    expect(actualOutForDelivery, 'Out for delivery count').toBe(expectedOutForDelivery);
    expect(actualDelivered, 'Delivered count').toBe(expectedDelivered);

    // Other stages should be unchanged by this scenario's fixtures.
    for (const stage of ['Created', 'At destination airport', 'Assigned to partner']) {
      const baselineCount = this.baselineCounts.get(stage) ?? 0;
      const actualCount = stages.find((s) => s.stage === stage)?.count ?? 0;
      expect(actualCount, `${stage} count unchanged`).toBe(baselineCount);
    }
  }
});

Then(/^they see the stages in journey order$/, async function (this: StageCountWorld) {
  if (this.page) {
    // E2E: verify the six stages appear in the table in journey order.
    const region = this.ui.getByRole('region', { name: /stage counts/i });
    const table = region.getByRole('table');
    const rows = await table.getByRole('row').all();
    // Expect at least 7 rows: 1 header + 6 data rows.
    expect(rows.length, 'table rows (header + 6 stages)').toBeGreaterThanOrEqual(7);

    // The first 6 data rows (after the header) should be in journey order.
    for (let i = 0; i < 6; i++) {
      const row = rows[i + 1]; // skip header
      await expect(row!).toContainText(JOURNEY_ORDER[i]!);
    }
    return;
  }

  // API: assert the stages array is in journey order.
  expect(this.stageCountsResponse, 'response body').toBeDefined();
  const stages = this.stageCountsResponse!.stages.map((s) => s.stage);
  expect(stages, 'stages in journey order').toEqual(JOURNEY_ORDER);
});

Then(
  /^the stage "([^"]+)" appears with a total of zero$/,
  async function (this: StageCountWorld, stage: string) {
    // @empty-db pass 2: absolute assertion (no baseline), the DB is empty.
    expect(this.response?.status(), 'GET /api/shipments/stage-counts').toBe(200);
    expect(this.stageCountsResponse, 'response body').toBeDefined();

    const entry = this.stageCountsResponse!.stages.find((s) => s.stage === stage);
    expect(entry, `stage "${stage}" in response`).toBeDefined();
    expect(entry!.count, `count for "${stage}"`).toBe(0);
  },
);

Then(
  /^they see the six stages, all with a total of zero$/,
  async function (this: StageCountWorld) {
    // @empty-db pass 2: absolute assertion (no baseline), the DB is empty.
    expect(this.response?.status(), 'GET /api/shipments/stage-counts').toBe(200);
    expect(this.stageCountsResponse, 'response body').toBeDefined();
    expect(this.stageCountsResponse!.stages, 'six stages').toHaveLength(6);

    for (const entry of this.stageCountsResponse!.stages) {
      expect(entry.count, `count for "${entry.stage}" in empty system`).toBe(0);
    }
    expect(this.stageCountsResponse!.total, 'total in empty system').toBe(0);
  },
);

Then(
  /^they are told they do not have permission to see that information$/,
  async function (this: StageCountWorld) {
    expect(this.response?.status(), 'non-operator forbidden').toBe(403);
    const body = (await this.response!.json()) as { error?: string; message?: string };
    expect(body.error, 'error code').toBe('FORBIDDEN');
    expect(body.message, 'error message').toBe('You do not have permission to see that information');
    // Verify no totals are exposed in the body.
    expect(body).not.toHaveProperty('stages');
    expect(body).not.toHaveProperty('total');
  },
);
