import { Given, When, Then } from '@cucumber/cucumber';
import { expect } from '@playwright/test';
import type { HubxWorld } from '../support/world.ts';

// Demo users (from migration 0001_baseline_schema.sql)
const USERS = {
  operator: 'operator@hubx.co',
  'ground partner': 'rapido@hubx.co',
  customer: 'ana@hubx.co',
};

const CUSTOMERS = {
  'Distribuciones Andina': 'andina@hubx.co',
};

// ========== STATE ==========

interface ShipmentData {
  origin?: string;
  destination?: string;
  flight?: string;
  weightKg?: number;
  customerEmail?: string;
}

// Extend the world to hold HU-02 state
declare module '../support/world.ts' {
  interface HubxWorld {
    shipmentData?: ShipmentData;
    createdShipmentCode?: string;
    createdShipments?: Array<{ code: string; flight: string }>;
  }
}

// ========== GIVEN STEPS ==========

// "an operator has signed in" and "a {role} has signed in" are defined once in
// common.steps.ts so HU-02 and HU-03 share a single non-ambiguous definition.

// "the customer ... is registered" is defined once in common.steps.ts (shared
// with HU-03), so HU-02 and HU-03 do not register an ambiguous step.

Given('a shipment already exists on flight {string}', async function (this: HubxWorld, flight: string) {
  // Create a shipment on the specified flight
  this.response = await this.api.post('/api/shipments', {
    headers: { Authorization: `Bearer ${this.token}` },
    data: {
      origin: 'Bogotá',
      destination: 'Miami',
      flight,
      weightKg: 100,
      customerEmail: CUSTOMERS['Distribuciones Andina'],
    },
  });
  expect(this.response.status()).toBe(201);
  const body = await this.response.json();
  this.createdShipments = [{ code: body.code, flight: body.flight }];
});

// ========== WHEN STEPS ==========

When(
  'the operator creates a shipment from {word} to {word} on flight {string}, weighing {float} kg, for that customer',
  async function (this: HubxWorld, origin: string, destination: string, flight: string, weightKg: number) {
    const data = {
      origin,
      destination,
      flight,
      weightKg,
      customerEmail: CUSTOMERS['Distribuciones Andina'],
    };

    if (this.page) {
      // E2E: use the MFE form
      await this.ui.getByRole('button', { name: /create shipment/i }).click();
      await expect(this.ui).toHaveURL(/\/shipments\/new$/);
      
      await this.ui.getByLabel(/origin/i).fill(origin);
      await this.ui.getByLabel(/destination/i).fill(destination);
      await this.ui.getByLabel(/flight/i).fill(flight);
      await this.ui.getByLabel(/weight \(kg\)/i).fill(weightKg.toString());
      await this.ui.getByLabel(/customer email/i).fill(data.customerEmail);
      
      await this.ui.getByRole('button', { name: /create shipment/i }).click();
    } else {
      // API: direct POST to the gateway
      this.response = await this.api.post('/api/shipments', {
        headers: { Authorization: `Bearer ${this.token}` },
        data,
      });
    }
  },
);

When(
  'the operator creates a shipment weighing {float} kg',
  async function (this: HubxWorld, weightKg: number) {
    this.response = await this.api.post('/api/shipments', {
      headers: { Authorization: `Bearer ${this.token}` },
      data: {
        origin: 'Bogotá',
        destination: 'Miami',
        flight: 'AV-100',
        weightKg,
        customerEmail: CUSTOMERS['Distribuciones Andina'],
      },
    });
  },
);

When(
  'the operator creates another shipment on flight {string}',
  async function (this: HubxWorld, flight: string) {
    this.response = await this.api.post('/api/shipments', {
      headers: { Authorization: `Bearer ${this.token}` },
      data: {
        origin: 'Bogotá',
        destination: 'Miami',
        flight,
        weightKg: 150,
        customerEmail: CUSTOMERS['Distribuciones Andina'],
      },
    });
  },
);

When(
  'the operator tries to create a shipment with the same origin and destination',
  async function (this: HubxWorld) {
    this.response = await this.api.post('/api/shipments', {
      headers: { Authorization: `Bearer ${this.token}` },
      data: {
        origin: 'Bogotá',
        destination: 'Bogotá',
        flight: 'AV-100',
        weightKg: 100,
        customerEmail: CUSTOMERS['Distribuciones Andina'],
      },
    });
  },
);

When(
  'the operator tries to create a shipment with a weight of {int} kg',
  async function (this: HubxWorld, weight: number) {
    this.response = await this.api.post('/api/shipments', {
      headers: { Authorization: `Bearer ${this.token}` },
      data: {
        origin: 'Bogotá',
        destination: 'Miami',
        flight: 'AV-100',
        weightKg: weight,
        customerEmail: CUSTOMERS['Distribuciones Andina'],
      },
    });
  },
);

When(
  'the operator tries to create a shipment with a customer who is not registered',
  async function (this: HubxWorld) {
    this.response = await this.api.post('/api/shipments', {
      headers: { Authorization: `Bearer ${this.token}` },
      data: {
        origin: 'Bogotá',
        destination: 'Miami',
        flight: 'AV-100',
        weightKg: 100,
        customerEmail: 'nobody@hubx.co',
      },
    });
  },
);

When(
  'the operator tries to create a shipment with a ground partner in the place of the customer',
  async function (this: HubxWorld) {
    this.response = await this.api.post('/api/shipments', {
      headers: { Authorization: `Bearer ${this.token}` },
      data: {
        origin: 'Bogotá',
        destination: 'Miami',
        flight: 'AV-100',
        weightKg: 100,
        customerEmail: USERS['ground partner'],
      },
    });
  },
);

When(
  'the operator tries to create a shipment with the flight left out',
  async function (this: HubxWorld) {
    this.response = await this.api.post('/api/shipments', {
      headers: { Authorization: `Bearer ${this.token}` },
      data: {
        origin: 'Bogotá',
        destination: 'Miami',
        weightKg: 100,
        customerEmail: CUSTOMERS['Distribuciones Andina'],
        // flight intentionally missing
      },
    });
  },
);

When(
  'the {role} tries to create a shipment',
  async function (this: HubxWorld, role: string) {
    this.response = await this.api.post('/api/shipments', {
      headers: { Authorization: `Bearer ${this.token}` },
      data: {
        origin: 'Bogotá',
        destination: 'Miami',
        flight: 'AV-100',
        weightKg: 100,
        customerEmail: CUSTOMERS['Distribuciones Andina'],
      },
    });
  },
);

// ========== THEN STEPS ==========

Then('the shipment is created with a unique code', async function (this: HubxWorld) {
  if (this.page) {
    // E2E: verify success page
    await expect(this.ui.getByRole('heading', { name: /shipment created/i })).toBeVisible();
    
    // Extract code from text "Code: HBX-..."
    const codeText = await this.ui.getByText(/code:/i).textContent();
    expect(codeText).toMatch(/Code:\s*HBX-[0-9A-F]{8}/);
    
    // Store the code for the stage check
    const match = codeText?.match(/HBX-[0-9A-F]{8}/);
    this.createdShipmentCode = match?.[0];
  } else {
    // API: verify 201 and code format
    expect(this.response?.status()).toBe(201);
    const body = await this.response?.json();
    
    expect(body.code).toMatch(/^HBX-[0-9A-F]{8}$/);
    this.createdShipmentCode = body.code;
  }
});

Then('its stage is {string}', async function (this: HubxWorld, expectedStage: string) {
  if (this.page) {
    // E2E: verify stage text on the page
    await expect(this.ui.getByText(new RegExp(`Stage:\\s*${expectedStage}`))).toBeVisible();
  } else {
    // API: verify stage in response body
    const body = await this.response?.json();
    expect(body.stage).toBe(expectedStage);
  }
});

Then('the shipment is created in the stage {string}', async function (this: HubxWorld, expectedStage: string) {
  expect(this.response?.status()).toBe(201);
  const body = await this.response?.json();
  expect(body.stage).toBe(expectedStage);
});

Then(
  'two shipments exist on that flight, each with its own code',
  async function (this: HubxWorld) {
    expect(this.response?.status()).toBe(201);
    const body = await this.response?.json();
    
    // Both shipments should have the same flight
    expect(body.flight).toBe('AV-204');
    
    // The new code should be different from the first one
    expect(body.code).toMatch(/^HBX-[0-9A-F]{8}$/);
    expect(this.createdShipments).toBeDefined();
    expect(body.code).not.toBe(this.createdShipments![0].code);
    
    // Store both codes
    this.createdShipments!.push({ code: body.code, flight: body.flight });
  },
);

Then('the shipment is not created', async function (this: HubxWorld) {
  // Not 201 and no code in body
  expect(this.response?.status()).not.toBe(201);
  const body = await this.response?.json();
  expect(body.code).toBeUndefined();
});

Then(
  'they are told that the origin and the destination must be different',
  async function (this: HubxWorld) {
    expect(this.response?.status()).toBe(400);
    const body = await this.response?.json();
    expect(body.error).toBe('SAME_ORIGIN_DESTINATION');
    expect(body.message).toBe('The origin and the destination must be different');
  },
);

Then(
  'they are told that the weight must be between {float} and {int} kg',
  async function (this: HubxWorld, _min: number, _max: number) {
    expect(this.response?.status()).toBe(400);
    const body = await this.response?.json();
    expect(body.error).toBe('WEIGHT_OUT_OF_RANGE');
    expect(body.message).toBe('The weight must be between 0.1 and 5000 kg');
  },
);

Then(
  'they are told that the customer indicated does not exist',
  async function (this: HubxWorld) {
    expect(this.response?.status()).toBe(422);
    const body = await this.response?.json();
    expect(body.error).toBe('CUSTOMER_NOT_FOUND');
    expect(body.message).toBe('The customer indicated does not exist');
  },
);

Then(
  'they are told that the customer indicated does not have the customer role',
  async function (this: HubxWorld) {
    expect(this.response?.status()).toBe(422);
    const body = await this.response?.json();
    expect(body.error).toBe('CUSTOMER_NOT_CUSTOMER_ROLE');
    expect(body.message).toBe('The customer indicated does not have the customer role');
  },
);

Then('they are told that required data is missing', async function (this: HubxWorld) {
  expect(this.response?.status()).toBe(400);
  const body = await this.response?.json();
  expect(body.error).toBe('MISSING_REQUIRED_FIELDS');
  expect(body.message).toBe('Required data is missing');
});

Then(
  'they are told they do not have permission to create shipments',
  async function (this: HubxWorld) {
    expect(this.response?.status()).toBe(403);
    const body = await this.response?.json();
    expect(body.error).toBe('FORBIDDEN');
    expect(body.message).toBe('You do not have permission to create shipments');
  },
);
