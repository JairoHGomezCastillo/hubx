import { Given, Then, When } from '@cucumber/cucumber';
import { expect } from '@playwright/test';
import {
  authHeader,
  DEMO_CREDENTIALS,
  DEMO_PASSWORD,
  emailForActor,
  prepareShipment,
  roleForActor,
  signInAs,
  type UpdateWorld,
} from './common.steps.ts';

// HU-07 step definitions: View my assigned shipments.
// Two demo ground partners from migration 0001: rapido@hubx.co (Transportes
// Rápido = "the first partner"/"the partner") and express@hubx.co (Logística
// Express = "another ground partner"). The test DB accumulates assignments
// across scenarios, so the Given drains the partner's pending list first through
// the gateway: the partner records Out for delivery and Delivered on each
// pending shipment it lists. This establishes "no shipment has been assigned"
// and "only their three shipments" (the drained ones are now Delivered).

interface AssignedWorld extends UpdateWorld {
  assignedShipments?: Array<{
    code: string;
    stage: string;
    origin: string;
    destination: string;
    flight: string;
    weightKg: number;
    assignedAt: string;
  }>;
  nextOffset?: number | null;
  message?: string | null;
  createdCodes?: string[];
}

const ANOTHER_PARTNER_EMAIL = process.env.BENDER_OTHER_PARTNER_EMAIL ?? 'express@hubx.co';

// Drains the partner's pending list by recording Out for delivery and Delivered
// on each pending shipment. The route does not exist yet (red phase), so the
// drain cannot list; this helper will fail until the route is implemented.
// F5: Always re-request offset=0 until the list is empty (delivering items while
// paging with offset skips items).
async function drainPendingShipments(world: AssignedWorld, partnerEmail: string): Promise<void> {
  await signInAs(world, 'ground partner', partnerEmail);
  
  while (true) {
    const listResponse = await world.api.get('/api/assigned-shipments?offset=0', {
      headers: authHeader(world),
    });
    
    // During red phase the route does not exist: 404 or similar
    if (listResponse.status() !== 200) {
      break;
    }
    
    const body = (await listResponse.json()) as {
      shipments: Array<{ code: string; stage: string }>;
      nextOffset: number | null;
    };
    
    if (body.shipments.length === 0) break;
    
    for (const ship of body.shipments) {
      // Advance each pending shipment to Delivered (Out for delivery, then Delivered)
      if (ship.stage === 'Assigned to partner') {
        const outResp = await world.api.post(`/api/shipments/${encodeURIComponent(ship.code)}/updates`, {
          headers: authHeader(world),
          data: { stage: 'Out for delivery', assignedPartnerId: null },
        });
        expect(outResp.status(), `drain ${ship.code} to Out for delivery`).toBe(201);
      }
      
      const deliveredResp = await world.api.post(`/api/shipments/${encodeURIComponent(ship.code)}/updates`, {
        headers: authHeader(world),
        data: { stage: 'Delivered', assignedPartnerId: null },
      });
      expect(deliveredResp.status(), `drain ${ship.code} to Delivered`).toBe(201);
      
      // F5: 100 ms spacing between assignments
      await new Promise((resolve) => setTimeout(resolve, 100));
    }
  }
}

// Creates N fresh shipments through the gateway (HU-02) and advances them to
// "Assigned to partner" (or further) for the specified ground partner.
// F2: partnerEmail parameter allows assigning to a specific partner.
async function createAssignedShipments(
  world: AssignedWorld,
  count: number,
  partnerEmail: string,
  finalStage: 'Assigned to partner' | 'Out for delivery' | 'Delivered' = 'Assigned to partner',
): Promise<string[]> {
  const codes: string[] = [];
  
  for (let i = 0; i < count; i += 1) {
    // Use prepareShipment from common.steps.ts with the partner parameter
    const targetStage = finalStage === 'Delivered' || finalStage === 'Out for delivery'
      ? 'Assigned to partner'
      : finalStage;
    
    const code = await prepareShipment(world, targetStage, partnerEmail);
    codes.push(code);
    
    // If we need to go further than Assigned to partner, the assigned partner must do it
    if (finalStage === 'Out for delivery' || finalStage === 'Delivered') {
      await signInAs(world, 'ground partner', partnerEmail);
      
      if (finalStage === 'Out for delivery' || finalStage === 'Delivered') {
        const outResp = await world.api.post(`/api/shipments/${encodeURIComponent(code)}/updates`, {
          headers: authHeader(world),
          data: { stage: 'Out for delivery', assignedPartnerId: null },
        });
        expect(outResp.status(), `${code} to Out for delivery`).toBe(201);
      }
      
      if (finalStage === 'Delivered') {
        const deliveredResp = await world.api.post(`/api/shipments/${encodeURIComponent(code)}/updates`, {
          headers: authHeader(world),
          data: { stage: 'Delivered', assignedPartnerId: null },
        });
        expect(deliveredResp.status(), `${code} to Delivered`).toBe(201);
      }
    }
    
    // Add a small delay to ensure distinct assignedAt timestamps for order verification
    await new Promise((resolve) => setTimeout(resolve, 100));
  }
  
  return codes;
}

Given(
  'three shipments pending delivery were assigned to a ground partner',
  async function (this: AssignedWorld) {
    // Drain existing pending shipments first
    await drainPendingShipments(this, DEMO_CREDENTIALS['ground partner']);
    
    // Create three fresh shipments assigned to the first partner
    this.createdCodes = await createAssignedShipments(this, 3, DEMO_CREDENTIALS['ground partner'], 'Assigned to partner');
  },
);

Given(
  'two shipments were assigned to another ground partner',
  async function (this: AssignedWorld) {
    // F2: Use ANOTHER_PARTNER_EMAIL (express@hubx.co) instead of the first partner
    await createAssignedShipments(this, 2, ANOTHER_PARTNER_EMAIL, 'Assigned to partner');
  },
);

// F1: Remove this step definition - use the generic one from common.steps.ts instead
// (removed: /^(the first partner|the partner) has signed in$/)

Given(
  'a ground partner was assigned one shipment that is already delivered and another that is out for delivery',
  async function (this: AssignedWorld) {
    // Drain existing pending shipments first
    await drainPendingShipments(this, DEMO_CREDENTIALS['ground partner']);
    
    // Create one delivered and one out for delivery
    await createAssignedShipments(this, 1, DEMO_CREDENTIALS['ground partner'], 'Delivered');
    this.createdCodes = await createAssignedShipments(this, 1, DEMO_CREDENTIALS['ground partner'], 'Out for delivery');
  },
);

Given(
  'no shipment has been assigned to a ground partner',
  async function (this: AssignedWorld) {
    // Drain all existing pending shipments for the first partner
    await drainPendingShipments(this, DEMO_CREDENTIALS['ground partner']);
    this.createdCodes = [];
  },
);

// F3: Explicit step for the outline When. The actor has already signed in via the
// Given step, so reuse this.token (do not re-sign-in).
When(
  /^(.+) tries to view the shipments assigned to a ground partner$/,
  async function (this: AssignedWorld, _actor: string) {
    // F3: Reuse the token from the Given step, do not sign in again
    this.response = await this.api.get('/api/assigned-shipments', {
      headers: authHeader(this),
    });
    
    if (this.response.status() === 200) {
      const body = (await this.response.json()) as {
        shipments: Array<{
          code: string;
          stage: string;
          origin: string;
          destination: string;
          flight: string;
          weightKg: number;
          assignedAt: string;
        }>;
        nextOffset: number | null;
        message: string | null;
      };
      this.assignedShipments = body.shipments;
      this.nextOffset = body.nextOffset;
      this.message = body.message;
    }
  },
);

When(
  'the partner views their assigned shipments',
  async function (this: AssignedWorld) {
    if (this.page) {
      // @e2e: browser through the MFE
      // The dashboard should already show the assigned shipments section after sign-in
      const region = this.ui.getByRole('region', { name: /my assigned shipments/i });
      await expect(region).toBeVisible();
    } else {
      // API: request through the gateway
      this.response = await this.api.get('/api/assigned-shipments', {
        headers: authHeader(this),
      });
      
      if (this.response.status() === 200) {
        const body = (await this.response.json()) as {
          shipments: Array<{
            code: string;
            stage: string;
            origin: string;
            destination: string;
            flight: string;
            weightKg: number;
            assignedAt: string;
          }>;
          nextOffset: number | null;
          message: string | null;
        };
        this.assignedShipments = body.shipments;
        this.nextOffset = body.nextOffset;
        this.message = body.message;
      }
    }
  },
);

Then(
  'they see only their three shipments',
  async function (this: AssignedWorld) {
    if (this.page) {
      // @e2e: verify the browser shows three shipments in the table
      const region = this.ui.getByRole('region', { name: /my assigned shipments/i });
      const table = region.getByRole('table');
      await expect(table).toBeVisible();
      
      // Count the number of data rows (excluding header)
      const rows = table.getByRole('row');
      const rowCount = await rows.count();
      
      // Header + 3 data rows = 4 total
      expect(rowCount, 'table rows including header').toBe(4);
      
      // Verify the codes are present
      for (const code of this.createdCodes ?? []) {
        await expect(table.getByText(code)).toBeVisible();
      }
    } else {
      // API: verify the response
      expect(this.response?.status(), 'GET /api/assigned-shipments status').toBe(200);
      expect(this.assignedShipments?.length, 'number of shipments').toBe(3);
      
      // Verify these are the shipments we created
      const returnedCodes = this.assignedShipments?.map((s) => s.code) ?? [];
      expect([...returnedCodes].sort()).toEqual([...(this.createdCodes ?? [])].sort());
    }
  },
);

Then(
  'they see first the one that has been assigned the longest',
  async function (this: AssignedWorld) {
    // F4: Assert that the first item is createdCodes[0] (the first one assigned)
    if (this.page) {
      // @e2e: Read the first data row of the table
      const region = this.ui.getByRole('region', { name: /my assigned shipments/i });
      const table = region.getByRole('table');
      
      // Get all rows and find the first data row (skip header)
      const rows = table.getByRole('row');
      const firstDataRow = rows.nth(1); // Index 0 is header, 1 is first data row
      
      // F4: The first row must contain createdCodes[0]
      await expect(firstDataRow).toContainText(this.createdCodes![0]!);
    } else {
      // API: Check that the first item in the response is createdCodes[0]
      expect(this.assignedShipments, 'assigned shipments list').toBeDefined();
      expect(this.assignedShipments!.length, 'at least 1 shipment').toBeGreaterThanOrEqual(1);
      expect(this.assignedShipments![0]!.code, 'first shipment is the one assigned longest ago').toBe(
        this.createdCodes![0],
      );
    }
  },
);

Then(
  'they see only the shipment that is out for delivery',
  function (this: AssignedWorld) {
    expect(this.response?.status(), 'GET /api/assigned-shipments status').toBe(200);
    expect(this.assignedShipments?.length, 'number of shipments').toBe(1);
    expect(this.assignedShipments?.[0]?.stage).toBe('Out for delivery');
    
    // Verify it's the one we created
    const returnedCode = this.assignedShipments?.[0]?.code;
    expect(this.createdCodes).toContain(returnedCode);
  },
);

Then(
  'they see an empty list',
  function (this: AssignedWorld) {
    expect(this.response?.status(), 'GET /api/assigned-shipments status').toBe(200);
    expect(this.assignedShipments?.length, 'number of shipments').toBe(0);
  },
);

Then(
  'they are told that they have no assigned shipments',
  function (this: AssignedWorld) {
    expect(this.message).toBe('You have no assigned shipments');
  },
);

Then(
  'they are told they do not have permission to see that information',
  async function (this: AssignedWorld) {
    expect(this.response?.status(), '403 FORBIDDEN for non-ground-partner').toBe(403);
    const body = (await this.response?.json()) as { error: string; message: string };
    expect(body.error).toBe('FORBIDDEN');
    expect(body.message).toBe('You do not have permission to see that information');
  },
);

// @e2e: The generic "has signed in" step from common.steps.ts handles both API and browser modes.
// The "the partner views their assigned shipments" When step above detects @e2e via this.page.
